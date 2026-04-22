# CreateAgentRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Connectors** | Pointer to **[]string** | optional KomputerConnector names to attach | [optional] 
**Instructions** | **string** |  | 
**Lifecycle** | Pointer to **string** | \&quot;\&quot;, \&quot;Sleep\&quot;, or \&quot;AutoDelete\&quot; | [optional] 
**Memories** | Pointer to **[]string** | optional KomputerMemory names to attach | [optional] 
**Model** | Pointer to **string** |  | [optional] 
**Name** | **string** |  | 
**Namespace** | Pointer to **string** | optional, defaults to server default | [optional] 
**OfficeManager** | Pointer to **string** | set by manager MCP tool | [optional] 
**PodSpec** | Pointer to [**V1PodSpec**](V1PodSpec.md) |  | [optional] 
**Role** | Pointer to **string** | \&quot;manager\&quot; or \&quot;\&quot; (default manager) | [optional] 
**SecretRefs** | Pointer to **[]string** | names of existing K8s Secrets to attach | [optional] 
**Skills** | Pointer to **[]string** | optional KomputerSkill names to attach | [optional] 
**Storage** | Pointer to [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) |  | [optional] 
**SystemPrompt** | Pointer to **string** | optional custom system prompt | [optional] 
**TemplateRef** | Pointer to **string** |  | [optional] 

## Methods

### NewCreateAgentRequest

`func NewCreateAgentRequest(instructions string, name string, ) *CreateAgentRequest`

NewCreateAgentRequest instantiates a new CreateAgentRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateAgentRequestWithDefaults

`func NewCreateAgentRequestWithDefaults() *CreateAgentRequest`

NewCreateAgentRequestWithDefaults instantiates a new CreateAgentRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConnectors

`func (o *CreateAgentRequest) GetConnectors() []string`

GetConnectors returns the Connectors field if non-nil, zero value otherwise.

### GetConnectorsOk

`func (o *CreateAgentRequest) GetConnectorsOk() (*[]string, bool)`

GetConnectorsOk returns a tuple with the Connectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConnectors

`func (o *CreateAgentRequest) SetConnectors(v []string)`

SetConnectors sets Connectors field to given value.

### HasConnectors

`func (o *CreateAgentRequest) HasConnectors() bool`

HasConnectors returns a boolean if a field has been set.

### GetInstructions

`func (o *CreateAgentRequest) GetInstructions() string`

GetInstructions returns the Instructions field if non-nil, zero value otherwise.

### GetInstructionsOk

`func (o *CreateAgentRequest) GetInstructionsOk() (*string, bool)`

GetInstructionsOk returns a tuple with the Instructions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstructions

`func (o *CreateAgentRequest) SetInstructions(v string)`

SetInstructions sets Instructions field to given value.


### GetLifecycle

`func (o *CreateAgentRequest) GetLifecycle() string`

GetLifecycle returns the Lifecycle field if non-nil, zero value otherwise.

### GetLifecycleOk

`func (o *CreateAgentRequest) GetLifecycleOk() (*string, bool)`

GetLifecycleOk returns a tuple with the Lifecycle field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLifecycle

`func (o *CreateAgentRequest) SetLifecycle(v string)`

SetLifecycle sets Lifecycle field to given value.

### HasLifecycle

`func (o *CreateAgentRequest) HasLifecycle() bool`

HasLifecycle returns a boolean if a field has been set.

### GetMemories

`func (o *CreateAgentRequest) GetMemories() []string`

GetMemories returns the Memories field if non-nil, zero value otherwise.

### GetMemoriesOk

`func (o *CreateAgentRequest) GetMemoriesOk() (*[]string, bool)`

GetMemoriesOk returns a tuple with the Memories field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemories

`func (o *CreateAgentRequest) SetMemories(v []string)`

SetMemories sets Memories field to given value.

### HasMemories

`func (o *CreateAgentRequest) HasMemories() bool`

HasMemories returns a boolean if a field has been set.

### GetModel

`func (o *CreateAgentRequest) GetModel() string`

GetModel returns the Model field if non-nil, zero value otherwise.

### GetModelOk

`func (o *CreateAgentRequest) GetModelOk() (*string, bool)`

GetModelOk returns a tuple with the Model field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetModel

`func (o *CreateAgentRequest) SetModel(v string)`

SetModel sets Model field to given value.

### HasModel

`func (o *CreateAgentRequest) HasModel() bool`

HasModel returns a boolean if a field has been set.

### GetName

`func (o *CreateAgentRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateAgentRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateAgentRequest) SetName(v string)`

SetName sets Name field to given value.


### GetNamespace

