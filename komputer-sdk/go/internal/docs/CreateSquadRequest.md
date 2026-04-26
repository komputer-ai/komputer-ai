# CreateSquadRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Members** | [**[]V1alpha1KomputerSquadMember**](V1alpha1KomputerSquadMember.md) |  | 
**Name** | **string** |  | 
**Namespace** | Pointer to **string** |  | [optional] 
**OrphanTTL** | Pointer to **string** | duration string e.g. \&quot;10m\&quot; | [optional] 

## Methods

### NewCreateSquadRequest

`func NewCreateSquadRequest(members []V1alpha1KomputerSquadMember, name string, ) *CreateSquadRequest`

NewCreateSquadRequest instantiates a new CreateSquadRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateSquadRequestWithDefaults

`func NewCreateSquadRequestWithDefaults() *CreateSquadRequest`

NewCreateSquadRequestWithDefaults instantiates a new CreateSquadRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMembers

`func (o *CreateSquadRequest) GetMembers() []V1alpha1KomputerSquadMember`

GetMembers returns the Members field if non-nil, zero value otherwise.

### GetMembersOk

`func (o *CreateSquadRequest) GetMembersOk() (*[]V1alpha1KomputerSquadMember, bool)`

GetMembersOk returns a tuple with the Members field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMembers

`func (o *CreateSquadRequest) SetMembers(v []V1alpha1KomputerSquadMember)`

SetMembers sets Members field to given value.


### GetName

`func (o *CreateSquadRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateSquadRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateSquadRequest) SetName(v string)`

SetName sets Name field to given value.


### GetNamespace

`func (o *CreateSquadRequest) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *CreateSquadRequest) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *CreateSquadRequest) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *CreateSquadRequest) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetOrphanTTL

`func (o *CreateSquadRequest) GetOrphanTTL() string`

GetOrphanTTL returns the OrphanTTL field if non-nil, zero value otherwise.

### GetOrphanTTLOk

`func (o *CreateSquadRequest) GetOrphanTTLOk() (*string, bool)`

GetOrphanTTLOk returns a tuple with the OrphanTTL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrphanTTL

`func (o *CreateSquadRequest) SetOrphanTTL(v string)`

SetOrphanTTL sets OrphanTTL field to given value.

### HasOrphanTTL

`func (o *CreateSquadRequest) HasOrphanTTL() bool`

HasOrphanTTL returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


