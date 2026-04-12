# PatchMemoryRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**content** | **str** |  | [optional] 
**description** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.patch_memory_request import PatchMemoryRequest

# TODO update the JSON string below
json = "{}"
# create an instance of PatchMemoryRequest from a JSON string
patch_memory_request_instance = PatchMemoryRequest.from_json(json)
# print the JSON string representation of the object
print(PatchMemoryRequest.to_json())

# convert the object into a dict
patch_memory_request_dict = patch_memory_request_instance.to_dict()
# create an instance of PatchMemoryRequest from a dict
patch_memory_request_from_dict = PatchMemoryRequest.from_dict(patch_memory_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


