# PatchSquadRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Members** | Pointer to [**[]V1alpha1KomputerSquadMember**](V1alpha1KomputerSquadMember.md) |  | [optional] 
**OrphanTTL** | Pointer to **string** |  | [optional] 

## Methods

### NewPatchSquadRequest

`func NewPatchSquadRequest() *PatchSquadRequest`

NewPatchSquadRequest instantiates a new PatchSquadRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPatchSquadRequestWithDefaults

`func NewPatchSquadRequestWithDefaults() *PatchSquadRequest`

NewPatchSquadRequestWithDefaults instantiates a new PatchSquadRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMembers

`func (o *PatchSquadRequest) GetMembers() []V1alpha1KomputerSquadMember`

GetMembers returns the Members field if non-nil, zero value otherwise.

### GetMembersOk

`func (o *PatchSquadRequest) GetMembersOk() (*[]V1alpha1KomputerSquadMember, bool)`

GetMembersOk returns a tuple with the Members field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMembers

`func (o *PatchSquadRequest) SetMembers(v []V1alpha1KomputerSquadMember)`

SetMembers sets Members field to given value.

### HasMembers

`func (o *PatchSquadRequest) HasMembers() bool`

HasMembers returns a boolean if a field has been set.

### GetOrphanTTL

`func (o *PatchSquadRequest) GetOrphanTTL() string`

GetOrphanTTL returns the OrphanTTL field if non-nil, zero value otherwise.

### GetOrphanTTLOk

`func (o *PatchSquadRequest) GetOrphanTTLOk() (*string, bool)`

GetOrphanTTLOk returns a tuple with the OrphanTTL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrphanTTL

`func (o *PatchSquadRequest) SetOrphanTTL(v string)`

SetOrphanTTL sets OrphanTTL field to given value.

### HasOrphanTTL

`func (o *PatchSquadRequest) HasOrphanTTL() bool`

HasOrphanTTL returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


