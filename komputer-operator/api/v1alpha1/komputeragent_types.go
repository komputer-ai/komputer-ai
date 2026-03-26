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

// KomputerAgentPhase represents the lifecycle phase of a KomputerAgent.
type KomputerAgentPhase string

const (
	AgentPhasePending   KomputerAgentPhase = "Pending"
	AgentPhaseRunning   KomputerAgentPhase = "Running"
	AgentPhaseSucceeded KomputerAgentPhase = "Succeeded"
	AgentPhaseFailed    KomputerAgentPhase = "Failed"
)

// AgentTaskStatus represents whether the agent is actively working on a task.
type AgentTaskStatus string

const (
	AgentTaskIdle  AgentTaskStatus = "Idle"
	AgentTaskBusy  AgentTaskStatus = "Busy"
	AgentTaskError AgentTaskStatus = "Error"
)

// KomputerAgentSpec defines the desired state of KomputerAgent.
type KomputerAgentSpec struct {
	// TemplateRef is the name of the KomputerAgentTemplate to use.
	// +kubebuilder:default="default"
	TemplateRef string `json:"templateRef,omitempty"`
	// Instructions is the prompt/task for the Claude agent.
	Instructions string `json:"instructions"`
	// Model is the Claude model to use.
	// +kubebuilder:default="claude-sonnet-4-20250514"
	Model string `json:"model,omitempty"`
	// Role is "manager" or "worker". Managers get orchestration tools.
	// Role is "manager" or "worker". Defaults to "manager" for top-level agents.
	// Sub-agents created by managers are explicitly set to "worker".
	// +kubebuilder:default="manager"
	// +kubebuilder:validation:Enum=worker;manager
	// +optional
	Role string `json:"role,omitempty"`
}

// KomputerAgentStatus defines the observed state of KomputerAgent.
type KomputerAgentStatus struct {
	// Phase is the current lifecycle phase.
	Phase KomputerAgentPhase `json:"phase,omitempty"`
	// PodName is the name of the agent pod.
	PodName string `json:"podName,omitempty"`
	// PvcName is the name of the agent PVC.
	PvcName string `json:"pvcName,omitempty"`
	// StartTime is when the agent was started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// CompletionTime is when the agent finished.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	// Message is a human-readable status message.
	Message string `json:"message,omitempty"`
	// TaskStatus indicates whether the agent is actively working on a task.
	// Managed by the API worker based on Redis events, not by the operator.
	// +optional
	TaskStatus AgentTaskStatus `json:"taskStatus,omitempty"`
	// LastTaskMessage is the most recent event summary from the agent.
	// Managed by the API worker based on Redis events, not by the operator.
	// +optional
	LastTaskMessage string `json:"lastTaskMessage,omitempty"`
	// SessionID is the Claude session ID for conversation continuity.
	// Set by the API worker when a task completes, read by the agent on startup.
	// +optional
	SessionID string `json:"sessionId,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Task",type=string,JSONPath=`.status.taskStatus`
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.model`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// KomputerAgent is the Schema for the komputeragents API.
type KomputerAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KomputerAgentSpec   `json:"spec,omitempty"`
	Status KomputerAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KomputerAgentList contains a list of KomputerAgent.
type KomputerAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KomputerAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KomputerAgent{}, &KomputerAgentList{})
}
