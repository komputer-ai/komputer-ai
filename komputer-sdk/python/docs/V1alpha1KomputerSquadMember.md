# V1alpha1KomputerSquadMember


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name is the desired KomputerAgent name when Spec is provided. When empty, the operator generates \&quot;&lt;squad&gt;-member-&lt;index&gt;\&quot;. Ignored when Ref is set. +optional | [optional] 
**ref** | [**V1alpha1KomputerSquadMemberRef**](V1alpha1KomputerSquadMemberRef.md) | Exactly one of Ref or Spec must be set. | [optional] 
**spec** | [**V1alpha1KomputerAgentSpec**](V1alpha1KomputerAgentSpec.md) |  | [optional] 

## Example

```python
from komputer_ai.models.v1alpha1_komputer_squad_member import V1alpha1KomputerSquadMember

# TODO update the JSON string below
json = "{}"
# create an instance of V1alpha1KomputerSquadMember from a JSON string
v1alpha1_komputer_squad_member_instance = V1alpha1KomputerSquadMember.from_json(json)
# print the JSON string representation of the object
print(V1alpha1KomputerSquadMember.to_json())

# convert the object into a dict
v1alpha1_komputer_squad_member_dict = v1alpha1_komputer_squad_member_instance.to_dict()
# create an instance of V1alpha1KomputerSquadMember from a dict
v1alpha1_komputer_squad_member_from_dict = V1alpha1KomputerSquadMember.from_dict(v1alpha1_komputer_squad_member_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


