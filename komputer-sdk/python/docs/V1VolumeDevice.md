# V1VolumeDevice


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_path** | **str** | devicePath is the path inside of the container that the device will be mapped to. | [optional] 
**name** | **str** | name must match the name of a persistentVolumeClaim in the pod | [optional] 

## Example

```python
from komputer_ai.models.v1_volume_device import V1VolumeDevice

# TODO update the JSON string below
json = "{}"
# create an instance of V1VolumeDevice from a JSON string
v1_volume_device_instance = V1VolumeDevice.from_json(json)
# print the JSON string representation of the object
print(V1VolumeDevice.to_json())

# convert the object into a dict
v1_volume_device_dict = v1_volume_device_instance.to_dict()
# create an instance of V1VolumeDevice from a dict
v1_volume_device_from_dict = V1VolumeDevice.from_dict(v1_volume_device_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


