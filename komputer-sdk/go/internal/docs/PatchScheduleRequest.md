# PatchScheduleRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Agent** | Pointer to [**CreateScheduleAgentSpec**](CreateScheduleAgentSpec.md) |  | [optional] 
**AgentName** | Pointer to **string** |  | [optional] 
**AutoDelete** | Pointer to **bool** |  | [optional] 
**Instructions** | Pointer to **string** |  | [optional] 
**KeepAgents** | Pointer to **bool** |  | [optional] 
**Schedule** | Pointer to **string** |  | [optional] 
**Suspended** | Pointer to **bool** |  | [optional] 
**Timezone** | Pointer to **string** |  | [optional] 

## Methods

### NewPatchScheduleRequest

`func NewPatchScheduleRequest() *PatchScheduleRequest`

NewPatchScheduleRequest instantiates a new PatchScheduleRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPatchScheduleRequestWithDefaults

`func NewPatchScheduleRequestWithDefaults() *PatchScheduleRequest`

NewPatchScheduleRequestWithDefaults instantiates a new PatchScheduleRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAgent

`func (o *PatchScheduleRequest) GetAgent() CreateScheduleAgentSpec`

GetAgent returns the Agent field if non-nil, zero value otherwise.

### GetAgentOk

`func (o *PatchScheduleRequest) GetAgentOk() (*CreateScheduleAgentSpec, bool)`

GetAgentOk returns a tuple with the Agent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgent

`func (o *PatchScheduleRequest) SetAgent(v CreateScheduleAgentSpec)`

SetAgent sets Agent field to given value.

### HasAgent

`func (o *PatchScheduleRequest) HasAgent() bool`

HasAgent returns a boolean if a field has been set.

### GetAgentName

`func (o *PatchScheduleRequest) GetAgentName() string`

GetAgentName returns the AgentName field if non-nil, zero value otherwise.

### GetAgentNameOk

`func (o *PatchScheduleRequest) GetAgentNameOk() (*string, bool)`

GetAgentNameOk returns a tuple with the AgentName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentName

`func (o *PatchScheduleRequest) SetAgentName(v string)`

SetAgentName sets AgentName field to given value.

### HasAgentName

`func (o *PatchScheduleRequest) HasAgentName() bool`

HasAgentName returns a boolean if a field has been set.

### GetAutoDelete

`func (o *PatchScheduleRequest) GetAutoDelete() bool`

GetAutoDelete returns the AutoDelete field if non-nil, zero value otherwise.

### GetAutoDeleteOk

`func (o *PatchScheduleRequest) GetAutoDeleteOk() (*bool, bool)`

GetAutoDeleteOk returns a tuple with the AutoDelete field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoDelete

`func (o *PatchScheduleRequest) SetAutoDelete(v bool)`

SetAutoDelete sets AutoDelete field to given value.

### HasAutoDelete

`func (o *PatchScheduleRequest) HasAutoDelete() bool`

HasAutoDelete returns a boolean if a field has been set.

### GetInstructions

`func (o *PatchScheduleRequest) GetInstructions() string`

GetInstructions returns the Instructions field if non-nil, zero value otherwise.

### GetInstructionsOk

`func (o *PatchScheduleRequest) GetInstructionsOk() (*string, bool)`

GetInstructionsOk returns a tuple with the Instructions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstructions

`func (o *PatchScheduleRequest) SetInstructions(v string)`

SetInstructions sets Instructions field to given value.

### HasInstructions

`func (o *PatchScheduleRequest) HasInstructions() bool`

HasInstructions returns a boolean if a field has been set.

### GetKeepAgents

`func (o *PatchScheduleRequest) GetKeepAgents() bool`

GetKeepAgents returns the KeepAgents field if non-nil, zero value otherwise.

### GetKeepAgentsOk

`func (o *PatchScheduleRequest) GetKeepAgentsOk() (*bool, bool)`

GetKeepAgentsOk returns a tuple with the KeepAgents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKeepAgents

`func (o *PatchScheduleRequest) SetKeepAgents(v bool)`

SetKeepAgents sets KeepAgents field to given value.

### HasKeepAgents

`func (o *PatchScheduleRequest) HasKeepAgents() bool`

HasKeepAgents returns a boolean if a field has been set.

### GetSchedule

`func (o *PatchScheduleRequest) GetSchedule() string`

GetSchedule returns the Schedule field if non-nil, zero value otherwise.

### GetScheduleOk

`func (o *PatchScheduleRequest) GetScheduleOk() (*string, bool)`

GetScheduleOk returns a tuple with the Schedule field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedule

`func (o *PatchScheduleRequest) SetSchedule(v string)`

SetSchedule sets Schedule field to given value.

### HasSchedule

`func (o *PatchScheduleRequest) HasSchedule() bool`

HasSchedule returns a boolean if a field has been set.

### GetSuspended

`func (o *PatchScheduleRequest) GetSuspended() bool`

GetSuspended returns the Suspended field if non-nil, zero value otherwise.

### GetSuspendedOk

`func (o *PatchScheduleRequest) GetSuspendedOk() (*bool, bool)`

GetSuspendedOk returns a tuple with the Suspended field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSuspended

`func (o *PatchScheduleRequest) SetSuspended(v bool)`

SetSuspended sets Suspended field to given value.

### HasSuspended

`func (o *PatchScheduleRequest) HasSuspended() bool`

HasSuspended returns a boolean if a field has been set.

### GetTimezone

`func (o *PatchScheduleRequest) GetTimezone() string`

GetTimezone returns the Timezone field if non-nil, zero value otherwise.

### GetTimezoneOk

`func (o *PatchScheduleRequest) GetTimezoneOk() (*string, bool)`

GetTimezoneOk returns a tuple with the Timezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimezone

`func (o *PatchScheduleRequest) SetTimezone(v string)`

SetTimezone sets Timezone field to given value.

### HasTimezone

`func (o *PatchScheduleRequest) HasTimezone() bool`

HasTimezone returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


