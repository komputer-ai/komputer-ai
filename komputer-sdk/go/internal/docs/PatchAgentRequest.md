# PatchAgentRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Connectors** | Pointer to **[]string** | connector names to attach | [optional] 
**Instructions** | Pointer to **string** |  | [optional] 
**Lifecycle** | Pointer to **string** |  | [optional] 
**Memories** | Pointer to **[]string** | memory names to attach | [optional] 
**Model** | Pointer to **string** |  | [optional] 
**PodSpec** | Pointer to [**V1PodSpec**](V1PodSpec.md) |  | [optional] 
**SecretRefs** | Pointer to **[]string** | full replacement list of K8s secret names | [optional] 
**Skills** | Pointer to **[]string** | skill names to attach | [optional] 
**Storage** | Pointer to [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) |  | [optional] 
**SystemPrompt** | Pointer to **string** | custom system prompt | [optional] 
**TemplateRef** | Pointer to **string** |  | [optional] 

## Methods

### NewPatchAgentRequest

`func NewPatchAgentRequest() *PatchAgentRequest`

NewPatchAgentRequest instantiates a new PatchAgentRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPatchAgentRequestWithDefaults

`func NewPatchAgentRequestWithDefaults() *PatchAgentRequest`

NewPatchAgentRequestWithDefaults instantiates a new PatchAgentRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConnectors

`func (o *PatchAgentRequest) GetConnectors() []string`

GetConnectors returns the Connectors field if non-nil, zero value otherwise.

### GetConnectorsOk

`func (o *PatchAgentRequest) GetConnectorsOk() (*[]string, bool)`

GetConnectorsOk returns a tuple with the Connectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConnectors

`func (o *PatchAgentRequest) SetConnectors(v []string)`

SetConnectors sets Connectors field to given value.

### HasConnectors

`func (o *PatchAgentRequest) HasConnectors() bool`

HasConnectors returns a boolean if a field has been set.

### GetInstructions

`func (o *PatchAgentRequest) GetInstructions() string`

GetInstructions returns the Instructions field if non-nil, zero value otherwise.

### GetInstructionsOk

`func (o *PatchAgentRequest) GetInstructionsOk() (*string, bool)`

GetInstructionsOk returns a tuple with the Instructions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstructions

`func (o *PatchAgentRequest) SetInstructions(v string)`

SetInstructions sets Instructions field to given value.

### HasInstructions

`func (o *PatchAgentRequest) HasInstructions() bool`

HasInstructions returns a boolean if a field has been set.

### GetLifecycle

`func (o *PatchAgentRequest) GetLifecycle() string`

GetLifecycle returns the Lifecycle field if non-nil, zero value otherwise.

### GetLifecycleOk

`func (o *PatchAgentRequest) GetLifecycleOk() (*string, bool)`

GetLifecycleOk returns a tuple with the Lifecycle field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLifecycle

`func (o *PatchAgentRequest) SetLifecycle(v string)`

SetLifecycle sets Lifecycle field to given value.

### HasLifecycle

`func (o *PatchAgentRequest) HasLifecycle() bool`

HasLifecycle returns a boolean if a field has been set.

### GetMemories

`func (o *PatchAgentRequest) GetMemories() []string`

GetMemories returns the Memories field if non-nil, zero value otherwise.

### GetMemoriesOk

`func (o *PatchAgentRequest) GetMemoriesOk() (*[]string, bool)`

GetMemoriesOk returns a tuple with the Memories field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemories

`func (o *PatchAgentRequest) SetMemories(v []string)`

SetMemories sets Memories field to given value.

### HasMemories

`func (o *PatchAgentRequest) HasMemories() bool`

HasMemories returns a boolean if a field has been set.

### GetModel

`func (o *PatchAgentRequest) GetModel() string`

GetModel returns the Model field if non-nil, zero value otherwise.

### GetModelOk

`func (o *PatchAgentRequest) GetModelOk() (*string, bool)`

GetModelOk returns a tuple with the Model field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetModel

