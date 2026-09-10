# CreateScheduleRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Agent** | Pointer to [**V1alpha1ScheduleAgentSpec**](V1alpha1ScheduleAgentSpec.md) |  | [optional] 
**AgentName** | Pointer to **string** |  | [optional] 
**AutoDelete** | Pointer to **bool** |  | [optional] 
**Instructions** | **string** |  | 
**KeepAgents** | Pointer to **bool** |  | [optional] 
**Name** | **string** |  | 
**Namespace** | Pointer to **string** |  | [optional] 
**Schedule** | **string** |  | 
**Timezone** | Pointer to **string** |  | [optional] 

## Methods

### NewCreateScheduleRequest

`func NewCreateScheduleRequest(instructions string, name string, schedule string, ) *CreateScheduleRequest`

NewCreateScheduleRequest instantiates a new CreateScheduleRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateScheduleRequestWithDefaults

`func NewCreateScheduleRequestWithDefaults() *CreateScheduleRequest`

NewCreateScheduleRequestWithDefaults instantiates a new CreateScheduleRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAgent

`func (o *CreateScheduleRequest) GetAgent() V1alpha1ScheduleAgentSpec`

GetAgent returns the Agent field if non-nil, zero value otherwise.

### GetAgentOk

`func (o *CreateScheduleRequest) GetAgentOk() (*V1alpha1ScheduleAgentSpec, bool)`

GetAgentOk returns a tuple with the Agent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgent

`func (o *CreateScheduleRequest) SetAgent(v V1alpha1ScheduleAgentSpec)`

SetAgent sets Agent field to given value.

### HasAgent

`func (o *CreateScheduleRequest) HasAgent() bool`

HasAgent returns a boolean if a field has been set.

### GetAgentName

`func (o *CreateScheduleRequest) GetAgentName() string`

GetAgentName returns the AgentName field if non-nil, zero value otherwise.

### GetAgentNameOk

`func (o *CreateScheduleRequest) GetAgentNameOk() (*string, bool)`

GetAgentNameOk returns a tuple with the AgentName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentName

`func (o *CreateScheduleRequest) SetAgentName(v string)`

SetAgentName sets AgentName field to given value.

### HasAgentName

`func (o *CreateScheduleRequest) HasAgentName() bool`

HasAgentName returns a boolean if a field has been set.

### GetAutoDelete

`func (o *CreateScheduleRequest) GetAutoDelete() bool`

GetAutoDelete returns the AutoDelete field if non-nil, zero value otherwise.

### GetAutoDeleteOk

`func (o *CreateScheduleRequest) GetAutoDeleteOk() (*bool, bool)`

GetAutoDeleteOk returns a tuple with the AutoDelete field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAutoDelete

`func (o *CreateScheduleRequest) SetAutoDelete(v bool)`

SetAutoDelete sets AutoDelete field to given value.

### HasAutoDelete

`func (o *CreateScheduleRequest) HasAutoDelete() bool`

HasAutoDelete returns a boolean if a field has been set.

### GetInstructions

`func (o *CreateScheduleRequest) GetInstructions() string`

GetInstructions returns the Instructions field if non-nil, zero value otherwise.

### GetInstructionsOk

`func (o *CreateScheduleRequest) GetInstructionsOk() (*string, bool)`

GetInstructionsOk returns a tuple with the Instructions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstructions

`func (o *CreateScheduleRequest) SetInstructions(v string)`

SetInstructions sets Instructions field to given value.


### GetKeepAgents

`func (o *CreateScheduleRequest) GetKeepAgents() bool`

GetKeepAgents returns the KeepAgents field if non-nil, zero value otherwise.

### GetKeepAgentsOk

`func (o *CreateScheduleRequest) GetKeepAgentsOk() (*bool, bool)`

GetKeepAgentsOk returns a tuple with the KeepAgents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetKeepAgents

`func (o *CreateScheduleRequest) SetKeepAgents(v bool)`

SetKeepAgents sets KeepAgents field to given value.

### HasKeepAgents

`func (o *CreateScheduleRequest) HasKeepAgents() bool`

HasKeepAgents returns a boolean if a field has been set.

### GetName

`func (o *CreateScheduleRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateScheduleRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateScheduleRequest) SetName(v string)`

SetName sets Name field to given value.


### GetNamespace

`func (o *CreateScheduleRequest) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *CreateScheduleRequest) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *CreateScheduleRequest) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *CreateScheduleRequest) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetSchedule

`func (o *CreateScheduleRequest) GetSchedule() string`

GetSchedule returns the Schedule field if non-nil, zero value otherwise.

### GetScheduleOk

`func (o *CreateScheduleRequest) GetScheduleOk() (*string, bool)`

GetScheduleOk returns a tuple with the Schedule field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchedule

`func (o *CreateScheduleRequest) SetSchedule(v string)`

SetSchedule sets Schedule field to given value.


### GetTimezone

`func (o *CreateScheduleRequest) GetTimezone() string`

GetTimezone returns the Timezone field if non-nil, zero value otherwise.

### GetTimezoneOk

`func (o *CreateScheduleRequest) GetTimezoneOk() (*string, bool)`

GetTimezoneOk returns a tuple with the Timezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimezone

`func (o *CreateScheduleRequest) SetTimezone(v string)`

SetTimezone sets Timezone field to given value.

### HasTimezone

`func (o *CreateScheduleRequest) HasTimezone() bool`

HasTimezone returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


