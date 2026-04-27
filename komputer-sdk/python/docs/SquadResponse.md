# SquadResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**break_up_requested** | **bool** |  | [optional] 
**created_at** | **str** |  | [optional] 
**members** | [**List[SquadMemberResponse]**](SquadMemberResponse.md) |  | [optional] 
**message** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**orphan_ttl** | **str** |  | [optional] 
**orphaned_since** | **str** |  | [optional] 
**phase** | **str** |  | [optional] 
**pod_name** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.squad_response import SquadResponse

# TODO update the JSON string below
json = "{}"
# create an instance of SquadResponse from a JSON string
squad_response_instance = SquadResponse.from_json(json)
# print the JSON string representation of the object
print(SquadResponse.to_json())

# convert the object into a dict
squad_response_dict = squad_response_instance.to_dict()
# create an instance of SquadResponse from a dict
squad_response_from_dict = SquadResponse.from_dict(squad_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


