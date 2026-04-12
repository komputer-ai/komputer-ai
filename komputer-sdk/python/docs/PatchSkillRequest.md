# PatchSkillRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**content** | **str** |  | [optional] 
**description** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.patch_skill_request import PatchSkillRequest

# TODO update the JSON string below
json = "{}"
# create an instance of PatchSkillRequest from a JSON string
patch_skill_request_instance = PatchSkillRequest.from_json(json)
# print the JSON string representation of the object
print(PatchSkillRequest.to_json())

# convert the object into a dict
patch_skill_request_dict = patch_skill_request_instance.to_dict()
# create an instance of PatchSkillRequest from a dict
patch_skill_request_from_dict = PatchSkillRequest.from_dict(patch_skill_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


