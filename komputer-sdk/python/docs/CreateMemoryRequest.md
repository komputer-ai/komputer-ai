# CreateMemoryRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**content** | **str** |  | 
**description** | **str** |  | [optional] 
**name** | **str** |  | 
**namespace** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.create_memory_request import CreateMemoryRequest

# TODO update the JSON string below
json = "{}"
# create an instance of CreateMemoryRequest from a JSON string
create_memory_request_instance = CreateMemoryRequest.from_json(json)
# print the JSON string representation of the object
print(CreateMemoryRequest.to_json())

# convert the object into a dict
create_memory_request_dict = create_memory_request_instance.to_dict()
# create an instance of CreateMemoryRequest from a dict
create_memory_request_from_dict = CreateMemoryRequest.from_dict(create_memory_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


