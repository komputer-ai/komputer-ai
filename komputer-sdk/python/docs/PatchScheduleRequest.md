# PatchScheduleRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**schedule** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.patch_schedule_request import PatchScheduleRequest

# TODO update the JSON string below
json = "{}"
# create an instance of PatchScheduleRequest from a JSON string
patch_schedule_request_instance = PatchScheduleRequest.from_json(json)
# print the JSON string representation of the object
print(PatchScheduleRequest.to_json())

# convert the object into a dict
patch_schedule_request_dict = patch_schedule_request_instance.to_dict()
# create an instance of PatchScheduleRequest from a dict
patch_schedule_request_from_dict = PatchScheduleRequest.from_dict(patch_schedule_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