`func (o *PatchAgentRequest) SetModel(v string)`

SetModel sets Model field to given value.

### HasModel

`func (o *PatchAgentRequest) HasModel() bool`

HasModel returns a boolean if a field has been set.

### GetPodSpec

`func (o *PatchAgentRequest) GetPodSpec() V1PodSpec`

GetPodSpec returns the PodSpec field if non-nil, zero value otherwise.

### GetPodSpecOk

`func (o *PatchAgentRequest) GetPodSpecOk() (*V1PodSpec, bool)`

GetPodSpecOk returns a tuple with the PodSpec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodSpec

`func (o *PatchAgentRequest) SetPodSpec(v V1PodSpec)`

SetPodSpec sets PodSpec field to given value.

### HasPodSpec

`func (o *PatchAgentRequest) HasPodSpec() bool`

HasPodSpec returns a boolean if a field has been set.

### GetSecretRefs

`func (o *PatchAgentRequest) GetSecretRefs() []string`

GetSecretRefs returns the SecretRefs field if non-nil, zero value otherwise.

### GetSecretRefsOk

`func (o *PatchAgentRequest) GetSecretRefsOk() (*[]string, bool)`

GetSecretRefsOk returns a tuple with the SecretRefs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecretRefs

`func (o *PatchAgentRequest) SetSecretRefs(v []string)`

SetSecretRefs sets SecretRefs field to given value.

### HasSecretRefs

`func (o *PatchAgentRequest) HasSecretRefs() bool`

HasSecretRefs returns a boolean if a field has been set.

### GetSkills

`func (o *PatchAgentRequest) GetSkills() []string`

GetSkills returns the Skills field if non-nil, zero value otherwise.

### GetSkillsOk

`func (o *PatchAgentRequest) GetSkillsOk() (*[]string, bool)`

GetSkillsOk returns a tuple with the Skills field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSkills

`func (o *PatchAgentRequest) SetSkills(v []string)`

SetSkills sets Skills field to given value.

### HasSkills

`func (o *PatchAgentRequest) HasSkills() bool`

HasSkills returns a boolean if a field has been set.

### GetStorage

`func (o *PatchAgentRequest) GetStorage() V1alpha1StorageSpec`

GetStorage returns the Storage field if non-nil, zero value otherwise.

### GetStorageOk

`func (o *PatchAgentRequest) GetStorageOk() (*V1alpha1StorageSpec, bool)`

GetStorageOk returns a tuple with the Storage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStorage

`func (o *PatchAgentRequest) SetStorage(v V1alpha1StorageSpec)`

SetStorage sets Storage field to given value.

### HasStorage

`func (o *PatchAgentRequest) HasStorage() bool`

HasStorage returns a boolean if a field has been set.

### GetSystemPrompt

`func (o *PatchAgentRequest) GetSystemPrompt() string`

GetSystemPrompt returns the SystemPrompt field if non-nil, zero value otherwise.

### GetSystemPromptOk

`func (o *PatchAgentRequest) GetSystemPromptOk() (*string, bool)`

GetSystemPromptOk returns a tuple with the SystemPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSystemPrompt

`func (o *PatchAgentRequest) SetSystemPrompt(v string)`

SetSystemPrompt sets SystemPrompt field to given value.

### HasSystemPrompt

`func (o *PatchAgentRequest) HasSystemPrompt() bool`

HasSystemPrompt returns a boolean if a field has been set.

### GetTemplateRef

`func (o *PatchAgentRequest) GetTemplateRef() string`

GetTemplateRef returns the TemplateRef field if non-nil, zero value otherwise.

### GetTemplateRefOk

`func (o *PatchAgentRequest) GetTemplateRefOk() (*string, bool)`

GetTemplateRefOk returns a tuple with the TemplateRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTemplateRef

`func (o *PatchAgentRequest) SetTemplateRef(v string)`

SetTemplateRef sets TemplateRef field to given value.

### HasTemplateRef

`func (o *PatchAgentRequest) HasTemplateRef() bool`

HasTemplateRef returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


