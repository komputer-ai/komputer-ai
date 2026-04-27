# PatchSquadRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**members** | [**List[V1alpha1KomputerSquadMember]**](V1alpha1KomputerSquadMember.md) |  | [optional] 
**orphan_ttl** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.patch_squad_request import PatchSquadRequest

# TODO update the JSON string below
json = "{}"
# create an instance of PatchSquadRequest from a JSON string
patch_squad_request_instance = PatchSquadRequest.from_json(json)
# print the JSON string representation of the object
print(PatchSquadRequest.to_json())

# convert the object into a dict
patch_squad_request_dict = patch_squad_request_instance.to_dict()
# create an instance of PatchSquadRequest from a dict
patch_squad_request_from_dict = PatchSquadRequest.from_dict(patch_squad_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


