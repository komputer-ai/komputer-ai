# V1SELinuxOptions


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**level** | **str** | Level is SELinux level label that applies to the container. +optional | [optional] 
**role** | **str** | Role is a SELinux role label that applies to the container. +optional | [optional] 
**type** | **str** | Type is a SELinux type label that applies to the container. +optional | [optional] 
**user** | **str** | User is a SELinux user label that applies to the container. +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_se_linux_options import V1SELinuxOptions

# TODO update the JSON string below
json = "{}"
# create an instance of V1SELinuxOptions from a JSON string
v1_se_linux_options_instance = V1SELinuxOptions.from_json(json)
# print the JSON string representation of the object
print(V1SELinuxOptions.to_json())

# convert the object into a dict
v1_se_linux_options_dict = v1_se_linux_options_instance.to_dict()
# create an instance of V1SELinuxOptions from a dict
v1_se_linux_options_from_dict = V1SELinuxOptions.from_dict(v1_se_linux_options_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


