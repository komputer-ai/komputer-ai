# ScheduleResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_name** | **str** |  | [optional] 
**auto_delete** | **bool** |  | [optional] 
**created_at** | **str** |  | [optional] 
**failed_runs** | **int** |  | [optional] 
**keep_agents** | **bool** |  | [optional] 
**last_run_cost_usd** | **str** |  | [optional] 
**last_run_status** | **str** |  | [optional] 
**last_run_time** | **str** |  | [optional] 
**last_run_tokens** | **int** |  | [optional] 
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**next_run_time** | **str** |  | [optional] 
**phase** | **str** |  | [optional] 
**run_count** | **int** |  | [optional] 
**schedule** | **str** |  | [optional] 
**successful_runs** | **int** |  | [optional] 
**timezone** | **str** |  | [optional] 
**total_cost_usd** | **str** |  | [optional] 
**total_tokens** | **int** |  | [optional] 

## Example

```python
from komputer_ai.models.schedule_response import ScheduleResponse

# TODO update the JSON string below
json = "{}"
# create an instance of ScheduleResponse from a JSON string
schedule_response_instance = ScheduleResponse.from_json(json)
# print the JSON string representation of the object
print(ScheduleResponse.to_json())

# convert the object into a dict
schedule_response_dict = schedule_response_instance.to_dict()
# create an instance of ScheduleResponse from a dict
schedule_response_from_dict = ScheduleResponse.from_dict(schedule_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


