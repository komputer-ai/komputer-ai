# ScheduleListResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**schedules** | [**List[ScheduleResponse]**](ScheduleResponse.md) |  | [optional] 

## Example

```python
from komputer_ai.models.schedule_list_response import ScheduleListResponse

# TODO update the JSON string below
json = "{}"
# create an instance of ScheduleListResponse from a JSON string
schedule_list_response_instance = ScheduleListResponse.from_json(json)
# print the JSON string representation of the object
print(ScheduleListResponse.to_json())

# convert the object into a dict
schedule_list_response_dict = schedule_list_response_instance.to_dict()
# create an instance of ScheduleListResponse from a dict
schedule_list_response_from_dict = ScheduleListResponse.from_dict(schedule_list_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


