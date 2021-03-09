/*
Copyright 2020 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OneAgentSpec defines the desired state of OneAgent
// +k8s:openapi-gen=true
type OneAgentSpec struct {
	BaseOneAgentSpec `json:",inline"`

	// Node selector to control the selection of nodes (optional)
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Node Selector"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:selector:Node"
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Optional: set tolerations for the OneAgent pods
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
	// Defaults to docker.io/dynatrace/oneagent:latest for Kubernetes and to registry.connect.redhat.com/dynatrace/oneagent for OpenShift
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Image"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Image string `json:"image,omitempty"`

	// Optional: If specified, indicates the OneAgent version to use
	// Defaults to latest
	// Example: {major.minor.release} - 1.200.0
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="OneAgent version"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	AgentVersion string `json:"agentVersion,omitempty"`

	// Optional: Pull secret for your private registry
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Custom PullSecret"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	CustomPullSecret string `json:"customPullSecret,omitempty"`

	// Optional: define resources requests and limits for single pods
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Resource Requirements"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:resourceRequirements"
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Work in progress
	// Disables automatic injection into applications
	// OneAgentAPM together with the webhook will then do the injection
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=false
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Webhook injection"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	WebhookInjection bool `json:"webhookInjection,omitempty"`

	// Optional: Arguments to the OneAgent installer
	// +listType=set
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="OneAgent installer arguments"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Args []string `json:"args,omitempty"`

	// Optional: List of environment variables to set for the installer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="OneAgent environment variable installer arguments"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Env []corev1.EnvVar `json:"env,omitempty"`

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

	// Optional: Runs the OneAgent Pods as unprivileged (Early Adopter)
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Use unprivileged mode"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	UseUnprivilegedMode *bool `json:"useUnprivilegedMode,omitempty"`
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
	BaseOneAgentStatus `json:",inline"`

	// Dynatrace version being used.
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Version"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	Version string `json:"version,omitempty"`

	Instances map[string]OneAgentInstance `json:"instances,omitempty"`

	// Defines the current state (Running, Updating, Error, ...)
	Phase OneAgentPhaseType `json:"phase,omitempty"`

	// LastUpdateProbeTimestamp defines the last timestamp when the querying for updates have been done
	LastUpdateProbeTimestamp *metav1.Time `json:"lastUpdateProbeTimestamp,omitempty"`

	// ImageHash contains the hash for the latest immutable image seen.
	ImageHash string `json:"imageHash,omitempty"`

	// ImageVersion contains the version for the latest immutable image seen.
	ImageVersion string `json:"imageVersion,omitempty"`

	// LastImageVersionProbeTimestamp keeps track of the last time the Operator looked at the image version
	LastImageVersionProbeTimestamp *metav1.Time `json:"lastImageVersionProbeTimestamp,omitempty"`
}

type OneAgentInstance struct {
	PodName   string `json:"podName,omitempty"`
	Version   string `json:"version,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// For full-stack monitoring, including complete APM and infrastructure layer observability.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=oneagents,scope=Namespaced,categories=dynatrace
// +kubebuilder:printcolumn:name="ApiUrl",type=string,JSONPath=`.spec.apiUrl`
// +kubebuilder:printcolumn:name="Tokens",type=string,JSONPath=`.status.tokens`
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

// GetSpec returns the corresponding BaseOneAgentSpec for the instance's Spec.
func (oa *OneAgent) GetSpec() *BaseOneAgentSpec {
	return &oa.Spec.BaseOneAgentSpec
}

// GetStatus returns the corresponding BaseOneAgentStatus for the instance's Status.
func (oa *OneAgent) GetStatus() *BaseOneAgentStatus {
	return &oa.Status.BaseOneAgentStatus
}

// SetPhase sets the status phase on the OneAgent object
func (oa *OneAgentStatus) SetPhase(phase OneAgentPhaseType) bool {
	upd := phase != oa.Phase
	oa.Phase = phase
	return upd
}

// SetPhaseOnError fills the phase with the Error value in case of any error
func (oa *OneAgentStatus) SetPhaseOnError(err error) bool {
	if err != nil {
		return oa.SetPhase(Error)
	}
	return false
}

func (oa *OneAgent) GetOneAgentSpec() *OneAgentSpec {
	return &oa.Spec
}

func (oa *OneAgent) GetOneAgentStatus() *OneAgentStatus {
	return &oa.Status
}
