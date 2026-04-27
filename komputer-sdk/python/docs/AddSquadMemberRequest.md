# AddSquadMemberRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name is the desired KomputerAgent name when Spec is set. Optional. | [optional] 
**ref** | [**V1alpha1KomputerSquadMemberRef**](V1alpha1KomputerSquadMemberRef.md) |  | [optional] 
**spec** | [**V1alpha1KomputerAgentSpec**](V1alpha1KomputerAgentSpec.md) |  | [optional] 

## Example

```python
from komputer_ai.models.add_squad_member_request import AddSquadMemberRequest

# TODO update the JSON string below
json = "{}"
# create an instance of AddSquadMemberRequest from a JSON string
add_squad_member_request_instance = AddSquadMemberRequest.from_json(json)
# print the JSON string representation of the object
print(AddSquadMemberRequest.to_json())

# convert the object into a dict
add_squad_member_request_dict = add_squad_member_request_instance.to_dict()
# create an instance of AddSquadMemberRequest from a dict
add_squad_member_request_from_dict = AddSquadMemberRequest.from_dict(add_squad_member_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


