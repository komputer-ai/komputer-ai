# V1alpha1KomputerAgentSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Connectors** | Pointer to **[]string** | Connectors is a list of KomputerConnector names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**Instructions** | Pointer to **string** | Instructions is the user&#39;s task for the Claude agent. | [optional] 
**InternalSystemPrompt** | Pointer to **string** | InternalSystemPrompt is the built-in system prompt set by the API (role prompt + memories). +optional | [optional] 
**Lifecycle** | Pointer to [**V1alpha1AgentLifecycle**](V1alpha1AgentLifecycle.md) | Lifecycle controls what happens after task completion. Empty (default) keeps the pod running, \&quot;Sleep\&quot; deletes the pod but keeps the PVC, \&quot;AutoDelete\&quot; deletes the entire agent after task completion. +kubebuilder:validation:Enum&#x3D;\&quot;\&quot;;Sleep;AutoDelete +optional | [optional] 
**Memories** | Pointer to **[]string** | Memories is a list of KomputerMemory names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**Model** | Pointer to **string** | Model is the Claude model to use. +kubebuilder:default&#x3D;\&quot;claude-sonnet-4-6\&quot; | [optional] 
**OfficeManager** | Pointer to **string** | OfficeManager is the name of the manager agent that created this sub-agent. When set, the operator creates/joins a KomputerOffice for the group. +optional | [optional] 
**PodSpec** | Pointer to [**V1PodSpec**](V1PodSpec.md) | PodSpec, when set, overrides the template&#39;s PodSpec for this agent. Container fields are merged by name; non-zero fields from this PodSpec override the template&#39;s container fields. Takes effect on next pod start (existing pods are not mutated). +optional | [optional] 
**Priority** | Pointer to **int32** | Priority controls admission order when the template&#39;s maxConcurrentAgents limit is reached. Higher number &#x3D; admitted first (matches K8s PodPriority). Ties broken by creationTimestamp (older first). Defaults to 0. +kubebuilder:default&#x3D;0 +optional | [optional] 
**Role** | Pointer to **string** | Role is \&quot;manager\&quot; or \&quot;worker\&quot;. Managers get orchestration tools. Role is \&quot;manager\&quot; or \&quot;worker\&quot;. Defaults to \&quot;manager\&quot; for top-level agents. Sub-agents created by managers are explicitly set to \&quot;worker\&quot;. +kubebuilder:default&#x3D;\&quot;manager\&quot; +kubebuilder:validation:Enum&#x3D;worker;manager +optional | [optional] 
**Secrets** | Pointer to **[]string** | Secrets is a list of K8s Secret names containing agent-specific secrets. Each key in each secret is injected as an env var into the agent pod. +optional | [optional] 
**Skills** | Pointer to **[]string** | Skills is a list of KomputerSkill names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**Storage** | Pointer to [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) | Storage, when set, overrides the template&#39;s storage settings for this agent. Existing PVCs are expanded in place when the storage class supports it. +optional | [optional] 
**SystemPrompt** | Pointer to **string** | SystemPrompt is a custom system prompt provided by the user, appended to the internal prompt. +optional | [optional] 
**TemplateRef** | Pointer to **string** | TemplateRef is the name of the KomputerAgentTemplate to use. +kubebuilder:default&#x3D;\&quot;default\&quot; | [optional] 

## Methods

### NewV1alpha1KomputerAgentSpec

`func NewV1alpha1KomputerAgentSpec() *V1alpha1KomputerAgentSpec`

NewV1alpha1KomputerAgentSpec instantiates a new V1alpha1KomputerAgentSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1alpha1KomputerAgentSpecWithDefaults

`func NewV1alpha1KomputerAgentSpecWithDefaults() *V1alpha1KomputerAgentSpec`

NewV1alpha1KomputerAgentSpecWithDefaults instantiates a new V1alpha1KomputerAgentSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConnectors

`func (o *V1alpha1KomputerAgentSpec) GetConnectors() []string`

GetConnectors returns the Connectors field if non-nil, zero value otherwise.

### GetConnectorsOk

`func (o *V1alpha1KomputerAgentSpec) GetConnectorsOk() (*[]string, bool)`

GetConnectorsOk returns a tuple with the Connectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConnectors

`func (o *V1alpha1KomputerAgentSpec) SetConnectors(v []string)`

SetConnectors sets Connectors field to given value.

### HasConnectors

`func (o *V1alpha1KomputerAgentSpec) HasConnectors() bool`

HasConnectors returns a boolean if a field has been set.

