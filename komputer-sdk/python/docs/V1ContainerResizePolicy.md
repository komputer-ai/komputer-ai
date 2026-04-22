# V1ContainerResizePolicy


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**resource_name** | [**V1ResourceName**](V1ResourceName.md) | Name of the resource to which this resource resize policy applies. Supported values: cpu, memory. | [optional] 
**restart_policy** | [**V1ResourceResizeRestartPolicy**](V1ResourceResizeRestartPolicy.md) | Restart policy to apply when specified resource is resized. If not specified, it defaults to NotRequired. | [optional] 

## Example

```python
from komputer_ai.models.v1_container_resize_policy import V1ContainerResizePolicy

# TODO update the JSON string below
json = "{}"
# create an instance of V1ContainerResizePolicy from a JSON string
v1_container_resize_policy_instance = V1ContainerResizePolicy.from_json(json)
# print the JSON string representation of the object
print(V1ContainerResizePolicy.to_json())

# convert the object into a dict
v1_container_resize_policy_dict = v1_container_resize_policy_instance.to_dict()
# create an instance of V1ContainerResizePolicy from a dict
v1_container_resize_policy_from_dict = V1ContainerResizePolicy.from_dict(v1_container_resize_policy_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