`func (o *CreateAgentRequest) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *CreateAgentRequest) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *CreateAgentRequest) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *CreateAgentRequest) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetOfficeManager

`func (o *CreateAgentRequest) GetOfficeManager() string`

GetOfficeManager returns the OfficeManager field if non-nil, zero value otherwise.

### GetOfficeManagerOk

`func (o *CreateAgentRequest) GetOfficeManagerOk() (*string, bool)`

GetOfficeManagerOk returns a tuple with the OfficeManager field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOfficeManager

`func (o *CreateAgentRequest) SetOfficeManager(v string)`

SetOfficeManager sets OfficeManager field to given value.

### HasOfficeManager

`func (o *CreateAgentRequest) HasOfficeManager() bool`

HasOfficeManager returns a boolean if a field has been set.

### GetPodSpec

`func (o *CreateAgentRequest) GetPodSpec() V1PodSpec`

GetPodSpec returns the PodSpec field if non-nil, zero value otherwise.

### GetPodSpecOk

`func (o *CreateAgentRequest) GetPodSpecOk() (*V1PodSpec, bool)`

GetPodSpecOk returns a tuple with the PodSpec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodSpec

`func (o *CreateAgentRequest) SetPodSpec(v V1PodSpec)`

SetPodSpec sets PodSpec field to given value.

### HasPodSpec

`func (o *CreateAgentRequest) HasPodSpec() bool`

HasPodSpec returns a boolean if a field has been set.

### GetRole

`func (o *CreateAgentRequest) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *CreateAgentRequest) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *CreateAgentRequest) SetRole(v string)`

SetRole sets Role field to given value.

### HasRole

`func (o *CreateAgentRequest) HasRole() bool`

HasRole returns a boolean if a field has been set.

### GetSecretRefs

`func (o *CreateAgentRequest) GetSecretRefs() []string`

GetSecretRefs returns the SecretRefs field if non-nil, zero value otherwise.

### GetSecretRefsOk

`func (o *CreateAgentRequest) GetSecretRefsOk() (*[]string, bool)`

GetSecretRefsOk returns a tuple with the SecretRefs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRefs

`func (o *CreateAgentRequest) SetSecretRefs(v []string)`

SetSecretRefs sets SecretRefs field to given value.

### HasSecretRefs

`func (o *CreateAgentRequest) HasSecretRefs() bool`

HasSecretRefs returns a boolean if a field has been set.

### GetSkills

`func (o *CreateAgentRequest) GetSkills() []string`

GetSkills returns the Skills field if non-nil, zero value otherwise.

### GetSkillsOk

`func (o *CreateAgentRequest) GetSkillsOk() (*[]string, bool)`

GetSkillsOk returns a tuple with the Skills field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSkills

`func (o *CreateAgentRequest) SetSkills(v []string)`

SetSkills sets Skills field to given value.

### HasSkills

`func (o *CreateAgentRequest) HasSkills() bool`

HasSkills returns a boolean if a field has been set.

### GetStorage

`func (o *CreateAgentRequest) GetStorage() V1alpha1StorageSpec`

GetStorage returns the Storage field if non-nil, zero value otherwise.

### GetStorageOk

`func (o *CreateAgentRequest) GetStorageOk() (*V1alpha1StorageSpec, bool)`

GetStorageOk returns a tuple with the Storage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStorage

`func (o *CreateAgentRequest) SetStorage(v V1alpha1StorageSpec)`

SetStorage sets Storage field to given value.

### HasStorage

`func (o *CreateAgentRequest) HasStorage() bool`

HasStorage returns a boolean if a field has been set.

### GetSystemPrompt

`func (o *CreateAgentRequest) GetSystemPrompt() string`

GetSystemPrompt returns the SystemPrompt field if non-nil, zero value otherwise.

### GetSystemPromptOk

`func (o *CreateAgentRequest) GetSystemPromptOk() (*string, bool)`

GetSystemPromptOk returns a tuple with the SystemPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSystemPrompt

`func (o *CreateAgentRequest) SetSystemPrompt(v string)`

SetSystemPrompt sets SystemPrompt field to given value.

### HasSystemPrompt

`func (o *CreateAgentRequest) HasSystemPrompt() bool`

HasSystemPrompt returns a boolean if a field has been set.

### GetTemplateRef

`func (o *CreateAgentRequest) GetTemplateRef() string`

GetTemplateRef returns the TemplateRef field if non-nil, zero value otherwise.

### GetTemplateRefOk

`func (o *CreateAgentRequest) GetTemplateRefOk() (*string, bool)`

GetTemplateRefOk returns a tuple with the TemplateRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTemplateRef

`func (o *CreateAgentRequest) SetTemplateRef(v string)`

SetTemplateRef sets TemplateRef field to given value.

### HasTemplateRef

`func (o *CreateAgentRequest) HasTemplateRef() bool`

HasTemplateRef returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


