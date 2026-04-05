/*
Copyright 2026.

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

// KomputerConnectorSpec defines the desired state of KomputerConnector.
type KomputerConnectorSpec struct {
	// Type is the MCP transport type. Currently only "remote" (HTTP/SSE) is supported.
	// +kubebuilder:validation:Enum=remote
	Type string `json:"type"`
	// Service is the service identifier (e.g. "github", "atlassian", "gmail").
	// Used for display purposes and marketplace matching.
	Service string `json:"service"`
	// DisplayName is a human-readable name for this connector instance.
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// URL is the remote MCP server endpoint.
	URL string `json:"url"`
	// AuthSecretKeyRef references a K8s Secret key containing the auth token.
	// +optional
	AuthSecretKeyRef *corev1.SecretKeySelector `json:"authSecretKeyRef,omitempty"`
	// AuthType is the authentication method: "token" (default, static Bearer token) or "oauth".
	// +kubebuilder:validation:Enum=token;oauth
	// +kubebuilder:default=token
	// +optional
	AuthType string `json:"authType,omitempty"`
}

// KomputerConnectorStatus defines the observed state of KomputerConnector.
type KomputerConnectorStatus struct {
	// AttachedAgents is the number of agents using this connector.
	AttachedAgents int `json:"attachedAgents,omitempty"`
	// AgentNames is the list of agent names using this connector.
	// +optional
	AgentNames []string `json:"agentNames,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Service",type=string,JSONPath=`.spec.service`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.url`
// +kubebuilder:printcolumn:name="Agents",type=integer,JSONPath=`.status.attachedAgents`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// KomputerConnector is a remote MCP server connection that can be attached to agents.
type KomputerConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KomputerConnectorSpec   `json:"spec,omitempty"`
	Status KomputerConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KomputerConnectorList contains a list of KomputerConnector.
type KomputerConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KomputerConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KomputerConnector{}, &KomputerConnectorList{})
}
