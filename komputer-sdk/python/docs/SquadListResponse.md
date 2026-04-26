# SquadListResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**squads** | [**List[SquadResponse]**](SquadResponse.md) |  | [optional] 

## Example

```python
from komputer_ai.models.squad_list_response import SquadListResponse

# TODO update the JSON string below
json = "{}"
# create an instance of SquadListResponse from a JSON string
squad_list_response_instance = SquadListResponse.from_json(json)
# print the JSON string representation of the object
print(SquadListResponse.to_json())

# convert the object into a dict
squad_list_response_dict = squad_list_response_instance.to_dict()
# create an instance of SquadListResponse from a dict
squad_list_response_from_dict = SquadListResponse.from_dict(squad_list_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