### GetInstructions

`func (o *V1alpha1KomputerAgentSpec) GetInstructions() string`

GetInstructions returns the Instructions field if non-nil, zero value otherwise.

### GetInstructionsOk

`func (o *V1alpha1KomputerAgentSpec) GetInstructionsOk() (*string, bool)`

GetInstructionsOk returns a tuple with the Instructions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstructions

`func (o *V1alpha1KomputerAgentSpec) SetInstructions(v string)`

SetInstructions sets Instructions field to given value.

### HasInstructions

`func (o *V1alpha1KomputerAgentSpec) HasInstructions() bool`

HasInstructions returns a boolean if a field has been set.

### GetInternalSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) GetInternalSystemPrompt() string`

GetInternalSystemPrompt returns the InternalSystemPrompt field if non-nil, zero value otherwise.

### GetInternalSystemPromptOk

`func (o *V1alpha1KomputerAgentSpec) GetInternalSystemPromptOk() (*string, bool)`

GetInternalSystemPromptOk returns a tuple with the InternalSystemPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInternalSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) SetInternalSystemPrompt(v string)`

SetInternalSystemPrompt sets InternalSystemPrompt field to given value.

### HasInternalSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) HasInternalSystemPrompt() bool`

HasInternalSystemPrompt returns a boolean if a field has been set.

### GetLifecycle

`func (o *V1alpha1KomputerAgentSpec) GetLifecycle() V1alpha1AgentLifecycle`

GetLifecycle returns the Lifecycle field if non-nil, zero value otherwise.

### GetLifecycleOk

`func (o *V1alpha1KomputerAgentSpec) GetLifecycleOk() (*V1alpha1AgentLifecycle, bool)`

GetLifecycleOk returns a tuple with the Lifecycle field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLifecycle

`func (o *V1alpha1KomputerAgentSpec) SetLifecycle(v V1alpha1AgentLifecycle)`

SetLifecycle sets Lifecycle field to given value.

### HasLifecycle

`func (o *V1alpha1KomputerAgentSpec) HasLifecycle() bool`

HasLifecycle returns a boolean if a field has been set.

### GetMemories

`func (o *V1alpha1KomputerAgentSpec) GetMemories() []string`

GetMemories returns the Memories field if non-nil, zero value otherwise.

### GetMemoriesOk

`func (o *V1alpha1KomputerAgentSpec) GetMemoriesOk() (*[]string, bool)`

GetMemoriesOk returns a tuple with the Memories field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemories

`func (o *V1alpha1KomputerAgentSpec) SetMemories(v []string)`

SetMemories sets Memories field to given value.

### HasMemories

`func (o *V1alpha1KomputerAgentSpec) HasMemories() bool`

HasMemories returns a boolean if a field has been set.

### GetModel

`func (o *V1alpha1KomputerAgentSpec) GetModel() string`

GetModel returns the Model field if non-nil, zero value otherwise.

### GetModelOk

`func (o *V1alpha1KomputerAgentSpec) GetModelOk() (*string, bool)`

GetModelOk returns a tuple with the Model field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetModel

`func (o *V1alpha1KomputerAgentSpec) SetModel(v string)`

SetModel sets Model field to given value.

### HasModel

`func (o *V1alpha1KomputerAgentSpec) HasModel() bool`

HasModel returns a boolean if a field has been set.

### GetOfficeManager

`func (o *V1alpha1KomputerAgentSpec) GetOfficeManager() string`

GetOfficeManager returns the OfficeManager field if non-nil, zero value otherwise.

### GetOfficeManagerOk

`func (o *V1alpha1KomputerAgentSpec) GetOfficeManagerOk() (*string, bool)`

GetOfficeManagerOk returns a tuple with the OfficeManager field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOfficeManager

`func (o *V1alpha1KomputerAgentSpec) SetOfficeManager(v string)`

SetOfficeManager sets OfficeManager field to given value.

### HasOfficeManager

`func (o *V1alpha1KomputerAgentSpec) HasOfficeManager() bool`

HasOfficeManager returns a boolean if a field has been set.

### GetPodSpec

`func (o *V1alpha1KomputerAgentSpec) GetPodSpec() V1PodSpec`

GetPodSpec returns the PodSpec field if non-nil, zero value otherwise.

### GetPodSpecOk

`func (o *V1alpha1KomputerAgentSpec) GetPodSpecOk() (*V1PodSpec, bool)`

GetPodSpecOk returns a tuple with the PodSpec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodSpec

