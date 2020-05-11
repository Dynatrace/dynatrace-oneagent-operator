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

// OneAgentIM configures the Dynatrace OneAgent for infrastructure monitoring
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=OneAgentIMs,scope=Namespaced,categories=dynatrace
// +kubebuilder:printcolumn:name="ApiUrl",type=string,JSONPath=`.spec.apiUrl`
// +kubebuilder:printcolumn:name="Tokens",type=string,JSONPath=`.spec.tokens`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Dynatrace OneAgent Infrastructure Monitoring"
type OneAgentIM struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OneAgentSpec `json:"spec"`
	// +optional
	Status OneAgentStatus `json:"status"`
}

// OneAgentIMList contains a list of OneAgentIM
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type OneAgentIMList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OneAgentIM `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OneAgentIM{}, &OneAgentIMList{})
}

// GetSpec returns the corresponding BaseOneAgentSpec for the instance's Spec.
func (oa *OneAgentIM) GetSpec() *BaseOneAgentSpec {
	return &oa.Spec.BaseOneAgentSpec
}

// GetStatus returns the corresponding BaseOneAgentStatus for the instance's Status.
func (oa *OneAgentIM) GetStatus() *BaseOneAgentStatus {
	return &oa.Status.BaseOneAgentStatus
}

func (oa *OneAgentIM) GetOneAgentSpec() *OneAgentSpec {
	return &oa.Spec
}

func (oa *OneAgentIM) GetOneAgentStatus() *OneAgentStatus {
	return &oa.Status
}

var _ BaseOneAgent = &OneAgentIM{}
