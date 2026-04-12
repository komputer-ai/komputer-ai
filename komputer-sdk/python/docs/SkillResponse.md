# SkillResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_names** | **List[str]** |  | [optional] 
**attached_agents** | **int** |  | [optional] 
**content** | **str** |  | [optional] 
**created_at** | **str** |  | [optional] 
**description** | **str** |  | [optional] 
**is_default** | **bool** |  | [optional] 
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.skill_response import SkillResponse

# TODO update the JSON string below
json = "{}"
# create an instance of SkillResponse from a JSON string
skill_response_instance = SkillResponse.from_json(json)
# print the JSON string representation of the object
print(SkillResponse.to_json())

# convert the object into a dict
skill_response_dict = skill_response_instance.to_dict()
# create an instance of SkillResponse from a dict
skill_response_from_dict = SkillResponse.from_dict(skill_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


