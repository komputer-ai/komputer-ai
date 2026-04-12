# CreateScheduleRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent** | [**CreateScheduleAgentSpec**](CreateScheduleAgentSpec.md) |  | [optional] 
**agent_name** | **str** |  | [optional] 
**auto_delete** | **bool** |  | [optional] 
**instructions** | **str** |  | 
**keep_agents** | **bool** |  | [optional] 
**name** | **str** |  | 
**namespace** | **str** |  | [optional] 
**schedule** | **str** |  | 
**timezone** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.create_schedule_request import CreateScheduleRequest

# TODO update the JSON string below
json = "{}"
# create an instance of CreateScheduleRequest from a JSON string
create_schedule_request_instance = CreateScheduleRequest.from_json(json)
# print the JSON string representation of the object
print(CreateScheduleRequest.to_json())

# convert the object into a dict
create_schedule_request_dict = create_schedule_request_instance.to_dict()
# create an instance of CreateScheduleRequest from a dict
create_schedule_request_from_dict = CreateScheduleRequest.from_dict(create_schedule_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


