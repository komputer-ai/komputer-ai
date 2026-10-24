# SessionStoreResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Error** | Pointer to **string** |  | [optional] 
**LastSavedAt** | Pointer to **string** |  | [optional] 
**Ok** | Pointer to **bool** |  | [optional] 

## Methods

### NewSessionStoreResponse

`func NewSessionStoreResponse() *SessionStoreResponse`

NewSessionStoreResponse instantiates a new SessionStoreResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionStoreResponseWithDefaults

`func NewSessionStoreResponseWithDefaults() *SessionStoreResponse`

NewSessionStoreResponseWithDefaults instantiates a new SessionStoreResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetError

`func (o *SessionStoreResponse) GetError() string`

GetError returns the Error field if non-nil, zero value otherwise.

### GetErrorOk

`func (o *SessionStoreResponse) GetErrorOk() (*string, bool)`

GetErrorOk returns a tuple with the Error field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetError

`func (o *SessionStoreResponse) SetError(v string)`

SetError sets Error field to given value.

### HasError

`func (o *SessionStoreResponse) HasError() bool`

HasError returns a boolean if a field has been set.

### GetLastSavedAt

`func (o *SessionStoreResponse) GetLastSavedAt() string`

GetLastSavedAt returns the LastSavedAt field if non-nil, zero value otherwise.

### GetLastSavedAtOk

`func (o *SessionStoreResponse) GetLastSavedAtOk() (*string, bool)`

GetLastSavedAtOk returns a tuple with the LastSavedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastSavedAt

`func (o *SessionStoreResponse) SetLastSavedAt(v string)`

SetLastSavedAt sets LastSavedAt field to given value.

### HasLastSavedAt

`func (o *SessionStoreResponse) HasLastSavedAt() bool`

HasLastSavedAt returns a boolean if a field has been set.

### GetOk

`func (o *SessionStoreResponse) GetOk() bool`

GetOk returns the Ok field if non-nil, zero value otherwise.

### GetOkOk

`func (o *SessionStoreResponse) GetOkOk() (*bool, bool)`

GetOkOk returns a tuple with the Ok field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOk

`func (o *SessionStoreResponse) SetOk(v bool)`

SetOk sets Ok field to given value.

### HasOk

`func (o *SessionStoreResponse) HasOk() bool`

HasOk returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


