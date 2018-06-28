package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type OneAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []OneAgent `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type OneAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              OneAgentSpec   `json:"spec"`
	Status            OneAgentStatus `json:"status,omitempty"`
}

type OneAgentSpec struct {
	ApiUrl           string              `json:"apiUrl"`
	SkipCertCheck    bool                `json:"skipCertCheck,omitempty"`
	NodeSelector     map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations      []corev1.Toleration `json:"tolerations,omitempty"`
	WaitReadySeconds *uint16             `json:"waitReadySeconds,omitempty"`
	// Installer image
	// Defaults to docker.io/dynatrace/oneagent:latest
	Image string `json:"image,omitempty"`
	// Name of secret containing tokens
	// Secret must contain keys `apiToken` and `paasToken`
	Tokens string `json:"tokens"`
}
type OneAgentStatus struct {
	Version          string                      `json:"version,omitempty"`
	Items            map[string]OneAgentInstance `json:"items,omitempty"`
	UpdatedTimestamp metav1.Time                 `json:"updatedTimestamp,omitempty"`
	Tokens           string                      `json:"tokens,omitempty"`
}
type OneAgentInstance struct {
	PodName string `json:"podName,omitempty"`
	Version string `json:"version,omitempty"`
}
