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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OneAgentAPMSpec defines the desired state of OneAgentAPM
// +k8s:openapi-gen=true
type OneAgentAPMSpec struct {
	BaseOneAgentSpec `json:",inline"`

	// Optional: Custom code modules OneAgent docker image
	// In case you have the docker image for the oneagent in a custom docker registry you need to provide it here
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Image"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonicx.ui:text"
	Image string `json:"image,omitempty"`

	// Optional: The version of the oneagent to be used
	// Default (if nothing set): latest
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Agent version"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonicx.ui:text"
	AgentVersion string `json:"agentVersion,omitempty"`
}

// OneAgentAPMStatus defines the observed state of OneAgentAPM
type OneAgentAPMStatus struct {
	BaseOneAgentStatus `json:",inline"`
}

// OneAgentAPM configures the Dynatrace OneAgent for application monitoring
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=oneagentapms,scope=Namespaced,categories=dynatrace
// +kubebuilder:printcolumn:name="ApiUrl",type=string,JSONPath=`.spec.apiUrl`
// +kubebuilder:printcolumn:name="Tokens",type=string,JSONPath=`.spec.tokens`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Dynatrace OneAgent Application Monitoring"
type OneAgentAPM struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OneAgentAPMSpec `json:"spec"`
	// +optional
	Status OneAgentAPMStatus `json:"status"`
}

// OneAgentAPMList contains a list of OneAgentAPM
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type OneAgentAPMList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OneAgentAPM `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OneAgentAPM{}, &OneAgentAPMList{})
}

// GetSpec returns the corresponding BaseOneAgentSpec for the instance's Spec.
func (oa *OneAgentAPM) GetSpec() *BaseOneAgentSpec {
	return &oa.Spec.BaseOneAgentSpec
}

// GetStatus returns the corresponding BaseOneAgentStatus for the instance's Status.
func (oa *OneAgentAPM) GetStatus() *BaseOneAgentStatus {
	return &oa.Status.BaseOneAgentStatus
}

var _ BaseOneAgent = &OneAgentAPM{}
