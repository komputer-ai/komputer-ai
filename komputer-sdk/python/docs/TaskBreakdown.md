# TaskBreakdown


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cache_create_tokens** | **int** |  | [optional] 
**cache_read_tokens** | **int** |  | [optional] 
**completed_at** | **str** |  | [optional] 
**cost_usd** | **float** |  | [optional] 
**duration_ms** | **int** |  | [optional] 
**events** | [**List[AgentEvent]**](AgentEvent.md) |  | [optional] 
**index** | **int** |  | [optional] 
**input_tokens** | **int** |  | [optional] 
**instruction** | **str** |  | [optional] 
**output_tokens** | **int** |  | [optional] 
**started_at** | **str** |  | [optional] 
**steer** | **bool** |  | [optional] 
**turns** | **int** |  | [optional] 

## Example

```python
from komputer_ai.models.task_breakdown import TaskBreakdown

# TODO update the JSON string below
json = "{}"
# create an instance of TaskBreakdown from a JSON string
task_breakdown_instance = TaskBreakdown.from_json(json)
# print the JSON string representation of the object
print(TaskBreakdown.to_json())

# convert the object into a dict
task_breakdown_dict = task_breakdown_instance.to_dict()
# create an instance of TaskBreakdown from a dict
task_breakdown_from_dict = TaskBreakdown.from_dict(task_breakdown_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


