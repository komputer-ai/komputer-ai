# MemoryResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_names** | **List[str]** |  | [optional] 
**attached_agents** | **int** |  | [optional] 
**content** | **str** |  | [optional] 
**created_at** | **str** |  | [optional] 
**description** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.memory_response import MemoryResponse

# TODO update the JSON string below
json = "{}"
# create an instance of MemoryResponse from a JSON string
memory_response_instance = MemoryResponse.from_json(json)
# print the JSON string representation of the object
print(MemoryResponse.to_json())

# convert the object into a dict
memory_response_dict = memory_response_instance.to_dict()
# create an instance of MemoryResponse from a dict
memory_response_from_dict = MemoryResponse.from_dict(memory_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


