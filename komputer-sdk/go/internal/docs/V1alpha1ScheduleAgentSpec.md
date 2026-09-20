# V1alpha1ScheduleAgentSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AllowedTools** | Pointer to **[]string** | AllowedTools restricts the agent to exactly these tools. When empty, the default built-in tool set is used and all tools from attached connectors are permitted.  Setting this REPLACES the default set rather than extending it, so an agent given only [\&quot;Read\&quot;] loses Bash, Write, Edit and the rest. Connector tools are not auto-added either — list them explicitly, e.g. \&quot;mcp__figma__*\&quot; for a whole connector or \&quot;mcp__figma__get_design_context\&quot; for a single tool. +optional | [optional] 
**Connectors** | Pointer to **[]string** | Connectors is a list of KomputerConnector names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**DeleteTTL** | Pointer to [**V1Duration**](V1Duration.md) | DeleteTTL deletes the entire agent (pod + PVC) once this long has elapsed since metadata.creationTimestamp. This is an absolute lifetime cap: unlike SleepTTL it does not reset on wake and applies in every phase, including Sleeping. Unset (default) means the agent never auto-deletes. Overrides the template&#39;s deleteTTL when set. +optional | [optional] 
**DisallowedTools** | Pointer to **[]string** | DisallowedTools removes these tools from the agent. Purely subtractive: the default built-ins and all connector tools remain available except what is named here. Takes precedence over AllowedTools. Supports wildcards, e.g. \&quot;mcp__figma__*\&quot;. +optional | [optional] 
**Labels** | Pointer to **map[string]string** | Labels are user-defined key&#x3D;value labels attached to this agent and propagated to all child resources (Pod, PVC, ConfigMap, Service). Keys starting with \&quot;komputer.ai/\&quot; are reserved for system labels and should not be set directly through the API. +optional | [optional] 
**Lifecycle** | Pointer to [**V1alpha1AgentLifecycle**](V1alpha1AgentLifecycle.md) | Lifecycle controls what happens after task completion. Empty (default) keeps the pod running, \&quot;Sleep\&quot; deletes the pod but keeps the PVC, \&quot;AutoDelete\&quot; deletes the entire agent after task completion. +kubebuilder:validation:Enum&#x3D;\&quot;\&quot;;Sleep;AutoDelete +optional | [optional] 
**Memories** | Pointer to **[]string** | Memories is a list of KomputerMemory names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**Model** | Pointer to **string** | Model is the Claude model to use. +kubebuilder:default&#x3D;\&quot;claude-sonnet-4-6\&quot; | [optional] 
**PodSpec** | Pointer to [**V1PodSpec**](V1PodSpec.md) | PodSpec, when set, overrides the template&#39;s PodSpec for this agent. Container fields are merged by name; non-zero fields from this PodSpec override the template&#39;s container fields. Takes effect on next pod start (existing pods are not mutated). +optional | [optional] 
**Priority** | Pointer to **int32** | Priority controls admission order when the template&#39;s maxConcurrentAgents limit is reached. Higher number &#x3D; admitted first (matches K8s PodPriority). Ties broken by creationTimestamp (older first). Defaults to 0. +kubebuilder:default&#x3D;0 +optional | [optional] 
**Role** | Pointer to **string** | Role is \&quot;manager\&quot; or \&quot;worker\&quot;. Managers get orchestration tools. Role is \&quot;manager\&quot; or \&quot;worker\&quot;. Defaults to \&quot;manager\&quot; for top-level agents. Sub-agents created by managers are explicitly set to \&quot;worker\&quot;. +kubebuilder:default&#x3D;\&quot;manager\&quot; +kubebuilder:validation:Enum&#x3D;worker;manager +optional | [optional] 
**Secrets** | Pointer to **[]string** | Secrets is a list of K8s Secret names containing agent-specific secrets. Each key in each secret is injected as an env var into the agent pod. +optional | [optional] 
**Skills** | Pointer to **[]string** | Skills is a list of KomputerSkill names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**SleepTTL** | Pointer to [**V1Duration**](V1Duration.md) | SleepTTL puts the agent to sleep (pod deleted, PVC preserved) once it has been idle for this long. Idle is measured from Status.LastActivityAt, so the clock only starts once a task has actually started, and any later task event or wake resets it. Never fires while a task is in progress, and an agent that has never run a task is never auto-slept (use DeleteTTL to reclaim those). Unset (default) means the agent never auto-sleeps. Overrides the template&#39;s sleepTTL when set. +optional | [optional] 
**Storage** | Pointer to [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) | Storage, when set, overrides the template&#39;s storage settings for this agent. Existing PVCs are expanded in place when the storage class supports it. +optional | [optional] 
**SystemPrompt** | Pointer to **string** | SystemPrompt is a custom system prompt provided by the user, appended to the internal prompt. +optional | [optional] 
**TemplateRef** | Pointer to **string** | TemplateRef is the name of the KomputerAgentTemplate to use. +kubebuilder:default&#x3D;\&quot;default\&quot; | [optional] 

