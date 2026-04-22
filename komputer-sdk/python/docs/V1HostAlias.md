# V1HostAlias


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**hostnames** | **List[str]** | Hostnames for the above IP address. +listType&#x3D;atomic | [optional] 
**ip** | **str** | IP address of the host file entry. +required | [optional] 

## Example

```python
from komputer_ai.models.v1_host_alias import V1HostAlias

# TODO update the JSON string below
json = "{}"
# create an instance of V1HostAlias from a JSON string
v1_host_alias_instance = V1HostAlias.from_json(json)
# print the JSON string representation of the object
print(V1HostAlias.to_json())

# convert the object into a dict
v1_host_alias_dict = v1_host_alias_instance.to_dict()
# create an instance of V1HostAlias from a dict
v1_host_alias_from_dict = V1HostAlias.from_dict(v1_host_alias_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


