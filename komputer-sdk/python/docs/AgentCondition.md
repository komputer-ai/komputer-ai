# AgentCondition


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**last_transition_time** | **str** | RFC3339 | [optional] 
**message** | **str** |  | [optional] 
**reason** | **str** |  | [optional] 
**status** | **str** | \&quot;True\&quot;, \&quot;False\&quot; or \&quot;Unknown\&quot; | [optional] 
**type** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.agent_condition import AgentCondition

# TODO update the JSON string below
json = "{}"
# create an instance of AgentCondition from a JSON string
agent_condition_instance = AgentCondition.from_json(json)
# print the JSON string representation of the object
print(AgentCondition.to_json())

# convert the object into a dict
agent_condition_dict = agent_condition_instance.to_dict()
# create an instance of AgentCondition from a dict
agent_condition_from_dict = AgentCondition.from_dict(agent_condition_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


