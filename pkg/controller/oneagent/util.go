package oneagent

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/apis/dynatrace/v1alpha1"
	dtclient "github.com/Dynatrace/dynatrace-oneagent-operator/pkg/dynatrace-client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// BuildLabels returns generic labels based on the name given for a Dynatrace OneAgent
func buildLabels(name string) map[string]string {
	return map[string]string{
		"dynatrace": "oneagent",
		"oneagent":  name,
	}
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
// - ApiUrl empty
func validate(cr *dynatracev1alpha1.OneAgent) error {
	var msg []string
	if cr.Spec.ApiUrl == "" {
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
func hasSpecChanged(dsSpec *appsv1.DaemonSetSpec, crSpec *dynatracev1alpha1.OneAgentSpec) bool {
	actualSpec := crSpec.DeepCopy()
	copyDaemonSetSpecToOneAgentSpec(dsSpec, actualSpec)
	//fmt.Println(pretty.Compare(crSpec, actualSpec))
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
func copyDaemonSetSpecToOneAgentSpec(dsSpec *appsv1.DaemonSetSpec, crSpec *dynatracev1alpha1.OneAgentSpec) {
	// ApiUrl
	// SkipCertCheck
	// NodeSelector
	crSpec.NodeSelector = nil
	if dsSpec.Template.Spec.NodeSelector != nil {
		in, out := &dsSpec.Template.Spec.NodeSelector, &crSpec.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	// Tolerations
	crSpec.Tolerations = nil
	if dsSpec.Template.Spec.Tolerations != nil {
		in, out := &dsSpec.Template.Spec.Tolerations, &crSpec.Tolerations
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	// PriorityClassName
	crSpec.PriorityClassName = dsSpec.Template.Spec.PriorityClassName
	// Image
	crSpec.Image = ""
	if len(dsSpec.Template.Spec.Containers) == 1 {
		crSpec.Image = dsSpec.Template.Spec.Containers[0].Image
	}
	// Tokens
	// WaitReadySeconds: not used in DaemonSet
	// Args
	crSpec.Args = nil
	if len(dsSpec.Template.Spec.Containers) == 1 && dsSpec.Template.Spec.Containers[0].Args != nil {
		in, out := &dsSpec.Template.Spec.Containers[0].Args, &crSpec.Args
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	// Env
	crSpec.Env = nil
	if len(dsSpec.Template.Spec.Containers) == 1 && dsSpec.Template.Spec.Containers[0].Env != nil {
		in, out := &dsSpec.Template.Spec.Containers[0].Env, &crSpec.Env
		*out = make([]corev1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	// Resources
	crSpec.Resources = corev1.ResourceRequirements{}
	if len(dsSpec.Template.Spec.Containers) == 1 {
		dsSpec.Template.Spec.Containers[0].Resources.DeepCopyInto(&crSpec.Resources)
	}
}

func getToken(secret *corev1.Secret, key string) (string, error) {
	value, ok := secret.Data[key]
	if !ok {
		err := fmt.Errorf("missing token %s", key)
		return "", err
	}

	return strings.TrimSpace(string(value)), nil
}

func verifySecret(secret *corev1.Secret) error {
	var err error

	for _, token := range []string{dynatracePaasToken, dynatraceApiToken} {
		_, err = getToken(secret, token)
		if err != nil {
			return fmt.Errorf("invalid secret %s, %s", secret.Name, err)
		}
	}

	return nil
}

// getPodsToRestart determines if a pod needs to be restarted in order to get the desired agent version
// Returns an array of pods and an array of OneAgentInstance objects for status update
func getPodsToRestart(pods []corev1.Pod, dtc dtclient.Client, instance *dynatracev1alpha1.OneAgent) ([]corev1.Pod, map[string]dynatracev1alpha1.OneAgentInstance) {
	var doomedPods []corev1.Pod
	instances := make(map[string]dynatracev1alpha1.OneAgentInstance)

	for _, pod := range pods {
		item := dynatracev1alpha1.OneAgentInstance{
			PodName: pod.Name,
		}
		ver, err := dtc.GetVersionForIp(pod.Status.HostIP)
		if err != nil {
			// use last know version if available
			if i, ok := instance.Status.Items[pod.Spec.NodeName]; ok {
				item.Version = i.Version
			}
		} else {
			item.Version = ver
			if ver != instance.Status.Version {
				doomedPods = append(doomedPods, pod)
			}
		}
		instances[pod.Spec.NodeName] = item
	}

	return doomedPods, instances
}

func getInternalIPForNode(node corev1.Node) string {

	addresses := node.Status.Addresses
	if len(addresses) == 0 {
		return ""
	}
	for _, addr := range addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}
