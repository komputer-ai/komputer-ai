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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KomputerMemorySpec defines the desired state of KomputerMemory.
type KomputerMemorySpec struct {
	// Content is the memory/knowledge text to attach to agents.
	Content string `json:"content"`
	// Description is a short human-readable description of what this memory contains.
	// +optional
	Description string `json:"description,omitempty"`
}

// KomputerMemoryStatus defines the observed state of KomputerMemory.
type KomputerMemoryStatus struct {
	// AttachedAgents is the number of agents that reference this memory.
	AttachedAgents int `json:"attachedAgents,omitempty"`
	// AgentNames is the list of agent names that reference this memory.
	// +optional
	AgentNames []string `json:"agentNames,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:annotations="argocd.argoproj.io/sync-options=ServerSideApply=true"
// +kubebuilder:metadata:annotations="argocd.argoproj.io/sync-wave=-1"
// +kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`
// +kubebuilder:printcolumn:name="Agents",type=integer,JSONPath=`.status.attachedAgents`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// KomputerMemory is a persistent knowledge resource that can be attached to agents.
// Agents reference memories by name in their spec. Memory content is injected into
// the agent's system prompt as contextual knowledge.
type KomputerMemory struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KomputerMemorySpec   `json:"spec,omitempty"`
	Status KomputerMemoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KomputerMemoryList contains a list of KomputerMemory.
type KomputerMemoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KomputerMemory `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KomputerMemory{}, &KomputerMemoryList{})
}
