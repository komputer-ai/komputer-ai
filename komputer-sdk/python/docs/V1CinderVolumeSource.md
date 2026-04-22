# V1CinderVolumeSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | fsType is the filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://examples.k8s.io/mysql-cinder-pd/README.md +optional | [optional] 
**read_only** | **bool** | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/mysql-cinder-pd/README.md +optional | [optional] 
**secret_ref** | [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef is optional: points to a secret object containing parameters used to connect to OpenStack. +optional | [optional] 
**volume_id** | **str** | volumeID used to identify the volume in cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md | [optional] 

## Example

```python
from komputer_ai.models.v1_cinder_volume_source import V1CinderVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1CinderVolumeSource from a JSON string
v1_cinder_volume_source_instance = V1CinderVolumeSource.from_json(json)
# print the JSON string representation of the object
print(V1CinderVolumeSource.to_json())

# convert the object into a dict
v1_cinder_volume_source_dict = v1_cinder_volume_source_instance.to_dict()
# create an instance of V1CinderVolumeSource from a dict
v1_cinder_volume_source_from_dict = V1CinderVolumeSource.from_dict(v1_cinder_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


