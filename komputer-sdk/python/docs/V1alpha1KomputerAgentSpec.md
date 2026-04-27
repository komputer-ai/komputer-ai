# V1alpha1KomputerAgentSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**connectors** | **List[str]** | Connectors is a list of KomputerConnector names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**instructions** | **str** | Instructions is the user&#39;s task for the Claude agent. | [optional] 
**internal_system_prompt** | **str** | InternalSystemPrompt is the built-in system prompt set by the API (role prompt + memories). +optional | [optional] 
**lifecycle** | [**V1alpha1AgentLifecycle**](V1alpha1AgentLifecycle.md) | Lifecycle controls what happens after task completion. Empty (default) keeps the pod running, \&quot;Sleep\&quot; deletes the pod but keeps the PVC, \&quot;AutoDelete\&quot; deletes the entire agent after task completion. +kubebuilder:validation:Enum&#x3D;\&quot;\&quot;;Sleep;AutoDelete +optional | [optional] 
**memories** | **List[str]** | Memories is a list of KomputerMemory names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**model** | **str** | Model is the Claude model to use. +kubebuilder:default&#x3D;\&quot;claude-sonnet-4-6\&quot; | [optional] 
**office_manager** | **str** | OfficeManager is the name of the manager agent that created this sub-agent. When set, the operator creates/joins a KomputerOffice for the group. +optional | [optional] 
**pod_spec** | [**V1PodSpec**](V1PodSpec.md) | PodSpec, when set, overrides the template&#39;s PodSpec for this agent. Container fields are merged by name; non-zero fields from this PodSpec override the template&#39;s container fields. Takes effect on next pod start (existing pods are not mutated). +optional | [optional] 
**priority** | **int** | Priority controls admission order when the template&#39;s maxConcurrentAgents limit is reached. Higher number &#x3D; admitted first (matches K8s PodPriority). Ties broken by creationTimestamp (older first). Defaults to 0. +kubebuilder:default&#x3D;0 +optional | [optional] 
**role** | **str** | Role is \&quot;manager\&quot; or \&quot;worker\&quot;. Managers get orchestration tools. Role is \&quot;manager\&quot; or \&quot;worker\&quot;. Defaults to \&quot;manager\&quot; for top-level agents. Sub-agents created by managers are explicitly set to \&quot;worker\&quot;. +kubebuilder:default&#x3D;\&quot;manager\&quot; +kubebuilder:validation:Enum&#x3D;worker;manager +optional | [optional] 
**secrets** | **List[str]** | Secrets is a list of K8s Secret names containing agent-specific secrets. Each key in each secret is injected as an env var into the agent pod. +optional | [optional] 
**skills** | **List[str]** | Skills is a list of KomputerSkill names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**storage** | [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) | Storage, when set, overrides the template&#39;s storage settings for this agent. Existing PVCs are expanded in place when the storage class supports it. +optional | [optional] 
**system_prompt** | **str** | SystemPrompt is a custom system prompt provided by the user, appended to the internal prompt. +optional | [optional] 
**template_ref** | **str** | TemplateRef is the name of the KomputerAgentTemplate to use. +kubebuilder:default&#x3D;\&quot;default\&quot; | [optional] 

## Example

```python
from komputer_ai.models.v1alpha1_komputer_agent_spec import V1alpha1KomputerAgentSpec

# TODO update the JSON string below
json = "{}"
# create an instance of V1alpha1KomputerAgentSpec from a JSON string
v1alpha1_komputer_agent_spec_instance = V1alpha1KomputerAgentSpec.from_json(json)
# print the JSON string representation of the object
print(V1alpha1KomputerAgentSpec.to_json())

# convert the object into a dict
v1alpha1_komputer_agent_spec_dict = v1alpha1_komputer_agent_spec_instance.to_dict()
# create an instance of V1alpha1KomputerAgentSpec from a dict
v1alpha1_komputer_agent_spec_from_dict = V1alpha1KomputerAgentSpec.from_dict(v1alpha1_komputer_agent_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


