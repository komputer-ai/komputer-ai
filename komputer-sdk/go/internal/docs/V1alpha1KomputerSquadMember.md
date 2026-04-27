# V1alpha1KomputerSquadMember

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name is the desired KomputerAgent name when Spec is provided. When empty, the operator generates \&quot;&lt;squad&gt;-member-&lt;index&gt;\&quot;. Ignored when Ref is set. +optional | [optional] 
**Ref** | Pointer to [**V1alpha1KomputerSquadMemberRef**](V1alpha1KomputerSquadMemberRef.md) | Exactly one of Ref or Spec must be set. | [optional] 
**Spec** | Pointer to [**V1alpha1KomputerAgentSpec**](V1alpha1KomputerAgentSpec.md) |  | [optional] 

## Methods

### NewV1alpha1KomputerSquadMember

`func NewV1alpha1KomputerSquadMember() *V1alpha1KomputerSquadMember`

NewV1alpha1KomputerSquadMember instantiates a new V1alpha1KomputerSquadMember object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1alpha1KomputerSquadMemberWithDefaults

`func NewV1alpha1KomputerSquadMemberWithDefaults() *V1alpha1KomputerSquadMember`

NewV1alpha1KomputerSquadMemberWithDefaults instantiates a new V1alpha1KomputerSquadMember object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *V1alpha1KomputerSquadMember) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1alpha1KomputerSquadMember) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1alpha1KomputerSquadMember) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1alpha1KomputerSquadMember) HasName() bool`

HasName returns a boolean if a field has been set.

### GetRef

`func (o *V1alpha1KomputerSquadMember) GetRef() V1alpha1KomputerSquadMemberRef`

GetRef returns the Ref field if non-nil, zero value otherwise.

### GetRefOk

`func (o *V1alpha1KomputerSquadMember) GetRefOk() (*V1alpha1KomputerSquadMemberRef, bool)`

GetRefOk returns a tuple with the Ref field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRef

`func (o *V1alpha1KomputerSquadMember) SetRef(v V1alpha1KomputerSquadMemberRef)`

SetRef sets Ref field to given value.

### HasRef

`func (o *V1alpha1KomputerSquadMember) HasRef() bool`

HasRef returns a boolean if a field has been set.

### GetSpec

`func (o *V1alpha1KomputerSquadMember) GetSpec() V1alpha1KomputerAgentSpec`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *V1alpha1KomputerSquadMember) GetSpecOk() (*V1alpha1KomputerAgentSpec, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *V1alpha1KomputerSquadMember) SetSpec(v V1alpha1KomputerAgentSpec)`

SetSpec sets Spec field to given value.

### HasSpec

`func (o *V1alpha1KomputerSquadMember) HasSpec() bool`

HasSpec returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


