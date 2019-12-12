package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OneAgentSpec defines the desired state of OneAgent
// +k8s:openapi-gen=true
type OneAgentSpec struct {
	// Dynatrace api url including `/api` path at the end
	// either set ENVIRONMENTID to the proper tenant id or change the apiUrl as a whole, e.q. for Managed
	// +kubebuilder:validation:Required
	ApiUrl string `json:"apiUrl"`
	// Disable certificate validation checks for installer download and API communication
	SkipCertCheck bool `json:"skipCertCheck,omitempty"`
	// Node selector to control the selection of nodes (optional)
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ (optional)
	// +listType=set
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Defines the time to wait until OneAgent pod is ready after update - default 300 sec (optional)
	// +kubebuilder:validation:Minimum=0
	WaitReadySeconds *uint16 `json:"waitReadySeconds,omitempty"`
	// Installer image
	// Defaults to docker.io/dynatrace/oneagent:latest
	Image string `json:"image,omitempty"`
	// Name of secret containing tokens
	// Secret must contain keys `apiToken` and `paasToken`
	Tokens string `json:"tokens,omitempty"`
	// Arguments to the installer.
	// +listType=set
	Args []string `json:"args,omitempty"`
	// List of environment variables to set for the installer.
	// +listType=set
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Compute Resources required by OneAgent containers.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that
	// name. If not specified the setting will be removed from the DaemonSet.
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// If enabled, OneAgent pods won't be restarted automatically in case a new version is available
	DisableAgentUpdate bool `json:"disableAgentUpdate,omitempty"`
	// If enabled, Istio on the cluster will be configured automatically to allow access to the Dynatrace environment.
	EnableIstio bool `json:"enableIstio,omitempty"`
	// DNS Policy for the OneAgent pods.
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`
	// Name of the service account for the OneAgent
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

type OneAgentConditionType string

const (
	ApiTokenConditionType  OneAgentConditionType = "ApiToken"
	PaaSTokenConditionType OneAgentConditionType = "PaaSToken"
)

type OneAgentCondition struct {
	Type    OneAgentConditionType  `json:"type"`
	Status  corev1.ConditionStatus `json:"status"`
	Reason  string                 `json:"reason"`
	Message string                 `json:"message"`
}

type OneAgentPhaseType string

const (
	Running   OneAgentPhaseType = "Running"
	Deploying OneAgentPhaseType = "Deploying"
	Error     OneAgentPhaseType = "Error"
)

// OneAgentStatus defines the observed state of OneAgent
// +k8s:openapi-gen=true
type OneAgentStatus struct {
	Version          string                      `json:"version,omitempty"`
	Instances        map[string]OneAgentInstance `json:"instances,omitempty"`
	UpdatedTimestamp metav1.Time                 `json:"updatedTimestamp,omitempty"`
	// Defines the current state (Running, Updating, Error, ...)
	Phase OneAgentPhaseType `json:"phase,omitempty"`
	// +listType=set
	// +optional
	Conditions []*OneAgentCondition `json:"conditions,omitempty"`
}

type OneAgentInstance struct {
	PodName   string `json:"podName,omitempty"`
	Version   string `json:"version,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OneAgent is the Schema for the oneagents API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=oneagents,scope=Namespaced
// +kubebuilder:printcolumn:name="ApiUrl",type=string,JSONPath=`.spec.apiUrl`
// +kubebuilder:printcolumn:name="Tokens",type=string,JSONPath=`.spec.tokens`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.version`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OneAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OneAgentSpec `json:"spec"`
	// +optional
	Status OneAgentStatus `json:"status"`
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
