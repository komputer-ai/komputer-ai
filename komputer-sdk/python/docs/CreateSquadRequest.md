# CreateSquadRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**members** | [**List[V1alpha1KomputerSquadMember]**](V1alpha1KomputerSquadMember.md) |  | 
**name** | **str** |  | 
**namespace** | **str** |  | [optional] 
**orphan_ttl** | **str** | duration string e.g. \&quot;10m\&quot; | [optional] 

## Example

```python
from komputer_ai.models.create_squad_request import CreateSquadRequest

# TODO update the JSON string below
json = "{}"
# create an instance of CreateSquadRequest from a JSON string
create_squad_request_instance = CreateSquadRequest.from_json(json)
# print the JSON string representation of the object
print(CreateSquadRequest.to_json())

# convert the object into a dict
create_squad_request_dict = create_squad_request_instance.to_dict()
# create an instance of CreateSquadRequest from a dict
create_squad_request_from_dict = CreateSquadRequest.from_dict(create_squad_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


