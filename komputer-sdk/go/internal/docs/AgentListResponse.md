# AgentListResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Agents** | Pointer to [**[]AgentResponse**](AgentResponse.md) |  | [optional] 
**Facets** | Pointer to [**AgentListFacets**](AgentListFacets.md) | Facets lists every distinct namespace and \&quot;key&#x3D;value\&quot; label pair across agents in the requested namespace scope, computed BEFORE status/label/q filtering — see AgentListFacets doc. | [optional] 
**NextOffset** | Pointer to **int32** | NextOffset is the offset to request for the next page; omitted once there is no more data (or when &#x60;limit&#x60; was omitted). | [optional] 
**Total** | Pointer to **int32** | Total is the number of agents matching the status/label/q filters, BEFORE the limit/offset slice is applied. Always populated (even when &#x60;limit&#x60; was omitted, in which case it simply equals len(Agents)). | [optional] 

## Methods

### NewAgentListResponse

`func NewAgentListResponse() *AgentListResponse`

NewAgentListResponse instantiates a new AgentListResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAgentListResponseWithDefaults

`func NewAgentListResponseWithDefaults() *AgentListResponse`

NewAgentListResponseWithDefaults instantiates a new AgentListResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAgents

`func (o *AgentListResponse) GetAgents() []AgentResponse`

GetAgents returns the Agents field if non-nil, zero value otherwise.

### GetAgentsOk

`func (o *AgentListResponse) GetAgentsOk() (*[]AgentResponse, bool)`

GetAgentsOk returns a tuple with the Agents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgents

`func (o *AgentListResponse) SetAgents(v []AgentResponse)`

SetAgents sets Agents field to given value.

### HasAgents

`func (o *AgentListResponse) HasAgents() bool`

HasAgents returns a boolean if a field has been set.

### GetFacets

`func (o *AgentListResponse) GetFacets() AgentListFacets`

GetFacets returns the Facets field if non-nil, zero value otherwise.

### GetFacetsOk

`func (o *AgentListResponse) GetFacetsOk() (*AgentListFacets, bool)`

GetFacetsOk returns a tuple with the Facets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFacets

`func (o *AgentListResponse) SetFacets(v AgentListFacets)`

SetFacets sets Facets field to given value.

### HasFacets

`func (o *AgentListResponse) HasFacets() bool`

HasFacets returns a boolean if a field has been set.

### GetNextOffset

`func (o *AgentListResponse) GetNextOffset() int32`

GetNextOffset returns the NextOffset field if non-nil, zero value otherwise.

### GetNextOffsetOk

`func (o *AgentListResponse) GetNextOffsetOk() (*int32, bool)`

GetNextOffsetOk returns a tuple with the NextOffset field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextOffset

`func (o *AgentListResponse) SetNextOffset(v int32)`

SetNextOffset sets NextOffset field to given value.

### HasNextOffset

`func (o *AgentListResponse) HasNextOffset() bool`

HasNextOffset returns a boolean if a field has been set.

### GetTotal

`func (o *AgentListResponse) GetTotal() int32`

GetTotal returns the Total field if non-nil, zero value otherwise.

### GetTotalOk

`func (o *AgentListResponse) GetTotalOk() (*int32, bool)`

GetTotalOk returns a tuple with the Total field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotal

`func (o *AgentListResponse) SetTotal(v int32)`

SetTotal sets Total field to given value.

### HasTotal

`func (o *AgentListResponse) HasTotal() bool`

HasTotal returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


