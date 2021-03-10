package oneagent

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileOneAgent) reconcileVersion(ctx context.Context, logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) (bool, error) {
	if !instance.GetOneAgentStatus().UseImmutableImage {
		return r.reconcileVersionInstaller(ctx, logger, instance, dtc)
	}
	return false, nil
}

func (r *ReconcileOneAgent) reconcileVersionInstaller(ctx context.Context, logger logr.Logger, instance *dynatracev1alpha1.OneAgent, dtc dtclient.Client) (bool, error) {
	updateCR := false

	desired, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypeDefault)
	if err != nil {
		return false, fmt.Errorf("failed to get desired version: %w", err)
	} else if desired != "" && desired != instance.Status.Version {
		instance.Status.Version = desired
		updateCR = true
		if isDesiredNewer(instance.Status.Version, desired, logger) {
			logger.Info("new version available", "actual", instance.Status.Version, "desired", desired)
		}
	}

	podList, err := r.findPods(ctx, instance)
	if err != nil {
		logger.Error(err, "failed to list pods", "podList", podList)
		return updateCR, err
	}

	podsToDelete, err := findOutdatedPodsInstaller(podList, dtc, instance, logger)
	if err != nil {
		return updateCR, err
	}

	var waitSecs uint16 = 300
	if instance.GetOneAgentSpec().WaitReadySeconds != nil {
		waitSecs = *instance.GetOneAgentSpec().WaitReadySeconds
	}

	if len(podsToDelete) > 0 {
		if instance.GetOneAgentStatus().SetPhase(dynatracev1alpha1.Deploying) {
			err := r.updateCR(ctx, instance)
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

// findOutdatedPodsInstaller determines if a pod needs to be restarted in order to get the desired agent version
// Returns an array of pods and an array of OneAgentInstance objects for status update
func findOutdatedPodsInstaller(pods []corev1.Pod, dtc dtclient.Client, instance *dynatracev1alpha1.OneAgent, logger logr.Logger) ([]corev1.Pod, error) {
	var doomedPods []corev1.Pod

	for _, pod := range pods {
		ver, err := dtc.GetAgentVersionForIP(pod.Status.HostIP)
		if err != nil {
			err = handleAgentVersionForIPError(err, instance, pod, nil)
			if err != nil {
				return doomedPods, err
			}
		} else {
			if isDesiredNewer(ver, instance.Status.Version, logger) {
				doomedPods = append(doomedPods, pod)
			}
		}
	}

	return doomedPods, nil
}

func (r *ReconcileOneAgent) findPods(ctx context.Context, instance *dynatracev1alpha1.OneAgent) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(buildLabels(instance.GetName())),
	}
	err := r.client.List(ctx, podList, listOptions...)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func isDesiredNewer(actual string, desired string, logger logr.Logger) bool {
	aa := strings.Split(actual, ".")
	da := strings.Split(desired, ".")

	for i := 0; i < len(aa); i++ {
		if i == len(aa)-1 {
			if aa[i] < da[i] {
				return true
			} else if aa[i] > da[i] {
				var err = errors.New("downgrade error")
				logger.Error(err, "downgrade detected! downgrades are not supported")
				return false
			} else {
				return false
			}
		}

		av, err := strconv.Atoi(aa[i])
		if err != nil {
			logger.Error(err, "failed to parse actual version number", "actual", actual)
			return false
		}

		dv, err := strconv.Atoi(da[i])
		if err != nil {
			logger.Error(err, "failed to parse desired version number", "desired", desired)
			return false
		}

		if av < dv {
			return true
		}
		if av > dv {
			var err = errors.New("downgrade error")
			logger.Error(err, "downgrade detected! downgrades are not supported")
			return false
		}
	}

	return false
}
