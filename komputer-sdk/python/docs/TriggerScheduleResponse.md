# TriggerScheduleResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_name** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**status** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.trigger_schedule_response import TriggerScheduleResponse

# TODO update the JSON string below
json = "{}"
# create an instance of TriggerScheduleResponse from a JSON string
trigger_schedule_response_instance = TriggerScheduleResponse.from_json(json)
# print the JSON string representation of the object
print(TriggerScheduleResponse.to_json())

# convert the object into a dict
trigger_schedule_response_dict = trigger_schedule_response_instance.to_dict()
# create an instance of TriggerScheduleResponse from a dict
trigger_schedule_response_from_dict = TriggerScheduleResponse.from_dict(trigger_schedule_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


