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
	// +kubebuilder:default="claude-sonnet-4-6"
	Model string `json:"model,omitempty"`
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
	// "AutoDelete" deletes the entire agent after task completion.
	// +kubebuilder:validation:Enum="";Sleep;AutoDelete
	// +optional
	Lifecycle AgentLifecycle `json:"lifecycle,omitempty"`
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
