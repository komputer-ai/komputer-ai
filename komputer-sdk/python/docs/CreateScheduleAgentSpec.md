# CreateScheduleAgentSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**lifecycle** | **str** |  | [optional] 
**model** | **str** |  | [optional] 
**role** | **str** |  | [optional] 
**secret_refs** | **List[str]** |  | [optional] 
**template_ref** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.create_schedule_agent_spec import CreateScheduleAgentSpec

# TODO update the JSON string below
json = "{}"
# create an instance of CreateScheduleAgentSpec from a JSON string
create_schedule_agent_spec_instance = CreateScheduleAgentSpec.from_json(json)
# print the JSON string representation of the object
print(CreateScheduleAgentSpec.to_json())

# convert the object into a dict
create_schedule_agent_spec_dict = create_schedule_agent_spec_instance.to_dict()
# create an instance of CreateScheduleAgentSpec from a dict
create_schedule_agent_spec_from_dict = CreateScheduleAgentSpec.from_dict(create_schedule_agent_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


