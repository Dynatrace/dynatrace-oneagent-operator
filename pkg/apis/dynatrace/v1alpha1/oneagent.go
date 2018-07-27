package v1alpha1

import (
	"errors"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildPodLabels returns generic labels based on this CR
func (oa *OneAgent) BuildLabels() map[string]string {
	return map[string]string{
		"dynatrace": "oneagent",
		"oneagent":  oa.Name,
	}
}

// BuildDaemonSet returns a basic DaemonSet object without DaemonSetSpec
func (oa *OneAgent) BuildDaemonSet() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      oa.Name,
			Namespace: oa.Namespace,
		},
	}
}

// Validate sanity checks if essential fields in the custom resource are available
//
// Return an error in the following conditions
// - ApiUrl empty
func (oa *OneAgent) Validate() error {
	var msg []string
	if len(oa.Spec.ApiUrl) == 0 {
		msg = append(msg, ".spec.apiUrl is missing")
	}
	if len(msg) > 0 {
		return errors.New(strings.Join(msg, ", "))
	}
	return nil
}
