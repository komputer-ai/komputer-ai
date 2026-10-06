# UpdateConnectorRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Disabled** | Pointer to **bool** | Disabled toggles whether the connector can be newly attached to agents and whether its tools are resolved for agent pods. true disables, false re-enables. Any caller may toggle this — there is no additional authorization check. | [optional] 
**Namespace** | Pointer to **string** |  | [optional] 
**Token** | Pointer to **string** | new auth token; replaces the value in the connector&#39;s secret | [optional] 

## Methods

### NewUpdateConnectorRequest

`func NewUpdateConnectorRequest() *UpdateConnectorRequest`

NewUpdateConnectorRequest instantiates a new UpdateConnectorRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpdateConnectorRequestWithDefaults

`func NewUpdateConnectorRequestWithDefaults() *UpdateConnectorRequest`

NewUpdateConnectorRequestWithDefaults instantiates a new UpdateConnectorRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDisabled

`func (o *UpdateConnectorRequest) GetDisabled() bool`

GetDisabled returns the Disabled field if non-nil, zero value otherwise.

### GetDisabledOk

`func (o *UpdateConnectorRequest) GetDisabledOk() (*bool, bool)`

GetDisabledOk returns a tuple with the Disabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisabled

`func (o *UpdateConnectorRequest) SetDisabled(v bool)`

SetDisabled sets Disabled field to given value.

### HasDisabled

`func (o *UpdateConnectorRequest) HasDisabled() bool`

HasDisabled returns a boolean if a field has been set.

### GetNamespace

`func (o *UpdateConnectorRequest) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *UpdateConnectorRequest) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *UpdateConnectorRequest) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *UpdateConnectorRequest) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetToken

`func (o *UpdateConnectorRequest) GetToken() string`

GetToken returns the Token field if non-nil, zero value otherwise.

### GetTokenOk

`func (o *UpdateConnectorRequest) GetTokenOk() (*string, bool)`

GetTokenOk returns a tuple with the Token field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetToken

`func (o *UpdateConnectorRequest) SetToken(v string)`

SetToken sets Token field to given value.

### HasToken

`func (o *UpdateConnectorRequest) HasToken() bool`

HasToken returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


