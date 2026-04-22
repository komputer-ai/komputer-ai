# AgentResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Connectors** | Pointer to **[]string** | KomputerConnector names attached to this agent | [optional] 
**CreatedAt** | Pointer to **string** |  | [optional] 
**Instructions** | Pointer to **string** | User task (spec.instructions) | [optional] 
**LastTaskCostUSD** | Pointer to **string** |  | [optional] 
**LastTaskMessage** | Pointer to **string** |  | [optional] 
**Lifecycle** | Pointer to **string** |  | [optional] 
**Memories** | Pointer to **[]string** | KomputerMemory names attached to this agent | [optional] 
**Model** | Pointer to **string** |  | [optional] 
**ModelContextWindow** | Pointer to **int32** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Namespace** | Pointer to **string** |  | [optional] 
**PodSpec** | Pointer to [**V1PodSpec**](V1PodSpec.md) |  | [optional] 
**Secrets** | Pointer to **[]string** | Key names from K8s Secrets (not values) | [optional] 
**Skills** | Pointer to **[]string** | KomputerSkill names attached to this agent | [optional] 
**Status** | Pointer to **string** |  | [optional] 
**Storage** | Pointer to [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) |  | [optional] 
**SystemPrompt** | Pointer to **string** | Custom system prompt (spec.systemPrompt) | [optional] 
**TaskStatus** | Pointer to **string** |  | [optional] 
**TotalCostUSD** | Pointer to **string** |  | [optional] 
**TotalTokens** | Pointer to **int32** |  | [optional] 

## Methods

### NewAgentResponse

`func NewAgentResponse() *AgentResponse`

NewAgentResponse instantiates a new AgentResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentResponseWithDefaults

`func NewAgentResponseWithDefaults() *AgentResponse`

NewAgentResponseWithDefaults instantiates a new AgentResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetConnectors

`func (o *AgentResponse) GetConnectors() []string`

GetConnectors returns the Connectors field if non-nil, zero value otherwise.

### GetConnectorsOk

`func (o *AgentResponse) GetConnectorsOk() (*[]string, bool)`

GetConnectorsOk returns a tuple with the Connectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConnectors

`func (o *AgentResponse) SetConnectors(v []string)`

SetConnectors sets Connectors field to given value.

### HasConnectors

`func (o *AgentResponse) HasConnectors() bool`

HasConnectors returns a boolean if a field has been set.

### GetCreatedAt

`func (o *AgentResponse) GetCreatedAt() string`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *AgentResponse) GetCreatedAtOk() (*string, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *AgentResponse) SetCreatedAt(v string)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *AgentResponse) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetInstructions

`func (o *AgentResponse) GetInstructions() string`

GetInstructions returns the Instructions field if non-nil, zero value otherwise.

### GetInstructionsOk

`func (o *AgentResponse) GetInstructionsOk() (*string, bool)`

GetInstructionsOk returns a tuple with the Instructions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstructions

`func (o *AgentResponse) SetInstructions(v string)`

SetInstructions sets Instructions field to given value.

### HasInstructions

`func (o *AgentResponse) HasInstructions() bool`

HasInstructions returns a boolean if a field has been set.

### GetLastTaskCostUSD

`func (o *AgentResponse) GetLastTaskCostUSD() string`

GetLastTaskCostUSD returns the LastTaskCostUSD field if non-nil, zero value otherwise.

### GetLastTaskCostUSDOk

`func (o *AgentResponse) GetLastTaskCostUSDOk() (*string, bool)`

GetLastTaskCostUSDOk returns a tuple with the LastTaskCostUSD field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastTaskCostUSD

`func (o *AgentResponse) SetLastTaskCostUSD(v string)`

SetLastTaskCostUSD sets LastTaskCostUSD field to given value.

### HasLastTaskCostUSD

`func (o *AgentResponse) HasLastTaskCostUSD() bool`

HasLastTaskCostUSD returns a boolean if a field has been set.

### GetLastTaskMessage

`func (o *AgentResponse) GetLastTaskMessage() string`

GetLastTaskMessage returns the LastTaskMessage field if non-nil, zero value otherwise.

