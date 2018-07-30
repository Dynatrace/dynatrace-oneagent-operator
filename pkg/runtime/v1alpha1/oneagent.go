package v1alpha1

import (
	"errors"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"

	api "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Validate sanity checks if essential fields in the custom resource are available
//
// Return an error in the following conditions
// - ApiUrl empty
func Validate(oa *api.OneAgent) error {
	var msg []string
	if oa.Spec.ApiUrl == "" {
		msg = append(msg, ".spec.apiUrl is missing")
	}
	if len(msg) > 0 {
		return errors.New(strings.Join(msg, ", "))
	}
	return nil
}

// hasSpecChanged compares essential OneAgent custom resource settings with the
// actual settings in the DaemonSet object
//
// actualSpec gets initialized with values from the custom resource and updated
// with values from the actual settings from the daemonset.
func HasSpecChanged(dsSpec *appsv1.DaemonSetSpec, crSpec *api.OneAgentSpec) bool {
	actualSpec := crSpec.DeepCopy()
	CopyDaemonSetSpecToOneAgentSpec(dsSpec, actualSpec)
	if !reflect.DeepEqual(crSpec, actualSpec) {
		return true
	}
	return false
}

// copyDaemonSetSpecToOneAgentSpec extracts essential data from a DaemonSetSpec
// into a OneAgentSpec
//
// Reference types in custom resource spec need to be reset to nil in case its
// value is missing in the daemonset as well.
func CopyDaemonSetSpecToOneAgentSpec(ds *appsv1.DaemonSetSpec, cr *api.OneAgentSpec) {
	// ApiUrl
	// SkipCertCheck
	// NodeSelector
	cr.NodeSelector = nil
	if ds.Template.Spec.NodeSelector != nil {
		in, out := &ds.Template.Spec.NodeSelector, &cr.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	// Tolerations
	cr.Tolerations = nil
	if ds.Template.Spec.Tolerations != nil {
		in, out := &ds.Template.Spec.Tolerations, &cr.Tolerations
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	// Image
	cr.Image = ""
	if len(ds.Template.Spec.Containers) == 1 {
		cr.Image = ds.Template.Spec.Containers[0].Image
	}
	// Tokens
	// WaitReadySeconds: not used in DaemonSet
	// Args
	cr.Args = nil
	if len(ds.Template.Spec.Containers) == 1 && ds.Template.Spec.Containers[0].Args != nil {
		in, out := &ds.Template.Spec.Containers[0].Args, &cr.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	// Env
	cr.Env = nil
	if len(ds.Template.Spec.Containers) == 1 && ds.Template.Spec.Containers[0].Env != nil {
		in, out := &ds.Template.Spec.Containers[0].Env, &cr.Env
		*out = make([]corev1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// applyOneAgentSettings applies the properties given by a OneAgent custom
// resource object to a DaemonSet object
func ApplyOneAgentSettings(ds *appsv1.DaemonSet, cr *api.OneAgent) {
	labels := cr.GetLabels()

	ds.ObjectMeta.Labels = labels

	ds.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}

	ds.Spec.Template.ObjectMeta = metav1.ObjectMeta{Labels: labels}

	ds.Spec.Template.Spec.NodeSelector = cr.Spec.NodeSelector
	ds.Spec.Template.Spec.Tolerations = cr.Spec.Tolerations

	if len(ds.Spec.Template.Spec.Containers) == 0 {
		ds.Spec.Template.Spec.Containers = []corev1.Container{{}}
	}
	ds.Spec.Template.Spec.Containers[0].Image = cr.Spec.Image
	ds.Spec.Template.Spec.Containers[0].Env = cr.Spec.Env
	ds.Spec.Template.Spec.Containers[0].Args = cr.Spec.Args
}

// applyOneAgentDefaults initializes a bare DaemonSet object with default
// values
func ApplyOneAgentDefaults(ds *appsv1.DaemonSet, cr *api.OneAgent) {
	trueVar := true

	ds.Spec = appsv1.DaemonSetSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{{
					Name: "host-root",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/",
						},
					},
				}},
				HostNetwork: true,
				HostPID:     true,
				HostIPC:     true,
				Containers: []corev1.Container{{
					Name:            "dynatrace-oneagent",
					ImagePullPolicy: corev1.PullAlways,
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "host-root",
						MountPath: "/mnt/root",
					}},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &trueVar,
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							Exec: &corev1.ExecAction{
								Command: []string{"pgrep", "-f", "oneagentwatchdog"},
							},
						},
						InitialDelaySeconds: 30,
						PeriodSeconds:       30,
					},
				}},
				ServiceAccountName: "dynatrace-oneagent",
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
}

// getPodsToRestart determines if a pod needs to be restarted in order to get the desired agent version
// Returns an array of pods and an array of OneAgentInstance objects for status update
func GetPodsToRestart(pods []corev1.Pod, dtc dtclient.Client, oneagent *api.OneAgent) ([]corev1.Pod, map[string]api.OneAgentInstance) {
	var doomedPods []corev1.Pod
	instances := make(map[string]api.OneAgentInstance)

	for _, pod := range pods {
		logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName}).Debug("processing pod")
		item := api.OneAgentInstance{
			PodName: pod.Name,
		}
		ver, err := dtc.GetVersionForIp(pod.Status.HostIP)
		if err != nil {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName, "hostIP": pod.Status.HostIP, "warning": err}).Warning("no agent found for host")
			// use last know version if available
			if i, ok := oneagent.Status.Items[pod.Spec.NodeName]; ok {
				item.Version = i.Version
			}
		} else {
			logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName, "version": ver}).Debug("")
			item.Version = ver
			if ver != oneagent.Status.Version {
				logrus.WithFields(logrus.Fields{"oneagent": oneagent.Name, "pod": pod.Name, "nodeName": pod.Spec.NodeName, "actual": ver, "desired": oneagent.Status.Version}).Info("")
				doomedPods = append(doomedPods, pod)
			}
		}
		instances[pod.Spec.NodeName] = item
	}

	return doomedPods, instances
}
