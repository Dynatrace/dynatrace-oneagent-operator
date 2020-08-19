package oneagent

import (
	"context"
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/controller/utils"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileOneAgent) reconcileVersion(logger logr.Logger, instance dynatracev1alpha1.BaseOneAgentDaemonSet, dtc dtclient.Client) (bool, error) {
	if instance.GetOneAgentSpec().UseImmutableImage {
		return r.reconcileVersionImmutableImage(instance, dtc)
	} else {
		return r.reconcileVersionInstaller(logger, instance, dtc)
	}
}

func (r *ReconcileOneAgent) reconcileVersionInstaller(logger logr.Logger, instance dynatracev1alpha1.BaseOneAgentDaemonSet, dtc dtclient.Client) (bool, error) {
	updateCR := false

	desired, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return false, fmt.Errorf("failed to get desired version: %w", err)
	} else if desired != "" && instance.GetOneAgentStatus().Version != desired {
		logger.Info("new version available", "actual", instance.GetOneAgentStatus().Version, "desired", desired)
		instance.GetOneAgentStatus().Version = desired
		updateCR = true
	}

	podList, err := r.findPods(instance)
	if err != nil {
		logger.Error(err, "failed to list pods", "podList", podList)
		return updateCR, err
	}

	podsToDelete, err := findOutdatedPodsInstaller(podList, dtc, instance)
	if err != nil {
		return updateCR, err
	}

	var waitSecs uint16 = 300
	if instance.GetOneAgentSpec().WaitReadySeconds != nil {
		waitSecs = *instance.GetOneAgentSpec().WaitReadySeconds
	}

	if len(podsToDelete) > 0 {
		if instance.GetOneAgentStatus().SetPhase(dynatracev1alpha1.Deploying) {
			err := r.updateCR(instance)
			if err != nil {
				logger.Error(err, fmt.Sprintf("failed to set phase to %s", dynatracev1alpha1.Deploying))
			}
		}
	}

	// restart daemonset
	err = r.deletePods(logger, podsToDelete, buildLabels(instance.GetName()), waitSecs)
	if err != nil {
		logger.Error(err, "failed to update version")
		return updateCR, err
	}

	return updateCR, nil
}

func (r *ReconcileOneAgent) reconcileVersionImmutableImage(instance dynatracev1alpha1.BaseOneAgentDaemonSet, dtc dtclient.Client) (bool, error) {
	updateCR := false
	var waitSecs uint16 = 300
	if instance.GetOneAgentSpec().WaitReadySeconds != nil {
		waitSecs = *instance.GetOneAgentSpec().WaitReadySeconds
	}

	if !instance.GetOneAgentSpec().DisableAgentUpdate {
		r.logger.Info("checking for outdated pods")
		// Check if pods have latest agent version
		outdatedPods, err := r.findOutdatedPodsImmutableImage(r.logger, instance, isLatest)
		if len(outdatedPods) > 0 {
			updateCR = true
		}
		if err != nil {
			return updateCR, err
		}
		if len(outdatedPods) > 0 {
			updateCR = true
			err = r.deletePods(r.logger, outdatedPods, buildLabels(instance.GetName()), waitSecs)
			if err != nil {
				r.logger.Error(err, err.Error())
				return updateCR, err
			}
			instance.GetOneAgentStatus().UpdatedTimestamp = metav1.Now()

			err = r.setVersionByIP(instance, dtc)
			if err != nil {
				r.logger.Error(err, err.Error())
				return updateCR, err
			}
		}

	} else if instance.GetOneAgentSpec().DisableAgentUpdate {
		r.logger.Info("Skipping updating pods because of configuration", "disableOneAgentUpdate", true)
	}
	return updateCR, nil
}

// findOutdatedPodsInstaller determines if a pod needs to be restarted in order to get the desired agent version
// Returns an array of pods and an array of OneAgentInstance objects for status update
func findOutdatedPodsInstaller(pods []corev1.Pod, dtc dtclient.Client, instance dynatracev1alpha1.BaseOneAgentDaemonSet) ([]corev1.Pod, error) {
	var doomedPods []corev1.Pod

	for _, pod := range pods {
		ver, err := dtc.GetAgentVersionForIP(pod.Status.HostIP)
		if err != nil {
			err = handleAgentVersionForIPError(err, instance, pod, nil)
			if err != nil {
				return doomedPods, err
			}
		} else {
			if ver != instance.GetOneAgentStatus().Version {
				doomedPods = append(doomedPods, pod)
			}
		}
	}

	return doomedPods, nil
}

func (r *ReconcileOneAgent) findOutdatedPodsImmutableImage(logger logr.Logger, instance dynatracev1alpha1.BaseOneAgentDaemonSet, isLatestFn func(logger logr.Logger, image string, imageID string, imagePullSecret *corev1.Secret) (bool, error)) ([]corev1.Pod, error) {
	pods, err := r.findPods(instance)
	if err != nil {
		logger.Error(err, "failed to list pods")
		return nil, err
	}

	var outdatedPods []corev1.Pod
	for _, pod := range pods {
		for _, status := range pod.Status.ContainerStatuses {
			if status.Image == "" || status.ImageID == "" {
				// If image is not yet pulled skip check
				continue
			}
			logger.Info("pods container status", "pod", pod.Name, "container", status.Name, "image id", status.ImageID)

			imagePullSecret := &corev1.Secret{}
			pullSecretName := instance.GetName() + "-pull-secret"
			if instance.GetOneAgentSpec().CustomPullSecret != "" {
				pullSecretName = instance.GetOneAgentSpec().CustomPullSecret
			}
			err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: pod.Namespace, Name: pullSecretName}, imagePullSecret)
			if err != nil {
				logger.Error(err, err.Error())
			}

			isLatest, err := isLatestFn(logger, status.Image, status.ImageID, imagePullSecret)
			if err != nil {
				logger.Info(err.Error())
				//Error during image check, do nothing an continue with next status
				continue
			}

			if !isLatest {
				logger.Info("pod is outdated", "name", pod.Name)
				outdatedPods = append(outdatedPods, pod)
				// Pod is outdated, break loop
				break
			}
		}
	}

	return outdatedPods, err
}

func isLatest(logger logr.Logger, image string, imageID string, imagePullSecret *corev1.Secret) (bool, error) {
	dockerConfig, err := utils.NewDockerConfig(imagePullSecret)
	if err != nil {
		logger.Info(err.Error())
	}

	dockerVersionChecker := utils.NewDockerVersionChecker(image, imageID, dockerConfig)
	return dockerVersionChecker.IsLatest()
}

func (r *ReconcileOneAgent) findPods(instance dynatracev1alpha1.BaseOneAgentDaemonSet) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(buildLabels(instance.GetName())),
	}
	err := r.client.List(context.TODO(), podList, listOptions...)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func (r *ReconcileOneAgent) setVersionByIP(instance dynatracev1alpha1.BaseOneAgentDaemonSet, dtc dtclient.Client) error {
	pods, err := r.findPods(instance)
	if err != nil {
		return err
	}
	for _, pod := range pods {
		ver, err := dtc.GetAgentVersionForIP(pod.Status.HostIP)
		if err != nil {
			return err
		}
		instance.GetOneAgentStatus().Version = ver
	}
	return nil
}
