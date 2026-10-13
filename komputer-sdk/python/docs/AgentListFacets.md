# AgentListFacets


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**labels** | **List[str]** |  | [optional] 
**namespaces** | **List[str]** |  | [optional] 

## Example

```python
from komputer_ai.models.agent_list_facets import AgentListFacets

# TODO update the JSON string below
json = "{}"
# create an instance of AgentListFacets from a JSON string
agent_list_facets_instance = AgentListFacets.from_json(json)
# print the JSON string representation of the object
print(AgentListFacets.to_json())

# convert the object into a dict
agent_list_facets_dict = agent_list_facets_instance.to_dict()
# create an instance of AgentListFacets from a dict
agent_list_facets_from_dict = AgentListFacets.from_dict(agent_list_facets_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


