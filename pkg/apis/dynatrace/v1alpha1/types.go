package v1alpha1

import (
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
	ApiUrl           string            `json:"apiUrl"`
	ApiToken         string            `json:"apiToken"`
	PaasToken        string            `json:"paasToken"`
	Version          string            `json:"version,omitempty"`
	SkipCertCheck    bool              `json:"skipCertCheck,omitempty"`
	NodeSelector     map[string]string `json:"nodeSelector,omitempty"`
	WaitReadySeconds *uint16           `json:"waitReadySeconds,omitempty"`
}
type OneAgentStatus struct {
	Version          string             `json:"version,omitempty"`
	Items            []OneAgentInstance `json:"items,omitempty"`
	UpdatedTimestamp metav1.Time        `json:"updatedTimestamp,omitempty"`
}
type OneAgentInstance struct {
	PodName  string `json:"podName,omitempty"`
	NodeName string `json:"nodeName,omitempty"`
	Version  string `json:"version,omitempty"`
}
