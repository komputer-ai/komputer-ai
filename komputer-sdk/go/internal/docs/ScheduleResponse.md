# ScheduleResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AgentName** | Pointer to **string** |  | [optional] 
**AutoDelete** | Pointer to **bool** |  | [optional] 
**CreatedAt** | Pointer to **string** |  | [optional] 
**FailedRuns** | Pointer to **int32** |  | [optional] 
**Instructions** | Pointer to **string** |  | [optional] 
**KeepAgents** | Pointer to **bool** |  | [optional] 
**LastRunCostUSD** | Pointer to **string** |  | [optional] 
**LastRunStatus** | Pointer to **string** |  | [optional] 
**LastRunTime** | Pointer to **string** |  | [optional] 
**LastRunTokens** | Pointer to **int32** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Namespace** | Pointer to **string** |  | [optional] 
**NextRunTime** | Pointer to **string** |  | [optional] 
**Phase** | Pointer to **string** |  | [optional] 
**RunCount** | Pointer to **int32** |  | [optional] 
**Schedule** | Pointer to **string** |  | [optional] 
**SuccessfulRuns** | Pointer to **int32** |  | [optional] 
**Timezone** | Pointer to **string** |  | [optional] 
**TotalCostUSD** | Pointer to **string** |  | [optional] 
**TotalTokens** | Pointer to **int32** |  | [optional] 

## Methods

### NewScheduleResponse

`func NewScheduleResponse() *ScheduleResponse`

NewScheduleResponse instantiates a new ScheduleResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewScheduleResponseWithDefaults

`func NewScheduleResponseWithDefaults() *ScheduleResponse`

NewScheduleResponseWithDefaults instantiates a new ScheduleResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAgentName

`func (o *ScheduleResponse) GetAgentName() string`

GetAgentName returns the AgentName field if non-nil, zero value otherwise.

### GetAgentNameOk

`func (o *ScheduleResponse) GetAgentNameOk() (*string, bool)`

GetAgentNameOk returns a tuple with the AgentName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentName

`func (o *ScheduleResponse) SetAgentName(v string)`

SetAgentName sets AgentName field to given value.

### HasAgentName

`func (o *ScheduleResponse) HasAgentName() bool`

HasAgentName returns a boolean if a field has been set.

### GetAutoDelete

`func (o *ScheduleResponse) GetAutoDelete() bool`

GetAutoDelete returns the AutoDelete field if non-nil, zero value otherwise.

### GetAutoDeleteOk

`func (o *ScheduleResponse) GetAutoDeleteOk() (*bool, bool)`

GetAutoDeleteOk returns a tuple with the AutoDelete field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoDelete

`func (o *ScheduleResponse) SetAutoDelete(v bool)`

SetAutoDelete sets AutoDelete field to given value.

### HasAutoDelete

`func (o *ScheduleResponse) HasAutoDelete() bool`

HasAutoDelete returns a boolean if a field has been set.

### GetCreatedAt

`func (o *ScheduleResponse) GetCreatedAt() string`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *ScheduleResponse) GetCreatedAtOk() (*string, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *ScheduleResponse) SetCreatedAt(v string)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *ScheduleResponse) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetFailedRuns

`func (o *ScheduleResponse) GetFailedRuns() int32`

GetFailedRuns returns the FailedRuns field if non-nil, zero value otherwise.

### GetFailedRunsOk

`func (o *ScheduleResponse) GetFailedRunsOk() (*int32, bool)`

GetFailedRunsOk returns a tuple with the FailedRuns field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFailedRuns

`func (o *ScheduleResponse) SetFailedRuns(v int32)`

SetFailedRuns sets FailedRuns field to given value.

### HasFailedRuns

`func (o *ScheduleResponse) HasFailedRuns() bool`

HasFailedRuns returns a boolean if a field has been set.

### GetInstructions

`func (o *ScheduleResponse) GetInstructions() string`

GetInstructions returns the Instructions field if non-nil, zero value otherwise.

### GetInstructionsOk

`func (o *ScheduleResponse) GetInstructionsOk() (*string, bool)`

GetInstructionsOk returns a tuple with the Instructions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstructions

`func (o *ScheduleResponse) SetInstructions(v string)`

SetInstructions sets Instructions field to given value.

