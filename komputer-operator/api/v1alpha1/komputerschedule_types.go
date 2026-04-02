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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type SchedulePhase string

const (
	SchedulePhaseActive    SchedulePhase = "Active"
	SchedulePhaseSuspended SchedulePhase = "Suspended"
	SchedulePhaseError     SchedulePhase = "Error"
)

// ScheduleAgentSpec defines the template for creating an agent on each scheduled run.
type ScheduleAgentSpec struct {
	// +kubebuilder:default="claude-sonnet-4-6"
	// +optional
	Model string `json:"model,omitempty"`
	// +kubebuilder:validation:Enum="";Sleep;AutoDelete
	// +kubebuilder:default="Sleep"
	// +optional
	Lifecycle AgentLifecycle `json:"lifecycle,omitempty"`
	// +kubebuilder:default="worker"
	// +optional
	Role string `json:"role,omitempty"`
	// +kubebuilder:default="default"
	// +optional
	TemplateRef string `json:"templateRef,omitempty"`
	// +optional
	Secrets []string `json:"secrets,omitempty"`
}

type KomputerScheduleSpec struct {
	// Schedule is a cron expression (5-field: min hour dom month dow).
	Schedule string `json:"schedule"`
	// Instructions is the task to run on each scheduled tick. Required.
	Instructions string `json:"instructions"`
	// Timezone is an IANA timezone for interpreting the cron schedule. Defaults to UTC.
	// +optional
	Timezone string `json:"timezone,omitempty"`
	// AutoDelete deletes this schedule CR after the first successful run.
	// +optional
	AutoDelete bool `json:"autoDelete,omitempty"`
	// KeepAgents, when true, removes the ownerReference before auto-deleting so agents survive.
	// +optional
	KeepAgents bool `json:"keepAgents,omitempty"`
	// Suspended pauses the schedule without deleting it.
	// +optional
	Suspended bool `json:"suspended,omitempty"`
	// AgentName references an existing KomputerAgent to trigger on schedule.
	// +optional
	AgentName string `json:"agentName,omitempty"`
	// Agent defines the template for creating a new agent. Used when AgentName is empty.
	// +optional
	Agent *ScheduleAgentSpec `json:"agent,omitempty"`
}

type KomputerScheduleStatus struct {
	Phase SchedulePhase `json:"phase,omitempty"`
	// +optional
	LastRunTime *metav1.Time `json:"lastRunTime,omitempty"`
	// +optional
	NextRunTime    *metav1.Time `json:"nextRunTime,omitempty"`
	RunCount       int          `json:"runCount,omitempty"`
	SuccessfulRuns int          `json:"successfulRuns,omitempty"`
	FailedRuns     int          `json:"failedRuns,omitempty"`
	TotalCostUSD   string       `json:"totalCostUSD,omitempty"`
	LastRunCostUSD string       `json:"lastRunCostUSD,omitempty"`
	// TotalTokens is the cumulative number of tokens consumed across all scheduled runs.
	// +optional
	TotalTokens int64 `json:"totalTokens,omitempty"`
	// LastRunTokens is the token count for the most recent scheduled run.
	// +optional
	LastRunTokens int64  `json:"lastRunTokens,omitempty"`
	AgentName     string `json:"agentName,omitempty"`
	LastRunStatus string `json:"lastRunStatus,omitempty"`
	Message       string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Agent",type=string,JSONPath=`.status.agentName`
// +kubebuilder:printcolumn:name="Runs",type=integer,JSONPath=`.status.runCount`
// +kubebuilder:printcolumn:name="Cost",type=string,JSONPath=`.status.totalCostUSD`
// +kubebuilder:printcolumn:name="Next",type=string,JSONPath=`.status.nextRunTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type KomputerSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KomputerScheduleSpec   `json:"spec,omitempty"`
	Status            KomputerScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KomputerScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KomputerSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KomputerSchedule{}, &KomputerScheduleList{})
}
