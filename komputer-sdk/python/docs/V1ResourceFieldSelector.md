# V1ResourceFieldSelector


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**container_name** | **str** | Container name: required for volumes, optional for env vars +optional | [optional] 
**divisor** | [**ResourceQuantity**](ResourceQuantity.md) | Specifies the output format of the exposed resources, defaults to \&quot;1\&quot; +optional | [optional] 
**resource** | **str** | Required: resource to select | [optional] 

## Example

```python
from komputer_ai.models.v1_resource_field_selector import V1ResourceFieldSelector

# TODO update the JSON string below
json = "{}"
# create an instance of V1ResourceFieldSelector from a JSON string
v1_resource_field_selector_instance = V1ResourceFieldSelector.from_json(json)
# print the JSON string representation of the object
print(V1ResourceFieldSelector.to_json())

# convert the object into a dict
v1_resource_field_selector_dict = v1_resource_field_selector_instance.to_dict()
# create an instance of V1ResourceFieldSelector from a dict
v1_resource_field_selector_from_dict = V1ResourceFieldSelector.from_dict(v1_resource_field_selector_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


