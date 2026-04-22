# V1RBDVolumeSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | fsType is the filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#rbd TODO: how do we prevent errors in the filesystem from compromising the machine +optional | [optional] 
**image** | **str** | image is the rados image name. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it | [optional] 
**keyring** | **str** | keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default&#x3D;\&quot;/etc/ceph/keyring\&quot; | [optional] 
**monitors** | **List[str]** | monitors is a collection of Ceph monitors. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +listType&#x3D;atomic | [optional] 
**pool** | **str** | pool is the rados pool name. Default is rbd. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default&#x3D;\&quot;rbd\&quot; | [optional] 
**read_only** | **bool** | readOnly here will force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional | [optional] 
**secret_ref** | [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef is name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional | [optional] 
**user** | **str** | user is the rados user name. Default is admin. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it +optional +default&#x3D;\&quot;admin\&quot; | [optional] 

## Example

```python
from komputer_ai.models.v1_rbd_volume_source import V1RBDVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1RBDVolumeSource from a JSON string
v1_rbd_volume_source_instance = V1RBDVolumeSource.from_json(json)
# print the JSON string representation of the object
print(V1RBDVolumeSource.to_json())

# convert the object into a dict
v1_rbd_volume_source_dict = v1_rbd_volume_source_instance.to_dict()
# create an instance of V1RBDVolumeSource from a dict
v1_rbd_volume_source_from_dict = V1RBDVolumeSource.from_dict(v1_rbd_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


