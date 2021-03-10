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

// +kubebuilder:object:generate:=false
// BaseOneAgent is implemented by OneAgent CRD types to ease access to common fields among all of them.
type BaseOneAgent interface {
	metav1.Object
	GetSpec() *BaseOneAgentSpec
	GetStatus() *BaseOneAgentStatus
}

// BaseOneAgentSpec includes credentials common to the other OneAgent CRDs
type BaseOneAgentSpec struct {
	// Location of the Dynatrace API to connect to, including your specific environment ID
	// +kubebuilder:validation:Required
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="API URL"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	APIURL string `json:"apiUrl"`

	// Credentials for the OneAgent to connect back to Dynatrace.
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="API and PaaS Tokens"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Tokens string `json:"tokens,omitempty"`

	// Disable certificate validation checks for installer download and API communication
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Skip Certificate Check"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	SkipCertCheck bool `json:"skipCertCheck,omitempty"`

	// If enabled, Istio on the cluster will be configured automatically to allow access to the Dynatrace environment
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Enable Istio automatic management"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	EnableIstio bool `json:"enableIstio,omitempty"`

	// Optional: Set custom proxy settings either directly or from a secret with the field 'proxy'
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Proxy"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Proxy *OneAgentProxy `json:"proxy,omitempty"`

	// Optional: Adds custom RootCAs from a configmap
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="TrustedCAs"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	TrustedCAs string `json:"trustedCAs,omitempty"`

	// Optional: Adds the OneAgent to the given NetworkZone
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="NetworkZone"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	NetworkZone string `json:"networkZone,omitempty"`

	// Defines if you want to use the immutable image or the installer
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Use immutable image"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	UseImmutableImage bool `json:"useImmutableImage,omitempty"`
}

// BaseOneAgentStatus defines common files used by OneAgent CRDs
type BaseOneAgentStatus struct {
	// UpdatedTimestamp indicates when the instance was last updated
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Last Updated"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`

	// Conditions includes status about the current state of the instance
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastAPITokenProbeTimestamp tracks when the last request for the API token validity was sent
	LastAPITokenProbeTimestamp *metav1.Time `json:"lastAPITokenProbeTimestamp,omitempty"`

	// LastPaaSTokenProbeTimestamp tracks when the last request for the PaaS token validity was sent
	LastPaaSTokenProbeTimestamp *metav1.Time `json:"lastPaaSTokenProbeTimestamp,omitempty"`

	// EnvironmentID contains the environment ID corresponding to the API URL
	EnvironmentID string `json:"environmentID,omitempty"`

	// Credentials used for the OneAgent to connect back to Dynatrace.
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="API and PaaS Tokens"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:text"
	Tokens string `json:"tokens,omitempty"`

	// UseImmutableImage is set when an immutable image is currently in use
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="Using immutable image"
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	UseImmutableImage bool `json:"useImmutableImage,omitempty"`
}

type OneAgentProxy struct {
	Value     string `json:"value,omitempty"`
	ValueFrom string `json:"valueFrom,omitempty"`
}

const (
	// APITokenConditionType identifies the API Token validity condition
	APITokenConditionType string = "APIToken"

	// PaaSTokenConditionType identifies the PaaS Token validity condition
	PaaSTokenConditionType string = "PaaSToken"
)

// Possible reasons for ApiToken and PaaSToken conditions
const (
	// ReasonTokenReady is set when a token has passed verifications
	ReasonTokenReady string = "TokenReady"

	// ReasonTokenSecretNotFound is set when the referenced secret can't be found
	ReasonTokenSecretNotFound string = "TokenSecretNotFound"

	// ReasonTokenMissing is set when the field is missing on the secret
	ReasonTokenMissing string = "TokenMissing"

	// ReasonTokenUnauthorized is set when a token is unauthorized to query the Dynatrace API
	ReasonTokenUnauthorized string = "TokenUnauthorized"

	// ReasonTokenScopeMissing is set when the token is missing the required scope for the Dynatrace API
	ReasonTokenScopeMissing string = "TokenScopeMissing"

	// ReasonTokenError is set when an unknown error has been found when verifying the token
	ReasonTokenError string = "TokenError"
)
