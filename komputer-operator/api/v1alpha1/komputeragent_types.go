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

// Condition type constants for KomputerAgentStatus.Conditions.
const (
	// ConditionSecretsMirrored indicates whether the operator has successfully
	// mirrored all required source secrets into the agent's namespace.
	// Owned by the operator.
	ConditionSecretsMirrored = "SecretsMirrored"
)

// KomputerAgentPhase represents the lifecycle phase of a KomputerAgent.
type KomputerAgentPhase string

const (
	AgentPhasePending   KomputerAgentPhase = "Pending"
	AgentPhaseRunning   KomputerAgentPhase = "Running"
	AgentPhaseSucceeded KomputerAgentPhase = "Succeeded"
	AgentPhaseFailed    KomputerAgentPhase = "Failed"
	AgentPhaseSleeping  KomputerAgentPhase = "Sleeping"
	AgentPhaseQueued    KomputerAgentPhase = "Queued"
)

// AgentLifecycle controls what happens after task completion.
type AgentLifecycle string

const (
	// AgentLifecycleDefault keeps the pod running after task completion.
	AgentLifecycleDefault AgentLifecycle = ""
	// AgentLifecycleSleep deletes the pod after task completion but keeps the PVC.
	// The agent wakes up when a new task is sent.
	AgentLifecycleSleep AgentLifecycle = "Sleep"
	// AgentLifecycleAutoDelete deletes the entire agent after task completion.
	AgentLifecycleAutoDelete AgentLifecycle = "AutoDelete"
)

// AgentTaskStatus represents whether the agent is actively working on a task.
type AgentTaskStatus string

const (
	AgentTaskComplete   AgentTaskStatus = "Complete"
	AgentTaskInProgress AgentTaskStatus = "InProgress"
	// AgentTaskCompacting is a transient sub-state of InProgress while Claude Code
	// is summarizing older conversation turns to free up context window space.
	// Cleared back to InProgress once the next non-compaction event arrives.
	AgentTaskCompacting AgentTaskStatus = "Compacting"
	AgentTaskError      AgentTaskStatus = "Error"
)

// KomputerAgentSpec defines the desired state of KomputerAgent.
type KomputerAgentSpec struct {
	// AgentConfigSpec holds every user-configurable agent setting. It is inlined
	// here and in ScheduleAgentSpec so the two stay in parity by construction.
	AgentConfigSpec `json:",inline"`
	// Instructions is the user's task for the Claude agent.
	Instructions string `json:"instructions"`
	// InternalSystemPrompt is the built-in system prompt set by the API (role prompt + memories).
	// +optional
	InternalSystemPrompt string `json:"internalSystemPrompt,omitempty"`
	// OfficeManager is the name of the manager agent that created this sub-agent.
	// When set, the operator creates/joins a KomputerOffice for the group.
	// +optional
	OfficeManager string `json:"officeManager,omitempty"`
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
	// LastActivityAt is when the agent last produced or received task activity.
	// First stamped when a task starts, then refreshed on every agent event and on
	// wake; it is the idle clock for SleepTTL. Managed by the API worker, not by the
	// operator. While unset the agent has never started a task, so its SleepTTL clock
	// is not running at all.
	// +optional
	LastActivityAt *metav1.Time `json:"lastActivityAt,omitempty"`
	// SessionID is the Claude session ID for conversation continuity.
	// Set by the API worker when a task completes, read by the agent on startup.
	// +optional
	SessionID string `json:"sessionId,omitempty"`
	// LastTaskCostUSD is the cost of the most recent task in USD.
	// +optional
	LastTaskCostUSD string `json:"lastTaskCostUSD,omitempty"`
	// TotalCostUSD is the cumulative cost of all tasks run by this agent.
	// +optional
	TotalCostUSD string `json:"totalCostUSD,omitempty"`
	// TotalTokens is the cumulative number of tokens (input + output) consumed by all tasks run by this agent.
	// +optional
	TotalTokens int64 `json:"totalTokens,omitempty"`
	// ModelContextWindow is the context window size (in tokens) of the model currently assigned to this agent.
	// Fetched from the Anthropic API after each task completion or model change.
	// +optional
	ModelContextWindow int64 `json:"modelContextWindow,omitempty"`
	// QueuePosition is the 1-based position in the template admission queue.
	// Set by the operator when Phase=Queued. 0 when not queued.
	// Owned by operator.
	// +optional
	QueuePosition int32 `json:"queuePosition,omitempty"`
	// QueueReason explains why the agent is queued.
	// Owned by operator.
	// +optional
	QueueReason string `json:"queueReason,omitempty"`
	// SleepExpiresAt is when the agent will be put to sleep by SleepTTL.
	// Recomputed from the idle clock on each reconcile; nil when SleepTTL is unset
	// or a task is currently in progress.
	// Owned by operator.
	// +optional
	SleepExpiresAt *metav1.Time `json:"sleepExpiresAt,omitempty"`
	// DeleteExpiresAt is when the agent will be deleted by DeleteTTL.
	// nil when DeleteTTL is unset.
	// Owned by operator.
	// +optional
	DeleteExpiresAt *metav1.Time `json:"deleteExpiresAt,omitempty"`
	// TaskStartedAt is when the current (or most recent) task started. Stamped by the
	// API worker when the agent transitions into an in-progress task status, and NOT
	// refreshed while that task continues — a steer leaves it alone, which is what
	// makes TaskTimeout a hard cap. It is never cleared, so on an idle agent it reads
	// as "when the last task started".
	// Managed by the API worker, not by the operator.
	// +optional
	TaskStartedAt *metav1.Time `json:"taskStartedAt,omitempty"`
	// TaskExpiresAt is when the running task will be cancelled by TaskTimeout.
	// nil when TaskTimeout is unset, no task is in progress, or the agent has no
	// running pod.
	// Owned by operator.
	// +optional
	TaskExpiresAt *metav1.Time `json:"taskExpiresAt,omitempty"`
	// Squad indicates the agent is managed by a KomputerSquad. When true, the squad
	// controller owns the agent's pod lifecycle; the agent controller skips reconciliation.
	// Phase, PodName, etc. continue to reflect the real pod state (set by the squad controller).
	// Owned by squad controller.
	// +optional
	Squad bool `json:"squad,omitempty"`
	// Port is the agent's HTTP server port inside its container. Defaults to 8000
	// for solo agents. For squad members, the squad controller assigns 8000 + index
	// so multiple members can co-exist in the same pod's network namespace.
	// Owned by the controller that owns the pod (agent for solo, squad for squad members).
	// +optional
	Port int32 `json:"port,omitempty"`
	// Conditions holds the latest observations about the agent's state.
	// Owned by the operator.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:annotations="argocd.argoproj.io/sync-options=ServerSideApply=true"
// +kubebuilder:metadata:annotations="argocd.argoproj.io/sync-wave=-1"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Task",type=string,JSONPath=`.status.taskStatus`
// +kubebuilder:printcolumn:name="Cost",type=string,JSONPath=`.status.totalCostUSD`
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.model`
// +kubebuilder:printcolumn:name="Queue",type=integer,JSONPath=`.status.queuePosition`
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
