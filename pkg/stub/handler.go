package stub

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"

	"github.com/coreos/operator-sdk/pkg/sdk/action"
	"github.com/coreos/operator-sdk/pkg/sdk/handler"
	"github.com/coreos/operator-sdk/pkg/sdk/query"
	"github.com/coreos/operator-sdk/pkg/sdk/types"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	//"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewHandler() handler.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx types.Context, event types.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.OneAgent:
		oneagent := o
		updateStatus := false
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status": oneagent.Status}).Info("received oneagent")

		ds := getDaemonSet(oneagent)

		err := query.Get(ds)
		if err != nil && apierrors.IsNotFound(err) {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name}).Info("deploying daemonset")
			err = action.Create(ds)
			if err != nil {
				logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to deploy daemonset")
			}
		}

		// query oneagent pods
		podList := getPodList()
		labelSelector := labels.SelectorFromSet(getLabels(oneagent)).String()
		listOps := &metav1.ListOptions{LabelSelector: labelSelector}
		err = query.List(oneagent.Namespace, podList, query.WithListOptions(listOps))
		if err != nil {
			logrus.WithFields(logrus.Fields{"pods": podList, "error": err}).Error("failed to query pods")
			return err
		}

		// prepare update status.items
		instances := []v1alpha1.OneAgentInstance{}
		for _, pod := range podList.Items {
			logrus.WithFields(logrus.Fields{"pod": pod.Name, "nodeName": pod.Spec.NodeName}).Debug("processing pod")
			item := v1alpha1.OneAgentInstance{
				PodName:  pod.Name,
				NodeName: pod.Spec.NodeName,
			}
			instances = append(instances, item)
		}
		if !reflect.DeepEqual(instances, oneagent.Status.Items) {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status.items": instances}).Info("prepare status update")
			updateStatus = true
			oneagent.Status.Items = instances
		}

		// prepare update status.version
		podsToDelete := []corev1.Pod{}
		if oneagent.Status.Version != oneagent.Spec.Version {
			oneagent.Status.Version = oneagent.Spec.Version
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status.version": oneagent.Status.Version}).Info("prepare status update")
			updateStatus = true

			podsToDelete = podList.Items
		}

		// restart daemonset
		err = deletePods(oneagent, podsToDelete)
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to delete pods")
			return err
		}

		// update status
		if updateStatus {
			oneagent.Status.UpdatedTimestamp = metav1.Now()
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status": oneagent.Status}).Info("updating status")
			err := action.Update(oneagent)
			if err != nil {
				logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to update status")
				return err
			}
		}
	}
	return nil
}

// getPodList returns a v1.PodList object
func getPodList() *corev1.PodList {
	return &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
}

// deletePods deletes a list of pods
func deletePods(cr *v1alpha1.OneAgent, pods []corev1.Pod) error {
	for _, pod := range pods {
		// delete pod
		logrus.WithFields(logrus.Fields{"pod": pod.Name, "nodeName": pod.Spec.NodeName}).Info("deleting pod")
		err := action.Delete(&pod)
		if err != nil {
			logrus.WithFields(logrus.Fields{"pod": pod.Name, "error": err}).Error("failed to delete pod")
			return err
		}

		// wait for pod on node to get "Running" again
		fieldSelector, err := fields.ParseSelector(fmt.Sprintf("spec.nodeName=%v,status.phase=Running,metadata.name!=%v", pod.Spec.NodeName, pod.Name))
		labelSelector := labels.SelectorFromSet(getLabels(cr))
		logrus.WithFields(logrus.Fields{"field-selector": fieldSelector}).Debug("waiting for new pod to get ready")
		listOps := &metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
		state := 0
		for state == 0 {
			time.Sleep(10 * time.Second)
			pList := getPodList()
			err = query.List(cr.Namespace, pList, query.WithListOptions(listOps))
			if err != nil {
				logrus.WithFields(logrus.Fields{"pods": pList, "error": err}).Error("failed to query pods")
				return err
			}
			if len(pList.Items) > 0 {
				// assume all containers are ready
				state = 1
				for _, p := range pList.Items {
					for _, c := range p.Status.ContainerStatuses {
						if !c.Ready {
							// oops, this container is not ready
							state = 0
						}
					}
					logrus.WithFields(logrus.Fields{"pod": p.Name, "nodeName": p.Spec.NodeName}).Debug("pod not ready")
				}
			}
		}
	}
	return nil
}

// getDaemonSet returns a oneagent DaemonSet object
func getDaemonSet(cr *v1alpha1.OneAgent) *appsv1.DaemonSet {
	trueVar := true
	labels := getLabels(cr)
	// compound nodeSelector
	nodeSelector := map[string]string{"beta.kubernetes.io/os": "linux"}
	for k, v := range cr.Spec.NodeSelector {
		nodeSelector[k] = v
	}

	ds := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: "host-root",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/",
							}},
					},
					},
					NodeSelector: nodeSelector,
					HostNetwork:  true,
					HostPID:      true,
					HostIPC:      true,
					Containers: []corev1.Container{{
						Image: "dynatrace/oneagent",
						Name:  "dynatrace-oneagent",
						Env: []corev1.EnvVar{
							{Name: "ONEAGENT_INSTALLER_SCRIPT_URL", Value: fmt.Sprintf("%s/v1/deployment/installer/agent/unix/default/latest?Api-Token=%s&arch=x86&flavor=default", cr.Spec.ApiUrl, cr.Spec.ApiToken)},
							{Name: "ONEAGENT_INSTALLER_SKIP_CERT_CHECK", Value: strconv.FormatBool(cr.Spec.SkipCertCheck)},
						},
						Args: []string{"APP_LOG_CONTENT_ACCESS=1"},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "host-root",
							MountPath: "/mnt/root",
						}},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &trueVar,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.FromInt(50000),
									Host: "127.0.0.1",
								},
							},
							InitialDelaySeconds: 30,
						},
					}},
				},
			},
		},
	}

	ownerRef := metav1.OwnerReference{
		APIVersion:         cr.APIVersion,
		Kind:               cr.Kind,
		Name:               cr.Name,
		UID:                cr.UID,
		Controller:         &trueVar,
		BlockOwnerDeletion: &trueVar,
	}

	ds.SetOwnerReferences(append(ds.GetOwnerReferences(), ownerRef))

	return ds
}

// getPodLables return labels set on all objects created by this CR
func getLabels(cr *v1alpha1.OneAgent) map[string]string {
	return map[string]string{
		"dynatrace": "oneagent",
		"oneagent":  cr.Name,
	}
}
