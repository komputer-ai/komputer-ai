# AgentListResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agents** | [**List[AgentResponse]**](AgentResponse.md) |  | [optional] 
**facets** | [**AgentListFacets**](AgentListFacets.md) | Facets lists every distinct namespace and \&quot;key&#x3D;value\&quot; label pair across agents in the requested namespace scope, computed BEFORE status/label/q filtering — see AgentListFacets doc. | [optional] 
**next_offset** | **int** | NextOffset is the offset to request for the next page; omitted once there is no more data (or when &#x60;limit&#x60; was omitted). | [optional] 
**total** | **int** | Total is the number of agents matching the status/label/q filters, BEFORE the limit/offset slice is applied. Always populated (even when &#x60;limit&#x60; was omitted, in which case it simply equals len(Agents)). | [optional] 

## Example

```python
from komputer_ai.models.agent_list_response import AgentListResponse

# TODO update the JSON string below
json = "{}"
# create an instance of AgentListResponse from a JSON string
agent_list_response_instance = AgentListResponse.from_json(json)
# print the JSON string representation of the object
print(AgentListResponse.to_json())

# convert the object into a dict
agent_list_response_dict = agent_list_response_instance.to_dict()
# create an instance of AgentListResponse from a dict
agent_list_response_from_dict = AgentListResponse.from_dict(agent_list_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


