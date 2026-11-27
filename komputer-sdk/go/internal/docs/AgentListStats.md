# AgentListStats

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ByStatus** | Pointer to **map[string]int32** | ByStatus counts agents per phase. Agents with no phase yet aren&#39;t counted. | [optional] 
**TotalCostUSD** | Pointer to **string** | TotalCostUSD is the summed totalCostUSD, formatted like the agent field. | [optional] 

## Methods

### NewAgentListStats

`func NewAgentListStats() *AgentListStats`

NewAgentListStats instantiates a new AgentListStats object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentListStatsWithDefaults

`func NewAgentListStatsWithDefaults() *AgentListStats`

NewAgentListStatsWithDefaults instantiates a new AgentListStats object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetByStatus

`func (o *AgentListStats) GetByStatus() map[string]int32`

GetByStatus returns the ByStatus field if non-nil, zero value otherwise.

### GetByStatusOk

`func (o *AgentListStats) GetByStatusOk() (*map[string]int32, bool)`

GetByStatusOk returns a tuple with the ByStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetByStatus

`func (o *AgentListStats) SetByStatus(v map[string]int32)`

SetByStatus sets ByStatus field to given value.

### HasByStatus

`func (o *AgentListStats) HasByStatus() bool`

HasByStatus returns a boolean if a field has been set.

### GetTotalCostUSD

`func (o *AgentListStats) GetTotalCostUSD() string`

GetTotalCostUSD returns the TotalCostUSD field if non-nil, zero value otherwise.

### GetTotalCostUSDOk

`func (o *AgentListStats) GetTotalCostUSDOk() (*string, bool)`

GetTotalCostUSDOk returns a tuple with the TotalCostUSD field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalCostUSD

`func (o *AgentListStats) SetTotalCostUSD(v string)`

SetTotalCostUSD sets TotalCostUSD field to given value.

### HasTotalCostUSD

`func (o *AgentListStats) HasTotalCostUSD() bool`

HasTotalCostUSD returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


