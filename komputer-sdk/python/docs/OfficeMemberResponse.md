# OfficeMemberResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**last_task_cost_usd** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**role** | **str** |  | [optional] 
**task_status** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.office_member_response import OfficeMemberResponse

# TODO update the JSON string below
json = "{}"
# create an instance of OfficeMemberResponse from a JSON string
office_member_response_instance = OfficeMemberResponse.from_json(json)
# print the JSON string representation of the object
print(OfficeMemberResponse.to_json())

# convert the object into a dict
office_member_response_dict = office_member_response_instance.to_dict()
# create an instance of OfficeMemberResponse from a dict
office_member_response_from_dict = OfficeMemberResponse.from_dict(office_member_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


