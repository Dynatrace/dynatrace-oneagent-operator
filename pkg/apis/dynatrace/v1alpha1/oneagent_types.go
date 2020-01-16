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
	// Labels for the OneAgent pods
	Labels map[string]string `json:"labels,omitempty"`
}

type OneAgentConditionType string

const (
	APITokenConditionType  OneAgentConditionType = "APIToken"
	PaaSTokenConditionType OneAgentConditionType = "PaaSToken"
)

type OneAgentCondition struct {
	Type    OneAgentConditionType  `json:"type"`
	Status  corev1.ConditionStatus `json:"status"`
	Reason  string                 `json:"reason"`
	Message string                 `json:"message"`
}

// Possible reasons for ApiToken and PaaSToken conditions.
const (
	// ReasonTokenReady is set when a token has passed verifications.
	ReasonTokenReady = "TokenReady"
	// ReasonTokenSecretNotFound is set when the referenced secret can't be found.
	ReasonTokenSecretNotFound = "TokenSecretNotFound"
	// ReasonTokenMissing is set when the field is missing on the secret.
	ReasonTokenMissing = "TokenMissing"
	// ReasonTokenUnauthorized is set when a token is unauthorized to query the Dynatrace API.
	ReasonTokenUnauthorized = "TokenUnauthorized"
	// ReasonTokenScopeMissing is set when the token is missing the required scope for the Dynatrace API.
	ReasonTokenScopeMissing = "TokenScopeMissing"
	// ReasonTokenError is set when an unknown error has been found when verifying the token.
	ReasonTokenError = "TokenError"
)

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
	// LastAPITokenProbeTimestamp tracks when the last request for the API token validity was sent.
	LastAPITokenProbeTimestamp *metav1.Time `json:"lastAPITokenProbeTimestamp,omitempty"`
	// LastPaaSTokenProbeTimestamp tracks when the last request for the PaaS token validity was sent.
	LastPaaSTokenProbeTimestamp *metav1.Time `json:"lastPaaSTokenProbeTimestamp,omitempty"`
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

// Condition returns OneAgentCondition for the given conditionType
func (oa *OneAgent) Condition(conditionType OneAgentConditionType) *OneAgentCondition {
	for i := range oa.Status.Conditions {
		if oa.Status.Conditions[i].Type == conditionType {
			return oa.Status.Conditions[i]
		}
	}

	condition := OneAgentCondition{Type: conditionType}
	oa.Status.Conditions = append(oa.Status.Conditions, &condition)
	return &condition
}

// SetCondition fills the state for a condition, return true if there were changes on the condition.
func (oa *OneAgent) SetCondition(condType OneAgentConditionType, status corev1.ConditionStatus, reason, message string) bool {
	c := oa.Condition(condType)
	upd := c.Status != status || c.Reason != reason || c.Message != message
	c.Status = status
	c.Reason = reason
	c.Message = message
	return upd
}

// SetFailureCondition fills the state for a failing condition
func (oa *OneAgent) SetFailureCondition(conditionType OneAgentConditionType, reason, message string) bool {
	return oa.SetCondition(conditionType, corev1.ConditionFalse, reason, message)
}
