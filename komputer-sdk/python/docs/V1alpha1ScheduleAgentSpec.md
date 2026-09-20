# V1alpha1ScheduleAgentSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**allowed_tools** | **List[str]** | AllowedTools restricts the agent to exactly these tools. When empty, the default built-in tool set is used and all tools from attached connectors are permitted.  Setting this REPLACES the default set rather than extending it, so an agent given only [\&quot;Read\&quot;] loses Bash, Write, Edit and the rest. Connector tools are not auto-added either — list them explicitly, e.g. \&quot;mcp__figma__*\&quot; for a whole connector or \&quot;mcp__figma__get_design_context\&quot; for a single tool. +optional | [optional] 
**connectors** | **List[str]** | Connectors is a list of KomputerConnector names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**delete_ttl** | [**V1Duration**](V1Duration.md) | DeleteTTL deletes the entire agent (pod + PVC) once this long has elapsed since metadata.creationTimestamp. This is an absolute lifetime cap: unlike SleepTTL it does not reset on wake and applies in every phase, including Sleeping. Unset (default) means the agent never auto-deletes. Overrides the template&#39;s deleteTTL when set. +optional | [optional] 
**disallowed_tools** | **List[str]** | DisallowedTools removes these tools from the agent. Purely subtractive: the default built-ins and all connector tools remain available except what is named here. Takes precedence over AllowedTools. Supports wildcards, e.g. \&quot;mcp__figma__*\&quot;. +optional | [optional] 
**labels** | **Dict[str, str]** | Labels are user-defined key&#x3D;value labels attached to this agent and propagated to all child resources (Pod, PVC, ConfigMap, Service). Keys starting with \&quot;komputer.ai/\&quot; are reserved for system labels and should not be set directly through the API. +optional | [optional] 
**lifecycle** | [**V1alpha1AgentLifecycle**](V1alpha1AgentLifecycle.md) | Lifecycle controls what happens after task completion. Empty (default) keeps the pod running, \&quot;Sleep\&quot; deletes the pod but keeps the PVC, \&quot;AutoDelete\&quot; deletes the entire agent after task completion. +kubebuilder:validation:Enum&#x3D;\&quot;\&quot;;Sleep;AutoDelete +optional | [optional] 
**memories** | **List[str]** | Memories is a list of KomputerMemory names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**model** | **str** | Model is the Claude model to use. +kubebuilder:default&#x3D;\&quot;claude-sonnet-4-6\&quot; | [optional] 
**pod_spec** | [**V1PodSpec**](V1PodSpec.md) | PodSpec, when set, overrides the template&#39;s PodSpec for this agent. Container fields are merged by name; non-zero fields from this PodSpec override the template&#39;s container fields. Takes effect on next pod start (existing pods are not mutated). +optional | [optional] 
**priority** | **int** | Priority controls admission order when the template&#39;s maxConcurrentAgents limit is reached. Higher number &#x3D; admitted first (matches K8s PodPriority). Ties broken by creationTimestamp (older first). Defaults to 0. +kubebuilder:default&#x3D;0 +optional | [optional] 
**role** | **str** | Role is \&quot;manager\&quot; or \&quot;worker\&quot;. Managers get orchestration tools. Role is \&quot;manager\&quot; or \&quot;worker\&quot;. Defaults to \&quot;manager\&quot; for top-level agents. Sub-agents created by managers are explicitly set to \&quot;worker\&quot;. +kubebuilder:default&#x3D;\&quot;manager\&quot; +kubebuilder:validation:Enum&#x3D;worker;manager +optional | [optional] 
**secrets** | **List[str]** | Secrets is a list of K8s Secret names containing agent-specific secrets. Each key in each secret is injected as an env var into the agent pod. +optional | [optional] 
**skills** | **List[str]** | Skills is a list of KomputerSkill names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**sleep_ttl** | [**V1Duration**](V1Duration.md) | SleepTTL puts the agent to sleep (pod deleted, PVC preserved) once it has been idle for this long. Idle is measured from Status.LastActivityAt, so the clock only starts once a task has actually started, and any later task event or wake resets it. Never fires while a task is in progress, and an agent that has never run a task is never auto-slept (use DeleteTTL to reclaim those). Unset (default) means the agent never auto-sleeps. Overrides the template&#39;s sleepTTL when set. +optional | [optional] 
**storage** | [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) | Storage, when set, overrides the template&#39;s storage settings for this agent. Existing PVCs are expanded in place when the storage class supports it. +optional | [optional] 
**system_prompt** | **str** | SystemPrompt is a custom system prompt provided by the user, appended to the internal prompt. +optional | [optional] 
**template_ref** | **str** | TemplateRef is the name of the KomputerAgentTemplate to use. +kubebuilder:default&#x3D;\&quot;default\&quot; | [optional] 

## Example

```python
from komputer_ai.models.v1alpha1_schedule_agent_spec import V1alpha1ScheduleAgentSpec

# TODO update the JSON string below
json = "{}"
# create an instance of V1alpha1ScheduleAgentSpec from a JSON string
v1alpha1_schedule_agent_spec_instance = V1alpha1ScheduleAgentSpec.from_json(json)
# print the JSON string representation of the object
print(V1alpha1ScheduleAgentSpec.to_json())

# convert the object into a dict
v1alpha1_schedule_agent_spec_dict = v1alpha1_schedule_agent_spec_instance.to_dict()
# create an instance of V1alpha1ScheduleAgentSpec from a dict
v1alpha1_schedule_agent_spec_from_dict = V1alpha1ScheduleAgentSpec.from_dict(v1alpha1_schedule_agent_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


