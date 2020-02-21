package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OneAgentSpec defines the desired state of OneAgent
// +k8s:openapi-gen=true
type OneAgentSpec struct {
	// Location of the Dynatrace API to connect to, including your specific environment ID
	// +kubebuilder:validation:Required
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="API URL"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ApiUrl string `json:"apiUrl"`
	// Disable certificate validation checks for installer download and API communication
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Skip Certificate Check"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	SkipCertCheck bool `json:"skipCertCheck,omitempty"`
	// Node selector to control the selection of nodes (optional)
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Node Selector"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Optional: set tolerations for the OneAgent pods
	// +listType=set
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Tolerations"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:io.kubernetes:Tolerations"
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Optional: Defines the time to wait until OneAgent pod is ready after update - default 300 sec
	// +kubebuilder:validation:Minimum=0
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Wait seconds until ready"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:number"
	WaitReadySeconds *uint16 `json:"waitReadySeconds,omitempty"`
	// Optional: the Dynatrace installer container image
	// Defaults to docker.io/dynatrace/oneagent:latest
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Image"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Image string `json:"image,omitempty"`
	// Credentials for the OneAgent to connect back to Dynatrace.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="API and PaaS Tokens"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Tokens string `json:"tokens,omitempty"`
	// Optional: Arguments to the OneAgent installer
	// +listType=set
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="OneAgent installer arguments"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Args []string `json:"args,omitempty"`
	// Optional: List of environment variables to set for the installer
	// +listType=set
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="OneAgent environment variable installer arguments"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Optional: define resources requests and limits for single pods
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Resource Requirements"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:resourceRequirements"
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Optional: If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that
	// name. If not specified the setting will be removed from the DaemonSet.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Priority Class name"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:io.kubernetes:PriorityClass"
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// Disable automatic restarts of OneAgent pods in case a new version is available
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Disable Agent update"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	DisableAgentUpdate bool `json:"disableAgentUpdate,omitempty"`
	// If enabled, Istio on the cluster will be configured automatically to allow access to the Dynatrace environment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Enable Istio automatic management"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	EnableIstio bool `json:"enableIstio,omitempty"`
	// Optional: Sets DNS Policy for the OneAgent pods
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="DNS Policy"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`
	// Optional: set custom Service Account Name used with OneAgent pods
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Service Account name"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:io.kubernetes:ServiceAccount"
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// Optional: Adds additional labels for the OneAgent pods
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Labels"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Labels map[string]string `json:"labels,omitempty"`
	// Optional: Set custom proxy settings either directly or from a secret with the field 'proxy'
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Proxy"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Proxy *OneAgentProxy `json:"proxy,omitempty"`
	// Optional: Adds custom RootCAs from a configmap
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="RootCAs"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	RootCAs string `json:"rootCAs,omitempty"`
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

type OneAgentProxy struct {
	Value     string `json:"value,omitempty"`
	ValueFrom string `json:"valueFrom,omitempty"`
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
	// Dynatrace version being used.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Version"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	Version   string                      `json:"version,omitempty"`
	Instances map[string]OneAgentInstance `json:"instances,omitempty"`
	// The timestamp when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`
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

// Dyantrace OneAgent for full-stack monitoring
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=oneagents,scope=Namespaced
// +kubebuilder:printcolumn:name="ApiUrl",type=string,JSONPath=`.spec.apiUrl`
// +kubebuilder:printcolumn:name="Tokens",type=string,JSONPath=`.spec.tokens`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.status.version`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Dynatrace OneAgent"
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`DaemonSet,v1beta2,""`
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Pod,v1,""`
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

// SetPhase sets the status phase on the OneAgent object
func (oa *OneAgent) SetPhase(phase OneAgentPhaseType) bool {
	upd := phase != oa.Status.Phase
	oa.Status.Phase = phase
	return upd
}

// SetPhaseOnError fills the phase with the Error value in case of any error
func (oa *OneAgent) SetPhaseOnError(err error) bool {
	if err != nil {
		return oa.SetPhase(Error)
	}
	return false
}