### HasInstructions

`func (o *ScheduleResponse) HasInstructions() bool`

HasInstructions returns a boolean if a field has been set.

### GetKeepAgents

`func (o *ScheduleResponse) GetKeepAgents() bool`

GetKeepAgents returns the KeepAgents field if non-nil, zero value otherwise.

### GetKeepAgentsOk

`func (o *ScheduleResponse) GetKeepAgentsOk() (*bool, bool)`

GetKeepAgentsOk returns a tuple with the KeepAgents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKeepAgents

`func (o *ScheduleResponse) SetKeepAgents(v bool)`

SetKeepAgents sets KeepAgents field to given value.

### HasKeepAgents

`func (o *ScheduleResponse) HasKeepAgents() bool`

HasKeepAgents returns a boolean if a field has been set.

### GetLastRunCostUSD

`func (o *ScheduleResponse) GetLastRunCostUSD() string`

GetLastRunCostUSD returns the LastRunCostUSD field if non-nil, zero value otherwise.

### GetLastRunCostUSDOk

`func (o *ScheduleResponse) GetLastRunCostUSDOk() (*string, bool)`

GetLastRunCostUSDOk returns a tuple with the LastRunCostUSD field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastRunCostUSD

`func (o *ScheduleResponse) SetLastRunCostUSD(v string)`

SetLastRunCostUSD sets LastRunCostUSD field to given value.

### HasLastRunCostUSD

`func (o *ScheduleResponse) HasLastRunCostUSD() bool`

HasLastRunCostUSD returns a boolean if a field has been set.

### GetLastRunStatus

`func (o *ScheduleResponse) GetLastRunStatus() string`

GetLastRunStatus returns the LastRunStatus field if non-nil, zero value otherwise.

### GetLastRunStatusOk

`func (o *ScheduleResponse) GetLastRunStatusOk() (*string, bool)`

GetLastRunStatusOk returns a tuple with the LastRunStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastRunStatus

`func (o *ScheduleResponse) SetLastRunStatus(v string)`

SetLastRunStatus sets LastRunStatus field to given value.

### HasLastRunStatus

`func (o *ScheduleResponse) HasLastRunStatus() bool`

HasLastRunStatus returns a boolean if a field has been set.

### GetLastRunTime

`func (o *ScheduleResponse) GetLastRunTime() string`

GetLastRunTime returns the LastRunTime field if non-nil, zero value otherwise.

### GetLastRunTimeOk

`func (o *ScheduleResponse) GetLastRunTimeOk() (*string, bool)`

GetLastRunTimeOk returns a tuple with the LastRunTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastRunTime

`func (o *ScheduleResponse) SetLastRunTime(v string)`

SetLastRunTime sets LastRunTime field to given value.

### HasLastRunTime

`func (o *ScheduleResponse) HasLastRunTime() bool`

HasLastRunTime returns a boolean if a field has been set.

### GetLastRunTokens

`func (o *ScheduleResponse) GetLastRunTokens() int32`

GetLastRunTokens returns the LastRunTokens field if non-nil, zero value otherwise.

### GetLastRunTokensOk

`func (o *ScheduleResponse) GetLastRunTokensOk() (*int32, bool)`

GetLastRunTokensOk returns a tuple with the LastRunTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastRunTokens

`func (o *ScheduleResponse) SetLastRunTokens(v int32)`

SetLastRunTokens sets LastRunTokens field to given value.

### HasLastRunTokens

`func (o *ScheduleResponse) HasLastRunTokens() bool`

HasLastRunTokens returns a boolean if a field has been set.

### GetName

`func (o *ScheduleResponse) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ScheduleResponse) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ScheduleResponse) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *ScheduleResponse) HasName() bool`

HasName returns a boolean if a field has been set.

### GetNamespace

`func (o *ScheduleResponse) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *ScheduleResponse) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *ScheduleResponse) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *ScheduleResponse) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetNextRunTime

`func (o *ScheduleResponse) GetNextRunTime() string`

GetNextRunTime returns the NextRunTime field if non-nil, zero value otherwise.

### GetNextRunTimeOk

`func (o *ScheduleResponse) GetNextRunTimeOk() (*string, bool)`

GetNextRunTimeOk returns a tuple with the NextRunTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextRunTime

`func (o *ScheduleResponse) SetNextRunTime(v string)`

SetNextRunTime sets NextRunTime field to given value.

### HasNextRunTime

`func (o *ScheduleResponse) HasNextRunTime() bool`

HasNextRunTime returns a boolean if a field has been set.

### GetPhase

`func (o *ScheduleResponse) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *ScheduleResponse) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *ScheduleResponse) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *ScheduleResponse) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetRunCount

