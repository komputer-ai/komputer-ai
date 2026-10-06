# TaskBreakdown

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CacheCreateTokens** | Pointer to **int32** |  | [optional] 
**CacheReadTokens** | Pointer to **int32** |  | [optional] 
**CompletedAt** | Pointer to **string** |  | [optional] 
**CostUSD** | Pointer to **float32** |  | [optional] 
**DurationMs** | Pointer to **int32** |  | [optional] 
**Events** | Pointer to [**[]AgentEvent**](AgentEvent.md) |  | [optional] 
**Index** | Pointer to **int32** |  | [optional] 
**InputTokens** | Pointer to **int32** |  | [optional] 
**Instruction** | Pointer to **string** |  | [optional] 
**OutputTokens** | Pointer to **int32** |  | [optional] 
**StartedAt** | Pointer to **string** |  | [optional] 
**Steer** | Pointer to **bool** |  | [optional] 
**Turns** | Pointer to **int32** |  | [optional] 

## Methods

### NewTaskBreakdown

`func NewTaskBreakdown() *TaskBreakdown`

NewTaskBreakdown instantiates a new TaskBreakdown object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewTaskBreakdownWithDefaults

`func NewTaskBreakdownWithDefaults() *TaskBreakdown`

NewTaskBreakdownWithDefaults instantiates a new TaskBreakdown object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCacheCreateTokens

`func (o *TaskBreakdown) GetCacheCreateTokens() int32`

GetCacheCreateTokens returns the CacheCreateTokens field if non-nil, zero value otherwise.

### GetCacheCreateTokensOk

`func (o *TaskBreakdown) GetCacheCreateTokensOk() (*int32, bool)`

GetCacheCreateTokensOk returns a tuple with the CacheCreateTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCacheCreateTokens

`func (o *TaskBreakdown) SetCacheCreateTokens(v int32)`

SetCacheCreateTokens sets CacheCreateTokens field to given value.

### HasCacheCreateTokens

`func (o *TaskBreakdown) HasCacheCreateTokens() bool`

HasCacheCreateTokens returns a boolean if a field has been set.

### GetCacheReadTokens

`func (o *TaskBreakdown) GetCacheReadTokens() int32`

GetCacheReadTokens returns the CacheReadTokens field if non-nil, zero value otherwise.

### GetCacheReadTokensOk

`func (o *TaskBreakdown) GetCacheReadTokensOk() (*int32, bool)`

GetCacheReadTokensOk returns a tuple with the CacheReadTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCacheReadTokens

`func (o *TaskBreakdown) SetCacheReadTokens(v int32)`

SetCacheReadTokens sets CacheReadTokens field to given value.

### HasCacheReadTokens

`func (o *TaskBreakdown) HasCacheReadTokens() bool`

HasCacheReadTokens returns a boolean if a field has been set.

### GetCompletedAt

`func (o *TaskBreakdown) GetCompletedAt() string`

GetCompletedAt returns the CompletedAt field if non-nil, zero value otherwise.

### GetCompletedAtOk

`func (o *TaskBreakdown) GetCompletedAtOk() (*string, bool)`

GetCompletedAtOk returns a tuple with the CompletedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCompletedAt

`func (o *TaskBreakdown) SetCompletedAt(v string)`

SetCompletedAt sets CompletedAt field to given value.

### HasCompletedAt

`func (o *TaskBreakdown) HasCompletedAt() bool`

HasCompletedAt returns a boolean if a field has been set.

### GetCostUSD

`func (o *TaskBreakdown) GetCostUSD() float32`

GetCostUSD returns the CostUSD field if non-nil, zero value otherwise.

### GetCostUSDOk

`func (o *TaskBreakdown) GetCostUSDOk() (*float32, bool)`

GetCostUSDOk returns a tuple with the CostUSD field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCostUSD

`func (o *TaskBreakdown) SetCostUSD(v float32)`

SetCostUSD sets CostUSD field to given value.

### HasCostUSD

`func (o *TaskBreakdown) HasCostUSD() bool`

HasCostUSD returns a boolean if a field has been set.

### GetDurationMs

`func (o *TaskBreakdown) GetDurationMs() int32`

GetDurationMs returns the DurationMs field if non-nil, zero value otherwise.

### GetDurationMsOk

`func (o *TaskBreakdown) GetDurationMsOk() (*int32, bool)`

GetDurationMsOk returns a tuple with the DurationMs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDurationMs

`func (o *TaskBreakdown) SetDurationMs(v int32)`

SetDurationMs sets DurationMs field to given value.

