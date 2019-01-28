package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// OneAgentSpec defines the desired state of OneAgent
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
	// Arguments to the installer.
	Args []string `json:"args,omitempty"`
	// List of environment variables to set for the installer.
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Compute Resources required by OneAgent containers.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that
	// name. If not specified the setting will be removed from the DaemonSet.
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// If enabled, OneAgent pods won't be restarted automatically in case a new version is available
	DisableAgentUpdate bool `json:"disableAgentUpdate,omitempty"`
}

// OneAgentStatus defines the observed state of OneAgent
type OneAgentStatus struct {
	Version          string                      `json:"version,omitempty"`
	Items            map[string]OneAgentInstance `json:"items,omitempty"`
	UpdatedTimestamp metav1.Time                 `json:"updatedTimestamp,omitempty"`
}

type OneAgentInstance struct {
	PodName string `json:"podName,omitempty"`
	Version string `json:"version,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneAgent is the Schema for the oneagents API
// +k8s:openapi-gen=true
type OneAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OneAgentSpec   `json:"spec,omitempty"`
	Status OneAgentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneAgentList contains a list of OneAgent
type OneAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OneAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OneAgent{}, &OneAgentList{})
}
