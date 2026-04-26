# AddSquadMemberRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Ref** | Pointer to [**V1alpha1KomputerSquadMemberRef**](V1alpha1KomputerSquadMemberRef.md) |  | [optional] 
**Spec** | Pointer to [**V1alpha1KomputerAgentSpec**](V1alpha1KomputerAgentSpec.md) |  | [optional] 

## Methods

### NewAddSquadMemberRequest

`func NewAddSquadMemberRequest() *AddSquadMemberRequest`

NewAddSquadMemberRequest instantiates a new AddSquadMemberRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAddSquadMemberRequestWithDefaults

`func NewAddSquadMemberRequestWithDefaults() *AddSquadMemberRequest`

NewAddSquadMemberRequestWithDefaults instantiates a new AddSquadMemberRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRef

`func (o *AddSquadMemberRequest) GetRef() V1alpha1KomputerSquadMemberRef`

GetRef returns the Ref field if non-nil, zero value otherwise.

### GetRefOk

`func (o *AddSquadMemberRequest) GetRefOk() (*V1alpha1KomputerSquadMemberRef, bool)`

GetRefOk returns a tuple with the Ref field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRef

`func (o *AddSquadMemberRequest) SetRef(v V1alpha1KomputerSquadMemberRef)`

SetRef sets Ref field to given value.

### HasRef

`func (o *AddSquadMemberRequest) HasRef() bool`

HasRef returns a boolean if a field has been set.

### GetSpec

`func (o *AddSquadMemberRequest) GetSpec() V1alpha1KomputerAgentSpec`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *AddSquadMemberRequest) GetSpecOk() (*V1alpha1KomputerAgentSpec, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *AddSquadMemberRequest) SetSpec(v V1alpha1KomputerAgentSpec)`

SetSpec sets Spec field to given value.

### HasSpec

`func (o *AddSquadMemberRequest) HasSpec() bool`

HasSpec returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


