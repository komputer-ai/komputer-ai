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

// AgentConfigSpec is the set of agent settings a user can configure. It is
// inlined into KomputerAgentSpec and into every resource that creates agents
// from a template (KomputerSchedule), so those templates stay in parity with
// the agent spec by construction rather than by hand-copying fields.
//
// Fields the *caller* owns rather than the user — Instructions,
// InternalSystemPrompt, OfficeManager — deliberately live on KomputerAgentSpec
// instead, since a template's parent supplies them.
type AgentConfigSpec struct {
	// TemplateRef is the name of the KomputerAgentTemplate to use.
	// +kubebuilder:default="default"
	TemplateRef string `json:"templateRef,omitempty"`
	// SystemPrompt is a custom system prompt provided by the user, appended to the internal prompt.
	// +optional
	SystemPrompt string `json:"systemPrompt,omitempty"`
	// Model is the Claude model to use.
	// +kubebuilder:default="claude-opus-5-5"
	Model string `json:"model,omitempty"`
	// Effort controls the Claude Agent SDK's reasoning effort: "low", "medium",
	// "high", "xhigh", or "max". Empty (default) leaves it unset, so the CLI
	// uses the model's own default effort ("auto").
	// +kubebuilder:validation:Enum="";low;medium;high;xhigh;max
	// +optional
	Effort string `json:"effort,omitempty"`
	// Role is "manager" or "worker". Managers get orchestration tools.
	// Role is "manager" or "worker". Defaults to "manager" for top-level agents.
	// Sub-agents created by managers are explicitly set to "worker".
	// +kubebuilder:default="manager"
	// +kubebuilder:validation:Enum=worker;manager
	// +optional
	Role string `json:"role,omitempty"`
	// Secrets is a list of K8s Secret names containing agent-specific secrets.
	// Each key in each secret is injected as an env var into the agent pod.
	// +optional
	Secrets []string `json:"secrets,omitempty"`
	// Skills is a list of KomputerSkill names to attach to this agent.
	// Names can be "name" (same namespace) or "namespace/name" (cross-namespace).
	// +optional
	Skills []string `json:"skills,omitempty"`
	// Memories is a list of KomputerMemory names to attach to this agent.
	// Names can be "name" (same namespace) or "namespace/name" (cross-namespace).
	// +optional
	Memories []string `json:"memories,omitempty"`
	// Connectors is a list of KomputerConnector names to attach to this agent.
	// Names can be "name" (same namespace) or "namespace/name" (cross-namespace).
	// +optional
	Connectors []string `json:"connectors,omitempty"`
	// AllowedTools restricts the agent to exactly these tools. When empty, the
	// default built-in tool set is used and all tools from attached connectors
	// are permitted.
	//
	// Setting this REPLACES the default set rather than extending it, so an
	// agent given only ["Read"] loses Bash, Write, Edit and the rest. Connector
	// tools are not auto-added either — list them explicitly, e.g.
	// "mcp__figma__*" for a whole connector or "mcp__figma__get_design_context"
	// for a single tool.
	// +optional
	AllowedTools []string `json:"allowedTools,omitempty"`
	// DisallowedTools removes these tools from the agent. Purely subtractive:
	// the default built-ins and all connector tools remain available except
	// what is named here. Takes precedence over AllowedTools.
	// Supports wildcards, e.g. "mcp__figma__*".
	// +optional
	DisallowedTools []string `json:"disallowedTools,omitempty"`
	// Lifecycle controls what happens after task completion.
	// Empty (default) keeps the pod running, "Sleep" deletes the pod but keeps the PVC,
	// "AutoDelete" deletes the entire agent after task completion, and "Archive" deletes
	// the pod and PVC but keeps the agent as a record (Phase=Archived). Without the S3
	// session store an archived agent can't run again; with it, a new task or a wake
	// brings it back with its session restored.
	// +kubebuilder:validation:Enum="";Sleep;AutoDelete;Archive
	// +optional
	Lifecycle AgentLifecycle `json:"lifecycle,omitempty"`
	// SleepTTL puts the agent to sleep (pod deleted, PVC preserved) once it has been
	// idle for this long. Idle is measured from Status.LastActivityAt, so the clock
	// only starts once a task has actually started, and any later task event or wake
	// resets it. Never fires while a task is in progress, and an agent that has never
	// run a task is never auto-slept (use DeleteTTL to reclaim those).
	// Unset (default) means the agent never auto-sleeps.
	// Overrides the template's sleepTTL when set.
	// +optional
	SleepTTL *metav1.Duration `json:"sleepTTL,omitempty"`
	// DeleteTTL deletes the entire agent (pod + PVC) once this long has elapsed since
	// metadata.creationTimestamp. This is an absolute lifetime cap: unlike SleepTTL it
	// does not reset on wake and applies in every phase, including Sleeping.
	// Unset (default) means the agent never auto-deletes.
	// Overrides the template's deleteTTL when set.
	// +optional
	DeleteTTL *metav1.Duration `json:"deleteTTL,omitempty"`
	// ArchiveTTL archives the agent (pod and PVC deleted, CR kept as a record) once it
	// has been idle for this long. Idle is measured exactly like SleepTTL — from
	// Status.LastActivityAt, reset by any task event or wake, never while a task is
	// in progress, and never on an agent that has never run a task — but it also runs
	// while the agent is Sleeping, so SleepTTL < ArchiveTTL sleeps first, then
	// archives. With the S3 session store it waits until the latest task is saved;
	// without it the archived agent can't run again. Not applied to squad members.
	// When set, Lifecycle=Archive waits for it instead of archiving right after the
	// task. Unset (default) means the agent never auto-archives.
	// Overrides the template's archiveTTL when set.
	// +optional
	ArchiveTTL *metav1.Duration `json:"archiveTTL,omitempty"`
	// TaskTimeout cancels the agent's running task once it has been running for this
	// long. The clock starts when a task starts (Status.TaskStartedAt) and never
	// resets — steering a task does not extend it — so this is a hard wall-clock cap
	// on a single task, not an idle timeout.
	//
	// Only the task is cancelled; the agent itself stays alive and any configured
	// Lifecycle (Sleep / AutoDelete) then applies as it would after any other task end.
	// Unset (default) means tasks run without a time limit.
	// Overrides the template's taskTimeout when set.
	// +optional
	TaskTimeout *metav1.Duration `json:"taskTimeout,omitempty"`
	// BudgetUSD is a SOFT lifetime spending cap across every task this agent ever
	// runs, as a decimal USD string (e.g. "10.00"). Measured against
	// Status.TotalCostUSD: once spent reaches or exceeds this, the API refuses to
	// start any NEW task for the agent. A task already in progress when the cap is
	// crossed is always allowed to finish, even if it overshoots — the cap only
	// gates starting the next one. Raising (or clearing) BudgetUSD unblocks the
	// agent again.
	//
	// A string rather than a float to match how the rest of the CRD represents USD
	// (see Status.TotalCostUSD) and avoid controller-gen's allowDangerousTypes for
	// floats. Unset (default) means no lifetime cap. Overrides the template's
	// budgetUSD when set.
	// +kubebuilder:validation:Pattern=`^[0-9]+(\.[0-9]+)?$`
	// +optional
	BudgetUSD string `json:"budgetUSD,omitempty"`
	// TaskBudgetUSD is a HARD per-task spending cap, as a decimal USD string (e.g.
	// "2.50"). Passed to the Claude Agent SDK as max_budget_usd for each task the
	// agent runs: the SDK itself stops the task the instant this is exceeded,
	// wherever it is mid-task, returning an error_max_budget_usd result. Unlike
	// BudgetUSD this is enforced inside the running task rather than by refusing to
	// start one.
	//
	// Unset (default) means tasks run without a per-task cap. Overrides the
	// template's taskBudgetUSD when set.
	// +kubebuilder:validation:Pattern=`^[0-9]+(\.[0-9]+)?$`
	// +optional
	TaskBudgetUSD string `json:"taskBudgetUSD,omitempty"`
	// Priority controls admission order when the template's maxConcurrentAgents
	// limit is reached. Higher number = admitted first (matches K8s PodPriority).
	// Ties broken by creationTimestamp (older first). Defaults to 0.
	// +kubebuilder:default=0
	// +optional
	Priority int32 `json:"priority,omitempty"`
	// PodSpec, when set, overrides the template's PodSpec for this agent.
	// Container fields are merged by name; non-zero fields from this PodSpec
	// override the template's container fields. Takes effect on next pod start
	// (existing pods are not mutated).
	// +optional
	PodSpec *corev1.PodSpec `json:"podSpec,omitempty"`
	// Storage, when set, overrides the template's storage settings for this agent.
	// Existing PVCs are expanded in place when the storage class supports it.
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`
	// Labels are user-defined key=value labels attached to this agent and
	// propagated to all child resources (Pod, PVC, ConfigMap, Service).
	// Keys starting with "komputer.ai/" are reserved for system labels and
	// should not be set directly through the API.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}
