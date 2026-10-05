# CostBreakdownResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Agent** | Pointer to **string** |  | [optional] 
**CachedAt** | Pointer to **string** |  | [optional] 
**TaskCount** | Pointer to **int32** |  | [optional] 
**Tasks** | Pointer to [**[]TaskBreakdown**](TaskBreakdown.md) |  | [optional] 
**TotalCost** | Pointer to **float32** |  | [optional] 

## Methods

### NewCostBreakdownResponse

`func NewCostBreakdownResponse() *CostBreakdownResponse`

NewCostBreakdownResponse instantiates a new CostBreakdownResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCostBreakdownResponseWithDefaults

`func NewCostBreakdownResponseWithDefaults() *CostBreakdownResponse`

NewCostBreakdownResponseWithDefaults instantiates a new CostBreakdownResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAgent

`func (o *CostBreakdownResponse) GetAgent() string`

GetAgent returns the Agent field if non-nil, zero value otherwise.

### GetAgentOk

`func (o *CostBreakdownResponse) GetAgentOk() (*string, bool)`

GetAgentOk returns a tuple with the Agent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgent

`func (o *CostBreakdownResponse) SetAgent(v string)`

SetAgent sets Agent field to given value.

### HasAgent

`func (o *CostBreakdownResponse) HasAgent() bool`

HasAgent returns a boolean if a field has been set.

### GetCachedAt

`func (o *CostBreakdownResponse) GetCachedAt() string`

GetCachedAt returns the CachedAt field if non-nil, zero value otherwise.

### GetCachedAtOk

`func (o *CostBreakdownResponse) GetCachedAtOk() (*string, bool)`

GetCachedAtOk returns a tuple with the CachedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCachedAt

`func (o *CostBreakdownResponse) SetCachedAt(v string)`

SetCachedAt sets CachedAt field to given value.

### HasCachedAt

`func (o *CostBreakdownResponse) HasCachedAt() bool`

HasCachedAt returns a boolean if a field has been set.

### GetTaskCount

`func (o *CostBreakdownResponse) GetTaskCount() int32`

GetTaskCount returns the TaskCount field if non-nil, zero value otherwise.

### GetTaskCountOk

`func (o *CostBreakdownResponse) GetTaskCountOk() (*int32, bool)`

GetTaskCountOk returns a tuple with the TaskCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTaskCount

`func (o *CostBreakdownResponse) SetTaskCount(v int32)`

SetTaskCount sets TaskCount field to given value.

### HasTaskCount

`func (o *CostBreakdownResponse) HasTaskCount() bool`

HasTaskCount returns a boolean if a field has been set.

### GetTasks

`func (o *CostBreakdownResponse) GetTasks() []TaskBreakdown`

GetTasks returns the Tasks field if non-nil, zero value otherwise.

### GetTasksOk

`func (o *CostBreakdownResponse) GetTasksOk() (*[]TaskBreakdown, bool)`

GetTasksOk returns a tuple with the Tasks field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTasks

`func (o *CostBreakdownResponse) SetTasks(v []TaskBreakdown)`

SetTasks sets Tasks field to given value.

### HasTasks

`func (o *CostBreakdownResponse) HasTasks() bool`

HasTasks returns a boolean if a field has been set.

### GetTotalCost

`func (o *CostBreakdownResponse) GetTotalCost() float32`

GetTotalCost returns the TotalCost field if non-nil, zero value otherwise.

### GetTotalCostOk

`func (o *CostBreakdownResponse) GetTotalCostOk() (*float32, bool)`

GetTotalCostOk returns a tuple with the TotalCost field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalCost

`func (o *CostBreakdownResponse) SetTotalCost(v float32)`

SetTotalCost sets TotalCost field to given value.

### HasTotalCost

`func (o *CostBreakdownResponse) HasTotalCost() bool`

HasTotalCost returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