`func (o *V1alpha1KomputerAgentSpec) SetPodSpec(v V1PodSpec)`

SetPodSpec sets PodSpec field to given value.

### HasPodSpec

`func (o *V1alpha1KomputerAgentSpec) HasPodSpec() bool`

HasPodSpec returns a boolean if a field has been set.

### GetPriority

`func (o *V1alpha1KomputerAgentSpec) GetPriority() int32`

GetPriority returns the Priority field if non-nil, zero value otherwise.

### GetPriorityOk

`func (o *V1alpha1KomputerAgentSpec) GetPriorityOk() (*int32, bool)`

GetPriorityOk returns a tuple with the Priority field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriority

`func (o *V1alpha1KomputerAgentSpec) SetPriority(v int32)`

SetPriority sets Priority field to given value.

### HasPriority

`func (o *V1alpha1KomputerAgentSpec) HasPriority() bool`

HasPriority returns a boolean if a field has been set.

### GetRole

`func (o *V1alpha1KomputerAgentSpec) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *V1alpha1KomputerAgentSpec) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *V1alpha1KomputerAgentSpec) SetRole(v string)`

SetRole sets Role field to given value.

### HasRole

`func (o *V1alpha1KomputerAgentSpec) HasRole() bool`

HasRole returns a boolean if a field has been set.

### GetSecrets

`func (o *V1alpha1KomputerAgentSpec) GetSecrets() []string`

GetSecrets returns the Secrets field if non-nil, zero value otherwise.

### GetSecretsOk

`func (o *V1alpha1KomputerAgentSpec) GetSecretsOk() (*[]string, bool)`

GetSecretsOk returns a tuple with the Secrets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecrets

`func (o *V1alpha1KomputerAgentSpec) SetSecrets(v []string)`

SetSecrets sets Secrets field to given value.

### HasSecrets

`func (o *V1alpha1KomputerAgentSpec) HasSecrets() bool`

HasSecrets returns a boolean if a field has been set.

### GetSkills

`func (o *V1alpha1KomputerAgentSpec) GetSkills() []string`

GetSkills returns the Skills field if non-nil, zero value otherwise.

### GetSkillsOk

`func (o *V1alpha1KomputerAgentSpec) GetSkillsOk() (*[]string, bool)`

GetSkillsOk returns a tuple with the Skills field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSkills

`func (o *V1alpha1KomputerAgentSpec) SetSkills(v []string)`

SetSkills sets Skills field to given value.

### HasSkills

`func (o *V1alpha1KomputerAgentSpec) HasSkills() bool`

HasSkills returns a boolean if a field has been set.

### GetStorage

`func (o *V1alpha1KomputerAgentSpec) GetStorage() V1alpha1StorageSpec`

GetStorage returns the Storage field if non-nil, zero value otherwise.

### GetStorageOk

`func (o *V1alpha1KomputerAgentSpec) GetStorageOk() (*V1alpha1StorageSpec, bool)`

GetStorageOk returns a tuple with the Storage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStorage

`func (o *V1alpha1KomputerAgentSpec) SetStorage(v V1alpha1StorageSpec)`

SetStorage sets Storage field to given value.

### HasStorage

`func (o *V1alpha1KomputerAgentSpec) HasStorage() bool`

HasStorage returns a boolean if a field has been set.

### GetSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) GetSystemPrompt() string`

GetSystemPrompt returns the SystemPrompt field if non-nil, zero value otherwise.

### GetSystemPromptOk

`func (o *V1alpha1KomputerAgentSpec) GetSystemPromptOk() (*string, bool)`

GetSystemPromptOk returns a tuple with the SystemPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) SetSystemPrompt(v string)`

SetSystemPrompt sets SystemPrompt field to given value.

### HasSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) HasSystemPrompt() bool`

HasSystemPrompt returns a boolean if a field has been set.

### GetTemplateRef

`func (o *V1alpha1KomputerAgentSpec) GetTemplateRef() string`

GetTemplateRef returns the TemplateRef field if non-nil, zero value otherwise.

### GetTemplateRefOk

`func (o *V1alpha1KomputerAgentSpec) GetTemplateRefOk() (*string, bool)`

GetTemplateRefOk returns a tuple with the TemplateRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTemplateRef

`func (o *V1alpha1KomputerAgentSpec) SetTemplateRef(v string)`

SetTemplateRef sets TemplateRef field to given value.

### HasTemplateRef

`func (o *V1alpha1KomputerAgentSpec) HasTemplateRef() bool`

HasTemplateRef returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


