# V1CSIVolumeSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**driver** | **str** | driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster. | [optional] 
**fs_type** | **str** | fsType to mount. Ex. \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply. +optional | [optional] 
**node_publish_secret_ref** | [**V1LocalObjectReference**](V1LocalObjectReference.md) | nodePublishSecretRef is a reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls. This field is optional, and  may be empty if no secret is required. If the secret object contains more than one secret, all secret references are passed. +optional | [optional] 
**read_only** | **bool** | readOnly specifies a read-only configuration for the volume. Defaults to false (read/write). +optional | [optional] 
**volume_attributes** | **Dict[str, str]** | volumeAttributes stores driver-specific properties that are passed to the CSI driver. Consult your driver&#39;s documentation for supported values. +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_csi_volume_source import V1CSIVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1CSIVolumeSource from a JSON string
v1_csi_volume_source_instance = V1CSIVolumeSource.from_json(json)
# print the JSON string representation of the object
print(V1CSIVolumeSource.to_json())

# convert the object into a dict
v1_csi_volume_source_dict = v1_csi_volume_source_instance.to_dict()
# create an instance of V1CSIVolumeSource from a dict
v1_csi_volume_source_from_dict = V1CSIVolumeSource.from_dict(v1_csi_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


