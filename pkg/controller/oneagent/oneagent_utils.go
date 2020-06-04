package oneagent

import (
	"errors"
	"net/http"
	"reflect"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dtclient"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func mergeLabels(labels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range labels {
		if m != nil {
			for k, v := range m {
				res[k] = v
			}
		}
	}

	return res
}

// buildLabels returns generic labels based on the name given for a Dynatrace OneAgent
func buildLabels(name string) map[string]string {
	return map[string]string{
		"dynatrace": "oneagent",
		"oneagent":  name,
	}
}

// isPredefinedLabel returns true if the label is predefined by the Operator.
func isPredefinedLabel(label string) bool {
	return label == "dynatrace" || label == "oneagent"
}

// getPodReadyState determines the overall ready state of a Pod.
// Returns true if all containers in the Pod are ready.
func getPodReadyState(p *corev1.Pod) bool {
	ready := true
	for _, c := range p.Status.ContainerStatuses {
		ready = ready && c.Ready
	}

	return ready
}

// validate sanity checks if essential fields in the custom resource are available
//
// Return an error in the following conditions
// - APIURL empty
func validate(cr dynatracev1alpha1.BaseOneAgentDaemonSet) error {
	var msg []string
	if cr.GetOneAgentSpec().APIURL == "" {
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
func hasSpecChanged(dsSpec, dsExpSpec *appsv1.DaemonSetSpec) bool {
	if len(dsSpec.Template.Spec.Containers) != len(dsExpSpec.Template.Spec.Containers) {
		return true
	}

	for _, fn := range []func(*appsv1.DaemonSetSpec) interface{}{
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.NodeSelector },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.Tolerations },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.ServiceAccountName },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.PriorityClassName },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.DNSPolicy },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.Volumes },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Labels },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.Containers[0].Args },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.Containers[0].Env },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.Containers[0].Image },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.Containers[0].Resources },
		func(ds *appsv1.DaemonSetSpec) interface{} { return ds.Template.Spec.Containers[0].VolumeMounts },
	} {
		if !reflect.DeepEqual(fn(dsSpec), fn(dsExpSpec)) {
			return true
		}
	}

	return false
}

// getPodsToRestart determines if a pod needs to be restarted in order to get the desired agent version
// Returns an array of pods and an array of OneAgentInstance objects for status update
func getPodsToRestart(pods []corev1.Pod, dtc dtclient.Client, instance dynatracev1alpha1.BaseOneAgentDaemonSet) ([]corev1.Pod, map[string]dynatracev1alpha1.OneAgentInstance, error) {
	var doomedPods []corev1.Pod
	instances := make(map[string]dynatracev1alpha1.OneAgentInstance)

	for _, pod := range pods {
		item := dynatracev1alpha1.OneAgentInstance{
			PodName:   pod.Name,
			IPAddress: pod.Status.HostIP,
		}
		ver, err := dtc.GetAgentVersionForIP(pod.Status.HostIP, instance.GetOneAgentSpec().NetworkZone)
		if err != nil {
			var serr dtclient.ServerError
			if ok := errors.As(err, &serr); ok && serr.Code == http.StatusTooManyRequests {
				return nil, nil, err
			}
			// use last know version if available
			if i, ok := instance.GetOneAgentStatus().Instances[pod.Spec.NodeName]; ok {
				item.Version = i.Version
			}
		} else {
			item.Version = ver
			if ver != instance.GetOneAgentStatus().Version {
				doomedPods = append(doomedPods, pod)
			}
		}
		instances[pod.Spec.NodeName] = item
	}

	return doomedPods, instances, nil
}
