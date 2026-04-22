# IntstrIntOrString


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**int_val** | **int** |  | [optional] 
**str_val** | **str** |  | [optional] 
**type** | [**IntstrType**](IntstrType.md) |  | [optional] 

## Example

```python
from komputer_ai.models.intstr_int_or_string import IntstrIntOrString

# TODO update the JSON string below
json = "{}"
# create an instance of IntstrIntOrString from a JSON string
intstr_int_or_string_instance = IntstrIntOrString.from_json(json)
# print the JSON string representation of the object
print(IntstrIntOrString.to_json())

# convert the object into a dict
intstr_int_or_string_dict = intstr_int_or_string_instance.to_dict()
# create an instance of IntstrIntOrString from a dict
intstr_int_or_string_from_dict = IntstrIntOrString.from_dict(intstr_int_or_string_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