## Methods

### NewV1alpha1ScheduleAgentSpec

`func NewV1alpha1ScheduleAgentSpec() *V1alpha1ScheduleAgentSpec`

NewV1alpha1ScheduleAgentSpec instantiates a new V1alpha1ScheduleAgentSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1alpha1ScheduleAgentSpecWithDefaults

`func NewV1alpha1ScheduleAgentSpecWithDefaults() *V1alpha1ScheduleAgentSpec`

NewV1alpha1ScheduleAgentSpecWithDefaults instantiates a new V1alpha1ScheduleAgentSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAllowedTools

`func (o *V1alpha1ScheduleAgentSpec) GetAllowedTools() []string`

GetAllowedTools returns the AllowedTools field if non-nil, zero value otherwise.

### GetAllowedToolsOk

`func (o *V1alpha1ScheduleAgentSpec) GetAllowedToolsOk() (*[]string, bool)`

GetAllowedToolsOk returns a tuple with the AllowedTools field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowedTools

`func (o *V1alpha1ScheduleAgentSpec) SetAllowedTools(v []string)`

SetAllowedTools sets AllowedTools field to given value.

### HasAllowedTools

`func (o *V1alpha1ScheduleAgentSpec) HasAllowedTools() bool`

HasAllowedTools returns a boolean if a field has been set.

### GetConnectors

`func (o *V1alpha1ScheduleAgentSpec) GetConnectors() []string`

GetConnectors returns the Connectors field if non-nil, zero value otherwise.

### GetConnectorsOk

`func (o *V1alpha1ScheduleAgentSpec) GetConnectorsOk() (*[]string, bool)`

GetConnectorsOk returns a tuple with the Connectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConnectors

`func (o *V1alpha1ScheduleAgentSpec) SetConnectors(v []string)`

SetConnectors sets Connectors field to given value.

### HasConnectors

`func (o *V1alpha1ScheduleAgentSpec) HasConnectors() bool`

HasConnectors returns a boolean if a field has been set.

### GetDeleteTTL

`func (o *V1alpha1ScheduleAgentSpec) GetDeleteTTL() V1Duration`

GetDeleteTTL returns the DeleteTTL field if non-nil, zero value otherwise.

### GetDeleteTTLOk

`func (o *V1alpha1ScheduleAgentSpec) GetDeleteTTLOk() (*V1Duration, bool)`

GetDeleteTTLOk returns a tuple with the DeleteTTL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeleteTTL

`func (o *V1alpha1ScheduleAgentSpec) SetDeleteTTL(v V1Duration)`

SetDeleteTTL sets DeleteTTL field to given value.

### HasDeleteTTL

`func (o *V1alpha1ScheduleAgentSpec) HasDeleteTTL() bool`

HasDeleteTTL returns a boolean if a field has been set.

### GetDisallowedTools

`func (o *V1alpha1ScheduleAgentSpec) GetDisallowedTools() []string`

GetDisallowedTools returns the DisallowedTools field if non-nil, zero value otherwise.

### GetDisallowedToolsOk

`func (o *V1alpha1ScheduleAgentSpec) GetDisallowedToolsOk() (*[]string, bool)`

GetDisallowedToolsOk returns a tuple with the DisallowedTools field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisallowedTools

`func (o *V1alpha1ScheduleAgentSpec) SetDisallowedTools(v []string)`

SetDisallowedTools sets DisallowedTools field to given value.

### HasDisallowedTools