`func (o *ScheduleResponse) GetRunCount() int32`

GetRunCount returns the RunCount field if non-nil, zero value otherwise.

### GetRunCountOk

`func (o *ScheduleResponse) GetRunCountOk() (*int32, bool)`

GetRunCountOk returns a tuple with the RunCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRunCount

`func (o *ScheduleResponse) SetRunCount(v int32)`

SetRunCount sets RunCount field to given value.

### HasRunCount

`func (o *ScheduleResponse) HasRunCount() bool`

HasRunCount returns a boolean if a field has been set.

### GetSchedule

`func (o *ScheduleResponse) GetSchedule() string`

GetSchedule returns the Schedule field if non-nil, zero value otherwise.

### GetScheduleOk

`func (o *ScheduleResponse) GetScheduleOk() (*string, bool)`

GetScheduleOk returns a tuple with the Schedule field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedule

`func (o *ScheduleResponse) SetSchedule(v string)`

SetSchedule sets Schedule field to given value.

### HasSchedule

`func (o *ScheduleResponse) HasSchedule() bool`

HasSchedule returns a boolean if a field has been set.

### GetSuccessfulRuns

`func (o *ScheduleResponse) GetSuccessfulRuns() int32`

GetSuccessfulRuns returns the SuccessfulRuns field if non-nil, zero value otherwise.

### GetSuccessfulRunsOk

`func (o *ScheduleResponse) GetSuccessfulRunsOk() (*int32, bool)`

GetSuccessfulRunsOk returns a tuple with the SuccessfulRuns field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSuccessfulRuns

`func (o *ScheduleResponse) SetSuccessfulRuns(v int32)`

SetSuccessfulRuns sets SuccessfulRuns field to given value.

### HasSuccessfulRuns

`func (o *ScheduleResponse) HasSuccessfulRuns() bool`

HasSuccessfulRuns returns a boolean if a field has been set.

### GetTimezone

`func (o *ScheduleResponse) GetTimezone() string`

GetTimezone returns the Timezone field if non-nil, zero value otherwise.

### GetTimezoneOk

`func (o *ScheduleResponse) GetTimezoneOk() (*string, bool)`

GetTimezoneOk returns a tuple with the Timezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimezone

`func (o *ScheduleResponse) SetTimezone(v string)`

SetTimezone sets Timezone field to given value.

### HasTimezone

`func (o *ScheduleResponse) HasTimezone() bool`

HasTimezone returns a boolean if a field has been set.

### GetTotalCostUSD

`func (o *ScheduleResponse) GetTotalCostUSD() string`

GetTotalCostUSD returns the TotalCostUSD field if non-nil, zero value otherwise.

### GetTotalCostUSDOk

`func (o *ScheduleResponse) GetTotalCostUSDOk() (*string, bool)`

GetTotalCostUSDOk returns a tuple with the TotalCostUSD field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalCostUSD

`func (o *ScheduleResponse) SetTotalCostUSD(v string)`

SetTotalCostUSD sets TotalCostUSD field to given value.

### HasTotalCostUSD

`func (o *ScheduleResponse) HasTotalCostUSD() bool`

HasTotalCostUSD returns a boolean if a field has been set.

### GetTotalTokens

`func (o *ScheduleResponse) GetTotalTokens() int32`

GetTotalTokens returns the TotalTokens field if non-nil, zero value otherwise.

### GetTotalTokensOk

`func (o *ScheduleResponse) GetTotalTokensOk() (*int32, bool)`

GetTotalTokensOk returns a tuple with the TotalTokens field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalTokens

`func (o *ScheduleResponse) SetTotalTokens(v int32)`

SetTotalTokens sets TotalTokens field to given value.

### HasTotalTokens

`func (o *ScheduleResponse) HasTotalTokens() bool`

HasTotalTokens returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


