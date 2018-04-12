package stub

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"

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

// time between consecutive queries for a new pod to get ready
const splayTimeSeconds = uint16(10)

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

		// create'n'update daemonset
		err := updateDaemonSet(oneagent)
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Error("failed to create or update daemonset")
			return err
		}

		// initialize dynatrace client
		dtc, err := dtclient.NewClient(oneagent.Spec.ApiUrl, oneagent.Spec.ApiToken, oneagent.Spec.PaasToken)
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "error": err}).Warning("failed to get dynatrace rest client")
			return err
		}

		// get desired version
		desired, err := dtc.GetVersionForLatest(dtclient.OsUnix, dtclient.InstallerTypeDefault)
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "warning": err}).Warning("failed to get desired version")
			// TODO think about error handling
			// do not return err as it would trigger yet another reconciliation loop immediately
			return nil
		} else if desired != "" && oneagent.Status.Version != desired {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "previous": oneagent.Status.Version, "desired": desired}).Info("new version available")
			oneagent.Status.Version = desired
			updateStatus = true
		}

		// query oneagent pods
		podList := getPodList()
		labelSelector := labels.SelectorFromSet(getLabels(oneagent)).String()
		listOps := &metav1.ListOptions{LabelSelector: labelSelector}
		err = query.List(oneagent.Namespace, podList, query.WithListOptions(listOps))
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "pods": podList, "error": err}).Error("failed to query pods")
			return err
		}

		// determine pods to restart
		podsToDelete, instances := getPodsToRestart(podList.Items, dtc, oneagent)
		if !reflect.DeepEqual(instances, oneagent.Status.Items) {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "status.items": instances}).Info("status changed")
			updateStatus = true
			oneagent.Status.Items = instances
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
//
// Returns an error in the following conditions:
//  - failure on object deletion
//  - timeout on waiting for ready state
func deletePods(cr *v1alpha1.OneAgent, pods []corev1.Pod) error {
	for _, pod := range pods {
		// delete pod
		logrus.WithFields(logrus.Fields{"oneagent": cr.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName}).Info("deleting pod")
		err := action.Delete(&pod)
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": cr.Name, "pod": pod.Name, "error": err}).Error("failed to delete pod")
			return err
		}

		// wait for pod on node to get "Running" again
		var status error
		fieldSelector, _ := fields.ParseSelector(fmt.Sprintf("spec.nodeName=%v,status.phase=Running,metadata.name!=%v", pod.Spec.NodeName, pod.Name))
		labelSelector := labels.SelectorFromSet(getLabels(cr))
		logrus.WithFields(logrus.Fields{"field-selector": fieldSelector, "label-selector": labelSelector}).Debug("query pod")
		listOps := &metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
		for splay := uint16(0); splay < *cr.Spec.WaitReadySeconds; splay += splayTimeSeconds {
			time.Sleep(time.Duration(splayTimeSeconds) * time.Second)
			pList := getPodList()
			status = query.List(cr.Namespace, pList, query.WithListOptions(listOps))
			if status != nil {
				logrus.WithFields(logrus.Fields{"oneagent": cr.Name, "nodeName": pod.Spec.NodeName, "pods": pList, "warning": status}).Warning("failed to query pods")
				continue
			}
			if n := len(pList.Items); n == 1 && getPodReadyState(&pList.Items[0]) {
				break
			} else if n > 1 {
				status = fmt.Errorf("too many pods found: expected=1 actual=%i", n)
			}
		}
		if status != nil {
			logrus.WithFields(logrus.Fields{"oneagent": cr.Name, "nodeName": pod.Spec.NodeName, "warning": status}).Warning("timeout waiting on pod to get ready")
			return status
		}
	}

	return nil
}

// getPodReadyState determines the overall ready state of a Pod.
// Returns true if all containers in the Pod are ready.
func getPodReadyState(p *corev1.Pod) bool {
	ready := true
	for _, c := range p.Status.ContainerStatuses {
		logrus.WithFields(logrus.Fields{"pod": p.Name, "container": c.Name, "state": c.Ready}).Debug("test pod ready state")
		ready = ready && c.Ready
	}

	return ready
}

// updateDaemonSet creates a new DaemonSet object if it does not exist.
//
// Returns an error in the following conditions:
//  - all k8s apierrors except IsNotFound
//  - failure on daemonset creation
func updateDaemonSet(oa *v1alpha1.OneAgent) error {
	ds := getDaemonSet(oa)

	err := query.Get(ds)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logrus.WithFields(logrus.Fields{"oneagent": oa.Name}).Info("deploying daemonset")
			err = action.Create(ds)
			if err != nil {
				logrus.WithFields(logrus.Fields{"oneagent": oa.Name, "error": err}).Error("failed to deploy daemonset")
				return err
			}
		} else {
			logrus.WithFields(logrus.Fields{"oneagent": oa.Name, "error": err}).Error("failed to get daemonset")
			return err
		}
	}
	// TODO update daemonset

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
							},
						},
					}},
					NodeSelector: nodeSelector,
					HostNetwork:  true,
					HostPID:      true,
					HostIPC:      true,
					Containers: []corev1.Container{{
						Image: "dynatrace/oneagent",
						Name:  "dynatrace-oneagent",
						Env: []corev1.EnvVar{
							{Name: "ONEAGENT_INSTALLER_SCRIPT_URL", Value: fmt.Sprintf("%s/v1/deployment/installer/agent/unix/default/latest?Api-Token=%s&arch=x86&flavor=default", cr.Spec.ApiUrl, cr.Spec.PaasToken)},
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

// getPodsToRestart determines if a pod needs to be restarted in order to get the desired agent version
// Returns an array of pods and an array of OneAgentInstance objects for status update
func getPodsToRestart(pods []corev1.Pod, dtc dtclient.Client, oneagent *v1alpha1.OneAgent) ([]corev1.Pod, []v1alpha1.OneAgentInstance) {
	var doomedPods []corev1.Pod
	var instances []v1alpha1.OneAgentInstance

	for _, pod := range pods {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName}).Debug("processing pod")
		item := v1alpha1.OneAgentInstance{
			PodName:  pod.Name,
			NodeName: pod.Spec.NodeName,
		}
		ver, err := dtc.GetVersionForIp(net.ParseIP(pod.Status.HostIP))
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName, "hostIP": pod.Status.HostIP, "warning": err}).Warning("failed to get version")
			// use last know version if available
			// TODO replace .status.items with hash map for smarter lookups on nodename
			for i := range oneagent.Status.Items {
				if oneagent.Status.Items[i].NodeName == item.NodeName {
					item.Version = oneagent.Status.Items[i].Version
					break
				}
			}
		} else {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName, "version": ver}).Debug("")
			item.Version = ver
			if ver != oneagent.Status.Version {
				logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName, "actual": ver, "desired": oneagent.Status.Version}).Info("")
				doomedPods = append(doomedPods, pod)
			}
		}
		instances = append(instances, item)
	}

	return doomedPods, instances
}