### GetLastTaskMessageOk

`func (o *AgentResponse) GetLastTaskMessageOk() (*string, bool)`

GetLastTaskMessageOk returns a tuple with the LastTaskMessage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastTaskMessage

`func (o *AgentResponse) SetLastTaskMessage(v string)`

SetLastTaskMessage sets LastTaskMessage field to given value.

### HasLastTaskMessage

`func (o *AgentResponse) HasLastTaskMessage() bool`

HasLastTaskMessage returns a boolean if a field has been set.

### GetLifecycle

`func (o *AgentResponse) GetLifecycle() string`

GetLifecycle returns the Lifecycle field if non-nil, zero value otherwise.

### GetLifecycleOk

`func (o *AgentResponse) GetLifecycleOk() (*string, bool)`

GetLifecycleOk returns a tuple with the Lifecycle field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLifecycle

`func (o *AgentResponse) SetLifecycle(v string)`

SetLifecycle sets Lifecycle field to given value.

### HasLifecycle

`func (o *AgentResponse) HasLifecycle() bool`

HasLifecycle returns a boolean if a field has been set.

### GetMemories

`func (o *AgentResponse) GetMemories() []string`

GetMemories returns the Memories field if non-nil, zero value otherwise.

### GetMemoriesOk

`func (o *AgentResponse) GetMemoriesOk() (*[]string, bool)`

GetMemoriesOk returns a tuple with the Memories field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemories

`func (o *AgentResponse) SetMemories(v []string)`

SetMemories sets Memories field to given value.

### HasMemories

`func (o *AgentResponse) HasMemories() bool`

HasMemories returns a boolean if a field has been set.

### GetModel

`func (o *AgentResponse) GetModel() string`

GetModel returns the Model field if non-nil, zero value otherwise.

### GetModelOk

`func (o *AgentResponse) GetModelOk() (*string, bool)`

GetModelOk returns a tuple with the Model field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetModel

`func (o *AgentResponse) SetModel(v string)`

SetModel sets Model field to given value.

### HasModel

`func (o *AgentResponse) HasModel() bool`

HasModel returns a boolean if a field has been set.

### GetModelContextWindow

`func (o *AgentResponse) GetModelContextWindow() int32`

GetModelContextWindow returns the ModelContextWindow field if non-nil, zero value otherwise.

### GetModelContextWindowOk

`func (o *AgentResponse) GetModelContextWindowOk() (*int32, bool)`

GetModelContextWindowOk returns a tuple with the ModelContextWindow field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetModelContextWindow

`func (o *AgentResponse) SetModelContextWindow(v int32)`

SetModelContextWindow sets ModelContextWindow field to given value.

### HasModelContextWindow

`func (o *AgentResponse) HasModelContextWindow() bool`

HasModelContextWindow returns a boolean if a field has been set.

### GetName

`func (o *AgentResponse) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *AgentResponse) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *AgentResponse) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *AgentResponse) HasName() bool`

HasName returns a boolean if a field has been set.

### GetNamespace

`func (o *AgentResponse) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *AgentResponse) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *AgentResponse) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *AgentResponse) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetPodSpec

`func (o *AgentResponse) GetPodSpec() V1PodSpec`

GetPodSpec returns the PodSpec field if non-nil, zero value otherwise.

### GetPodSpecOk

`func (o *AgentResponse) GetPodSpecOk() (*V1PodSpec, bool)`

GetPodSpecOk returns a tuple with the PodSpec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodSpec

`func (o *AgentResponse) SetPodSpec(v V1PodSpec)`

SetPodSpec sets PodSpec field to given value.

### HasPodSpec

`func (o *AgentResponse) HasPodSpec() bool`

HasPodSpec returns a boolean if a field has been set.

### GetSecrets

`func (o *AgentResponse) GetSecrets() []string`

GetSecrets returns the Secrets field if non-nil, zero value otherwise.

### GetSecretsOk

`func (o *AgentResponse) GetSecretsOk() (*[]string, bool)`

GetSecretsOk returns a tuple with the Secrets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecrets

`func (o *AgentResponse) SetSecrets(v []string)`

SetSecrets sets Secrets field to given value.

### HasSecrets

`func (o *AgentResponse) HasSecrets() bool`

HasSecrets returns a boolean if a field has been set.

### GetSkills

`func (o *AgentResponse) GetSkills() []string`

GetSkills returns the Skills field if non-nil, zero value otherwise.

### GetSkillsOk

`func (o *AgentResponse) GetSkillsOk() (*[]string, bool)`

GetSkillsOk returns a tuple with the Skills field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSkills

`func (o *AgentResponse) SetSkills(v []string)`

SetSkills sets Skills field to given value.

### HasSkills

`func (o *AgentResponse) HasSkills() bool`

HasSkills returns a boolean if a field has been set.

### GetStatus

`func (o *AgentResponse) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *AgentResponse) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *AgentResponse) SetStatus(v string)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *AgentResponse) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### GetStorage

