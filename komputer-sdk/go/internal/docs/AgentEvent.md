# AgentEvent

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AgentName** | Pointer to **string** |  | [optional] 
**Namespace** | Pointer to **string** |  | [optional] 
**Payload** | Pointer to **map[string]interface{}** |  | [optional] 
**Timestamp** | Pointer to **string** |  | [optional] 
**Type** | Pointer to **string** |  | [optional] 

## Methods

### NewAgentEvent

`func NewAgentEvent() *AgentEvent`

NewAgentEvent instantiates a new AgentEvent object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentEventWithDefaults

`func NewAgentEventWithDefaults() *AgentEvent`

NewAgentEventWithDefaults instantiates a new AgentEvent object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAgentName

`func (o *AgentEvent) GetAgentName() string`

GetAgentName returns the AgentName field if non-nil, zero value otherwise.

### GetAgentNameOk

`func (o *AgentEvent) GetAgentNameOk() (*string, bool)`

GetAgentNameOk returns a tuple with the AgentName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentName

`func (o *AgentEvent) SetAgentName(v string)`

SetAgentName sets AgentName field to given value.

### HasAgentName

`func (o *AgentEvent) HasAgentName() bool`

HasAgentName returns a boolean if a field has been set.

### GetNamespace

`func (o *AgentEvent) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *AgentEvent) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *AgentEvent) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *AgentEvent) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetPayload

`func (o *AgentEvent) GetPayload() map[string]interface{}`

GetPayload returns the Payload field if non-nil, zero value otherwise.

### GetPayloadOk

`func (o *AgentEvent) GetPayloadOk() (*map[string]interface{}, bool)`

GetPayloadOk returns a tuple with the Payload field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPayload

`func (o *AgentEvent) SetPayload(v map[string]interface{})`

SetPayload sets Payload field to given value.

### HasPayload

`func (o *AgentEvent) HasPayload() bool`

HasPayload returns a boolean if a field has been set.

### GetTimestamp

`func (o *AgentEvent) GetTimestamp() string`

GetTimestamp returns the Timestamp field if non-nil, zero value otherwise.

### GetTimestampOk

`func (o *AgentEvent) GetTimestampOk() (*string, bool)`

GetTimestampOk returns a tuple with the Timestamp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimestamp

`func (o *AgentEvent) SetTimestamp(v string)`

SetTimestamp sets Timestamp field to given value.

### HasTimestamp

`func (o *AgentEvent) HasTimestamp() bool`

HasTimestamp returns a boolean if a field has been set.

### GetType

`func (o *AgentEvent) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *AgentEvent) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *AgentEvent) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *AgentEvent) HasType() bool`

HasType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


