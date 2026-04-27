# SquadResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CreatedAt** | Pointer to **string** |  | [optional] 
**Members** | Pointer to [**[]SquadMemberResponse**](SquadMemberResponse.md) |  | [optional] 
**Message** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Namespace** | Pointer to **string** |  | [optional] 
**OrphanTTL** | Pointer to **string** |  | [optional] 
**OrphanedSince** | Pointer to **string** |  | [optional] 
**Phase** | Pointer to **string** |  | [optional] 
**PodName** | Pointer to **string** |  | [optional] 

## Methods

### NewSquadResponse

`func NewSquadResponse() *SquadResponse`

NewSquadResponse instantiates a new SquadResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSquadResponseWithDefaults

`func NewSquadResponseWithDefaults() *SquadResponse`

NewSquadResponseWithDefaults instantiates a new SquadResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCreatedAt

`func (o *SquadResponse) GetCreatedAt() string`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *SquadResponse) GetCreatedAtOk() (*string, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *SquadResponse) SetCreatedAt(v string)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *SquadResponse) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetMembers

`func (o *SquadResponse) GetMembers() []SquadMemberResponse`

GetMembers returns the Members field if non-nil, zero value otherwise.

### GetMembersOk

`func (o *SquadResponse) GetMembersOk() (*[]SquadMemberResponse, bool)`

GetMembersOk returns a tuple with the Members field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMembers

`func (o *SquadResponse) SetMembers(v []SquadMemberResponse)`

SetMembers sets Members field to given value.

### HasMembers

`func (o *SquadResponse) HasMembers() bool`

HasMembers returns a boolean if a field has been set.

### GetMessage

`func (o *SquadResponse) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *SquadResponse) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *SquadResponse) SetMessage(v string)`

SetMessage sets Message field to given value.

### HasMessage

`func (o *SquadResponse) HasMessage() bool`

HasMessage returns a boolean if a field has been set.

### GetName

`func (o *SquadResponse) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *SquadResponse) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *SquadResponse) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *SquadResponse) HasName() bool`

HasName returns a boolean if a field has been set.

### GetNamespace

`func (o *SquadResponse) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *SquadResponse) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *SquadResponse) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *SquadResponse) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetOrphanTTL

`func (o *SquadResponse) GetOrphanTTL() string`

GetOrphanTTL returns the OrphanTTL field if non-nil, zero value otherwise.

### GetOrphanTTLOk

`func (o *SquadResponse) GetOrphanTTLOk() (*string, bool)`

GetOrphanTTLOk returns a tuple with the OrphanTTL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrphanTTL

`func (o *SquadResponse) SetOrphanTTL(v string)`

SetOrphanTTL sets OrphanTTL field to given value.

### HasOrphanTTL

`func (o *SquadResponse) HasOrphanTTL() bool`

HasOrphanTTL returns a boolean if a field has been set.

### GetOrphanedSince

`func (o *SquadResponse) GetOrphanedSince() string`

GetOrphanedSince returns the OrphanedSince field if non-nil, zero value otherwise.

### GetOrphanedSinceOk

`func (o *SquadResponse) GetOrphanedSinceOk() (*string, bool)`

GetOrphanedSinceOk returns a tuple with the OrphanedSince field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrphanedSince

`func (o *SquadResponse) SetOrphanedSince(v string)`

SetOrphanedSince sets OrphanedSince field to given value.

### HasOrphanedSince

`func (o *SquadResponse) HasOrphanedSince() bool`

HasOrphanedSince returns a boolean if a field has been set.

### GetPhase

`func (o *SquadResponse) GetPhase() string`

GetPhase returns the Phase field if non-nil, zero value otherwise.

### GetPhaseOk

`func (o *SquadResponse) GetPhaseOk() (*string, bool)`

GetPhaseOk returns a tuple with the Phase field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPhase

`func (o *SquadResponse) SetPhase(v string)`

SetPhase sets Phase field to given value.

### HasPhase

`func (o *SquadResponse) HasPhase() bool`

HasPhase returns a boolean if a field has been set.

### GetPodName

`func (o *SquadResponse) GetPodName() string`

GetPodName returns the PodName field if non-nil, zero value otherwise.

### GetPodNameOk

`func (o *SquadResponse) GetPodNameOk() (*string, bool)`

GetPodNameOk returns a tuple with the PodName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodName

`func (o *SquadResponse) SetPodName(v string)`

SetPodName sets PodName field to given value.

### HasPodName

`func (o *SquadResponse) HasPodName() bool`

HasPodName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


