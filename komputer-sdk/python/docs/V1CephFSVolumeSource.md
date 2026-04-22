# V1CephFSVolumeSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**monitors** | **List[str]** | monitors is Required: Monitors is a collection of Ceph monitors More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +listType&#x3D;atomic | [optional] 
**path** | **str** | path is Optional: Used as the mounted root, rather than the full Ceph tree, default is / +optional | [optional] 
**read_only** | **bool** | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional | [optional] 
**secret_file** | **str** | secretFile is Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional | [optional] 
**secret_ref** | [**V1LocalObjectReference**](V1LocalObjectReference.md) | secretRef is Optional: SecretRef is reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional | [optional] 
**user** | **str** | user is optional: User is the rados user name, default is admin More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_ceph_fs_volume_source import V1CephFSVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1CephFSVolumeSource from a JSON string
v1_ceph_fs_volume_source_instance = V1CephFSVolumeSource.from_json(json)
# print the JSON string representation of the object
print(V1CephFSVolumeSource.to_json())

# convert the object into a dict
v1_ceph_fs_volume_source_dict = v1_ceph_fs_volume_source_instance.to_dict()
# create an instance of V1CephFSVolumeSource from a dict
v1_ceph_fs_volume_source_from_dict = V1CephFSVolumeSource.from_dict(v1_ceph_fs_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


