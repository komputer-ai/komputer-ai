# ResourceQuantity


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**format** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.resource_quantity import ResourceQuantity

# TODO update the JSON string below
json = "{}"
# create an instance of ResourceQuantity from a JSON string
resource_quantity_instance = ResourceQuantity.from_json(json)
# print the JSON string representation of the object
print(ResourceQuantity.to_json())

# convert the object into a dict
resource_quantity_dict = resource_quantity_instance.to_dict()
# create an instance of ResourceQuantity from a dict
resource_quantity_from_dict = ResourceQuantity.from_dict(resource_quantity_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


