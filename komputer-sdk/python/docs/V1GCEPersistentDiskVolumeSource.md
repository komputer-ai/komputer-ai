# V1GCEPersistentDiskVolumeSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | fsType is filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk TODO: how do we prevent errors in the filesystem from compromising the machine +optional | [optional] 
**partition** | **int** | partition is the partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as \&quot;1\&quot;. Similarly, the volume partition for /dev/sda is \&quot;0\&quot; (or you can leave the property empty). More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk +optional | [optional] 
**pd_name** | **str** | pdName is unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk | [optional] 
**read_only** | **bool** | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_gce_persistent_disk_volume_source import V1GCEPersistentDiskVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1GCEPersistentDiskVolumeSource from a JSON string
v1_gce_persistent_disk_volume_source_instance = V1GCEPersistentDiskVolumeSource.from_json(json)
# print the JSON string representation of the object
print(V1GCEPersistentDiskVolumeSource.to_json())

# convert the object into a dict
v1_gce_persistent_disk_volume_source_dict = v1_gce_persistent_disk_volume_source_instance.to_dict()
# create an instance of V1GCEPersistentDiskVolumeSource from a dict
v1_gce_persistent_disk_volume_source_from_dict = V1GCEPersistentDiskVolumeSource.from_dict(v1_gce_persistent_disk_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


