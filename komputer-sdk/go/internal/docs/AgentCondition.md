# AgentCondition

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LastTransitionTime** | Pointer to **string** | RFC3339 | [optional] 
**Message** | Pointer to **string** |  | [optional] 
**Reason** | Pointer to **string** |  | [optional] 
**Status** | Pointer to **string** | \&quot;True\&quot;, \&quot;False\&quot; or \&quot;Unknown\&quot; | [optional] 
**Type** | Pointer to **string** |  | [optional] 

## Methods

### NewAgentCondition

`func NewAgentCondition() *AgentCondition`

NewAgentCondition instantiates a new AgentCondition object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentConditionWithDefaults

`func NewAgentConditionWithDefaults() *AgentCondition`

NewAgentConditionWithDefaults instantiates a new AgentCondition object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLastTransitionTime

`func (o *AgentCondition) GetLastTransitionTime() string`

GetLastTransitionTime returns the LastTransitionTime field if non-nil, zero value otherwise.

### GetLastTransitionTimeOk

`func (o *AgentCondition) GetLastTransitionTimeOk() (*string, bool)`

GetLastTransitionTimeOk returns a tuple with the LastTransitionTime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastTransitionTime

`func (o *AgentCondition) SetLastTransitionTime(v string)`

SetLastTransitionTime sets LastTransitionTime field to given value.

### HasLastTransitionTime

`func (o *AgentCondition) HasLastTransitionTime() bool`

HasLastTransitionTime returns a boolean if a field has been set.

### GetMessage

`func (o *AgentCondition) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *AgentCondition) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *AgentCondition) SetMessage(v string)`

SetMessage sets Message field to given value.

### HasMessage

`func (o *AgentCondition) HasMessage() bool`

HasMessage returns a boolean if a field has been set.

### GetReason

`func (o *AgentCondition) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *AgentCondition) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *AgentCondition) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *AgentCondition) HasReason() bool`

HasReason returns a boolean if a field has been set.

### GetStatus

`func (o *AgentCondition) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *AgentCondition) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *AgentCondition) SetStatus(v string)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *AgentCondition) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### GetType

`func (o *AgentCondition) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *AgentCondition) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *AgentCondition) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *AgentCondition) HasType() bool`

HasType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