`func (o *AgentResponse) GetStorage() V1alpha1StorageSpec`

GetStorage returns the Storage field if non-nil, zero value otherwise.

### GetStorageOk

`func (o *AgentResponse) GetStorageOk() (*V1alpha1StorageSpec, bool)`

GetStorageOk returns a tuple with the Storage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStorage

`func (o *AgentResponse) SetStorage(v V1alpha1StorageSpec)`

SetStorage sets Storage field to given value.

### HasStorage

`func (o *AgentResponse) HasStorage() bool`

HasStorage returns a boolean if a field has been set.

### GetSystemPrompt

`func (o *AgentResponse) GetSystemPrompt() string`

GetSystemPrompt returns the SystemPrompt field if non-nil, zero value otherwise.

### GetSystemPromptOk

`func (o *AgentResponse) GetSystemPromptOk() (*string, bool)`

GetSystemPromptOk returns a tuple with the SystemPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSystemPrompt

`func (o *AgentResponse) SetSystemPrompt(v string)`

SetSystemPrompt sets SystemPrompt field to given value.

### HasSystemPrompt

`func (o *AgentResponse) HasSystemPrompt() bool`

HasSystemPrompt returns a boolean if a field has been set.

### GetTaskStatus

`func (o *AgentResponse) GetTaskStatus() string`

GetTaskStatus returns the TaskStatus field if non-nil, zero value otherwise.

### GetTaskStatusOk

`func (o *AgentResponse) GetTaskStatusOk() (*string, bool)`

GetTaskStatusOk returns a tuple with the TaskStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTaskStatus

`func (o *AgentResponse) SetTaskStatus(v string)`

SetTaskStatus sets TaskStatus field to given value.

### HasTaskStatus

`func (o *AgentResponse) HasTaskStatus() bool`

HasTaskStatus returns a boolean if a field has been set.

### GetTotalCostUSD

`func (o *AgentResponse) GetTotalCostUSD() string`

GetTotalCostUSD returns the TotalCostUSD field if non-nil, zero value otherwise.

### GetTotalCostUSDOk

`func (o *AgentResponse) GetTotalCostUSDOk() (*string, bool)`

GetTotalCostUSDOk returns a tuple with the TotalCostUSD field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalCostUSD

`func (o *AgentResponse) SetTotalCostUSD(v string)`

SetTotalCostUSD sets TotalCostUSD field to given value.

### HasTotalCostUSD

`func (o *AgentResponse) HasTotalCostUSD() bool`

HasTotalCostUSD returns a boolean if a field has been set.

### GetTotalTokens

`func (o *AgentResponse) GetTotalTokens() int32`

GetTotalTokens returns the TotalTokens field if non-nil, zero value otherwise.

### GetTotalTokensOk

`func (o *AgentResponse) GetTotalTokensOk() (*int32, bool)`

GetTotalTokensOk returns a tuple with the TotalTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalTokens

`func (o *AgentResponse) SetTotalTokens(v int32)`

SetTotalTokens sets TotalTokens field to given value.

### HasTotalTokens

`func (o *AgentResponse) HasTotalTokens() bool`

HasTotalTokens returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


