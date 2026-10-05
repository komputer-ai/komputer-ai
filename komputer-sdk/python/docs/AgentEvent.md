# AgentEvent


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**payload** | **Dict[str, object]** |  | [optional] 
**timestamp** | **str** |  | [optional] 
**type** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.agent_event import AgentEvent

# TODO update the JSON string below
json = "{}"
# create an instance of AgentEvent from a JSON string
agent_event_instance = AgentEvent.from_json(json)
# print the JSON string representation of the object
print(AgentEvent.to_json())

# convert the object into a dict
agent_event_dict = agent_event_instance.to_dict()
# create an instance of AgentEvent from a dict
agent_event_from_dict = AgentEvent.from_dict(agent_event_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