### HasDurationMs

`func (o *TaskBreakdown) HasDurationMs() bool`

HasDurationMs returns a boolean if a field has been set.

### GetEvents

`func (o *TaskBreakdown) GetEvents() []AgentEvent`

GetEvents returns the Events field if non-nil, zero value otherwise.

### GetEventsOk

`func (o *TaskBreakdown) GetEventsOk() (*[]AgentEvent, bool)`

GetEventsOk returns a tuple with the Events field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEvents

`func (o *TaskBreakdown) SetEvents(v []AgentEvent)`

SetEvents sets Events field to given value.

### HasEvents

`func (o *TaskBreakdown) HasEvents() bool`

HasEvents returns a boolean if a field has been set.

### GetIndex

`func (o *TaskBreakdown) GetIndex() int32`

GetIndex returns the Index field if non-nil, zero value otherwise.

### GetIndexOk

`func (o *TaskBreakdown) GetIndexOk() (*int32, bool)`

GetIndexOk returns a tuple with the Index field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIndex

`func (o *TaskBreakdown) SetIndex(v int32)`

SetIndex sets Index field to given value.

### HasIndex

`func (o *TaskBreakdown) HasIndex() bool`

HasIndex returns a boolean if a field has been set.

### GetInputTokens

`func (o *TaskBreakdown) GetInputTokens() int32`

GetInputTokens returns the InputTokens field if non-nil, zero value otherwise.

### GetInputTokensOk

`func (o *TaskBreakdown) GetInputTokensOk() (*int32, bool)`

GetInputTokensOk returns a tuple with the InputTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInputTokens

`func (o *TaskBreakdown) SetInputTokens(v int32)`

SetInputTokens sets InputTokens field to given value.

### HasInputTokens

`func (o *TaskBreakdown) HasInputTokens() bool`

HasInputTokens returns a boolean if a field has been set.

### GetInstruction

`func (o *TaskBreakdown) GetInstruction() string`

GetInstruction returns the Instruction field if non-nil, zero value otherwise.

### GetInstructionOk

`func (o *TaskBreakdown) GetInstructionOk() (*string, bool)`

GetInstructionOk returns a tuple with the Instruction field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstruction

`func (o *TaskBreakdown) SetInstruction(v string)`

SetInstruction sets Instruction field to given value.

### HasInstruction

`func (o *TaskBreakdown) HasInstruction() bool`

HasInstruction returns a boolean if a field has been set.

### GetOutputTokens

`func (o *TaskBreakdown) GetOutputTokens() int32`

GetOutputTokens returns the OutputTokens field if non-nil, zero value otherwise.

### GetOutputTokensOk

`func (o *TaskBreakdown) GetOutputTokensOk() (*int32, bool)`

GetOutputTokensOk returns a tuple with the OutputTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutputTokens

`func (o *TaskBreakdown) SetOutputTokens(v int32)`

SetOutputTokens sets OutputTokens field to given value.

### HasOutputTokens

`func (o *TaskBreakdown) HasOutputTokens() bool`

HasOutputTokens returns a boolean if a field has been set.

### GetStartedAt

`func (o *TaskBreakdown) GetStartedAt() string`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *TaskBreakdown) GetStartedAtOk() (*string, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *TaskBreakdown) SetStartedAt(v string)`

SetStartedAt sets StartedAt field to given value.

### HasStartedAt

`func (o *TaskBreakdown) HasStartedAt() bool`

HasStartedAt returns a boolean if a field has been set.

### GetSteer

`func (o *TaskBreakdown) GetSteer() bool`

GetSteer returns the Steer field if non-nil, zero value otherwise.

### GetSteerOk

`func (o *TaskBreakdown) GetSteerOk() (*bool, bool)`

GetSteerOk returns a tuple with the Steer field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSteer

`func (o *TaskBreakdown) SetSteer(v bool)`

SetSteer sets Steer field to given value.

### HasSteer

`func (o *TaskBreakdown) HasSteer() bool`

HasSteer returns a boolean if a field has been set.

### GetTurns

`func (o *TaskBreakdown) GetTurns() int32`

GetTurns returns the Turns field if non-nil, zero value otherwise.

### GetTurnsOk

`func (o *TaskBreakdown) GetTurnsOk() (*int32, bool)`

GetTurnsOk returns a tuple with the Turns field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTurns

`func (o *TaskBreakdown) SetTurns(v int32)`

SetTurns sets Turns field to given value.

### HasTurns

`func (o *TaskBreakdown) HasTurns() bool`

HasTurns returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


