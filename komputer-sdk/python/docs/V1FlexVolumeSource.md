# V1FlexVolumeSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**driver** | **str** | driver is the name of the driver to use for this volume. | [optional] 
**fs_type** | **str** | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. The default filesystem depends on FlexVolume script. +optional | [optional] 
**options** | **Dict[str, str]** | options is Optional: this field holds extra command options if any. +optional | [optional] 
**read_only** | **bool** | readOnly is Optional: defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. +optional | [optional] 
**secret_ref** | [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef is Optional: secretRef is reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts. +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_flex_volume_source import V1FlexVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1FlexVolumeSource from a JSON string
v1_flex_volume_source_instance = V1FlexVolumeSource.from_json(json)
# print the JSON string representation of the object
print(V1FlexVolumeSource.to_json())

# convert the object into a dict
v1_flex_volume_source_dict = v1_flex_volume_source_instance.to_dict()
# create an instance of V1FlexVolumeSource from a dict
v1_flex_volume_source_from_dict = V1FlexVolumeSource.from_dict(v1_flex_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


