# AgentListStats


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**by_status** | **Dict[str, int]** | ByStatus counts agents per phase. Agents with no phase yet aren&#39;t counted. | [optional] 
**total_cost_usd** | **str** | TotalCostUSD is the summed totalCostUSD, formatted like the agent field. | [optional] 

## Example

```python
from komputer_ai.models.agent_list_stats import AgentListStats

# TODO update the JSON string below
json = "{}"
# create an instance of AgentListStats from a JSON string
agent_list_stats_instance = AgentListStats.from_json(json)
# print the JSON string representation of the object
print(AgentListStats.to_json())

# convert the object into a dict
agent_list_stats_dict = agent_list_stats_instance.to_dict()
# create an instance of AgentListStats from a dict
agent_list_stats_from_dict = AgentListStats.from_dict(agent_list_stats_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


