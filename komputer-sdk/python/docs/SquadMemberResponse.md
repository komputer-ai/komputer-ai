# SquadMemberResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** |  | [optional] 
**ready** | **bool** |  | [optional] 
**task_status** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.squad_member_response import SquadMemberResponse

# TODO update the JSON string below
json = "{}"
# create an instance of SquadMemberResponse from a JSON string
squad_member_response_instance = SquadMemberResponse.from_json(json)
# print the JSON string representation of the object
print(SquadMemberResponse.to_json())

# convert the object into a dict
squad_member_response_dict = squad_member_response_instance.to_dict()
# create an instance of SquadMemberResponse from a dict
squad_member_response_from_dict = SquadMemberResponse.from_dict(squad_member_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


