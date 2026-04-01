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

// KomputerSkillSpec defines the desired state of KomputerSkill.
type KomputerSkillSpec struct {
	// Description is a short description of when to use this skill.
	Description string `json:"description"`
	// Content is the skill's markdown body (instructions/prompt).
	Content string `json:"content"`
}

// KomputerSkillStatus defines the observed state of KomputerSkill.
type KomputerSkillStatus struct {
	// AttachedAgents is the number of agents that reference this skill.
	AttachedAgents int `json:"attachedAgents,omitempty"`
	// AgentNames is the list of agent names that reference this skill.
	// +optional
	AgentNames []string `json:"agentNames,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Description",type=string,JSONPath=`.spec.description`
// +kubebuilder:printcolumn:name="Agents",type=integer,JSONPath=`.status.attachedAgents`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// KomputerSkill is a reusable skill that can be attached to agents.
// Skills are written to the agent's filesystem as Claude SDK skill files.
type KomputerSkill struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KomputerSkillSpec   `json:"spec,omitempty"`
	Status KomputerSkillStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KomputerSkillList contains a list of KomputerSkill.
type KomputerSkillList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KomputerSkill `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KomputerSkill{}, &KomputerSkillList{})
}