`func (o *V1alpha1ScheduleAgentSpec) HasDisallowedTools() bool`

HasDisallowedTools returns a boolean if a field has been set.

### GetLabels

`func (o *V1alpha1ScheduleAgentSpec) GetLabels() map[string]string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *V1alpha1ScheduleAgentSpec) GetLabelsOk() (*map[string]string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *V1alpha1ScheduleAgentSpec) SetLabels(v map[string]string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *V1alpha1ScheduleAgentSpec) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetLifecycle

`func (o *V1alpha1ScheduleAgentSpec) GetLifecycle() V1alpha1AgentLifecycle`

GetLifecycle returns the Lifecycle field if non-nil, zero value otherwise.

### GetLifecycleOk

`func (o *V1alpha1ScheduleAgentSpec) GetLifecycleOk() (*V1alpha1AgentLifecycle, bool)`

GetLifecycleOk returns a tuple with the Lifecycle field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLifecycle

`func (o *V1alpha1ScheduleAgentSpec) SetLifecycle(v V1alpha1AgentLifecycle)`

SetLifecycle sets Lifecycle field to given value.

### HasLifecycle

`func (o *V1alpha1ScheduleAgentSpec) HasLifecycle() bool`

HasLifecycle returns a boolean if a field has been set.

### GetMemories

`func (o *V1alpha1ScheduleAgentSpec) GetMemories() []string`

GetMemories returns the Memories field if non-nil, zero value otherwise.

### GetMemoriesOk

`func (o *V1alpha1ScheduleAgentSpec) GetMemoriesOk() (*[]string, bool)`

GetMemoriesOk returns a tuple with the Memories field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemories

`func (o *V1alpha1ScheduleAgentSpec) SetMemories(v []string)`

SetMemories sets Memories field to given value.

### HasMemories

`func (o *V1alpha1ScheduleAgentSpec) HasMemories() bool`

HasMemories returns a boolean if a field has been set.

### GetModel

`func (o *V1alpha1ScheduleAgentSpec) GetModel() string`

GetModel returns the Model field if non-nil, zero value otherwise.

### GetModelOk

`func (o *V1alpha1ScheduleAgentSpec) GetModelOk() (*string, bool)`

GetModelOk returns a tuple with the Model field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetModel

`func (o *V1alpha1ScheduleAgentSpec) SetModel(v string)`

SetModel sets Model field to given value.

### HasModel

`func (o *V1alpha1ScheduleAgentSpec) HasModel() bool`

HasModel returns a boolean if a field has been set.

### GetPodSpec

`func (o *V1alpha1ScheduleAgentSpec) GetPodSpec() V1PodSpec`

GetPodSpec returns the PodSpec field if non-nil, zero value otherwise.

### GetPodSpecOk

`func (o *V1alpha1ScheduleAgentSpec) GetPodSpecOk() (*V1PodSpec, bool)`

GetPodSpecOk returns a tuple with the PodSpec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodSpec

`func (o *V1alpha1ScheduleAgentSpec) SetPodSpec(v V1PodSpec)`

SetPodSpec sets PodSpec field to given value.

### HasPodSpec

`func (o *V1alpha1ScheduleAgentSpec) HasPodSpec() bool`

HasPodSpec returns a boolean if a field has been set.

### GetPriority

`func (o *V1alpha1ScheduleAgentSpec) GetPriority() int32`

GetPriority returns the Priority field if non-nil, zero value otherwise.

### GetPriorityOk

`func (o *V1alpha1ScheduleAgentSpec) GetPriorityOk() (*int32, bool)`

GetPriorityOk returns a tuple with the Priority field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriority

`func (o *V1alpha1ScheduleAgentSpec) SetPriority(v int32)`

SetPriority sets Priority field to given value.

### HasPriority

`func (o *V1alpha1ScheduleAgentSpec) HasPriority() bool`

HasPriority returns a boolean if a field has been set.

### GetRole

`func (o *V1alpha1ScheduleAgentSpec) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *V1alpha1ScheduleAgentSpec) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *V1alpha1ScheduleAgentSpec) SetRole(v string)`

SetRole sets Role field to given value.

### HasRole

`func (o *V1alpha1ScheduleAgentSpec) HasRole() bool`

HasRole returns a boolean if a field has been set.

### GetSecrets

`func (o *V1alpha1ScheduleAgentSpec) GetSecrets() []string`

GetSecrets returns the Secrets field if non-nil, zero value otherwise.

### GetSecretsOk

`func (o *V1alpha1ScheduleAgentSpec) GetSecretsOk() (*[]string, bool)`

GetSecretsOk returns a tuple with the Secrets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecrets

`func (o *V1alpha1ScheduleAgentSpec) SetSecrets(v []string)`

SetSecrets sets Secrets field to given value.

### HasSecrets

`func (o *V1alpha1ScheduleAgentSpec) HasSecrets() bool`

HasSecrets returns a boolean if a field has been set.

### GetSkills

`func (o *V1alpha1ScheduleAgentSpec) GetSkills() []string`

GetSkills returns the Skills field if non-nil, zero value otherwise.

### GetSkillsOk

`func (o *V1alpha1ScheduleAgentSpec) GetSkillsOk() (*[]string, bool)`

GetSkillsOk returns a tuple with the Skills field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSkills

`func (o *V1alpha1ScheduleAgentSpec) SetSkills(v []string)`

SetSkills sets Skills field to given value.

### HasSkills

`func (o *V1alpha1ScheduleAgentSpec) HasSkills() bool`

HasSkills returns a boolean if a field has been set.

### GetSleepTTL

`func (o *V1alpha1ScheduleAgentSpec) GetSleepTTL() V1Duration`

GetSleepTTL returns the SleepTTL field if non-nil, zero value otherwise.

### GetSleepTTLOk

`func (o *V1alpha1ScheduleAgentSpec) GetSleepTTLOk() (*V1Duration, bool)`

GetSleepTTLOk returns a tuple with the SleepTTL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSleepTTL

`func (o *V1alpha1ScheduleAgentSpec) SetSleepTTL(v V1Duration)`

SetSleepTTL sets SleepTTL field to given value.

### HasSleepTTL

`func (o *V1alpha1ScheduleAgentSpec) HasSleepTTL() bool`

HasSleepTTL returns a boolean if a field has been set.

### GetStorage

`func (o *V1alpha1ScheduleAgentSpec) GetStorage() V1alpha1StorageSpec`

GetStorage returns the Storage field if non-nil, zero value otherwise.

### GetStorageOk

`func (o *V1alpha1ScheduleAgentSpec) GetStorageOk() (*V1alpha1StorageSpec, bool)`

GetStorageOk returns a tuple with the Storage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStorage

`func (o *V1alpha1ScheduleAgentSpec) SetStorage(v V1alpha1StorageSpec)`

SetStorage sets Storage field to given value.

### HasStorage

`func (o *V1alpha1ScheduleAgentSpec) HasStorage() bool`

HasStorage returns a boolean if a field has been set.

### GetSystemPrompt

`func (o *V1alpha1ScheduleAgentSpec) GetSystemPrompt() string`

GetSystemPrompt returns the SystemPrompt field if non-nil, zero value otherwise.

### GetSystemPromptOk

`func (o *V1alpha1ScheduleAgentSpec) GetSystemPromptOk() (*string, bool)`

GetSystemPromptOk returns a tuple with the SystemPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSystemPrompt

`func (o *V1alpha1ScheduleAgentSpec) SetSystemPrompt(v string)`

SetSystemPrompt sets SystemPrompt field to given value.

### HasSystemPrompt

`func (o *V1alpha1ScheduleAgentSpec) HasSystemPrompt() bool`

HasSystemPrompt returns a boolean if a field has been set.

### GetTemplateRef

`func (o *V1alpha1ScheduleAgentSpec) GetTemplateRef() string`

GetTemplateRef returns the TemplateRef field if non-nil, zero value otherwise.

### GetTemplateRefOk

`func (o *V1alpha1ScheduleAgentSpec) GetTemplateRefOk() (*string, bool)`

GetTemplateRefOk returns a tuple with the TemplateRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTemplateRef

`func (o *V1alpha1ScheduleAgentSpec) SetTemplateRef(v string)`

SetTemplateRef sets TemplateRef field to given value.

### HasTemplateRef

`func (o *V1alpha1ScheduleAgentSpec) HasTemplateRef() bool`

HasTemplateRef returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


