# V1PersistentVolumeClaimVolumeSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**claim_name** | **str** | claimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims | [optional] 
**read_only** | **bool** | readOnly Will force the ReadOnly setting in VolumeMounts. Default false. +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_persistent_volume_claim_volume_source import V1PersistentVolumeClaimVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1PersistentVolumeClaimVolumeSource from a JSON string
v1_persistent_volume_claim_volume_source_instance = V1PersistentVolumeClaimVolumeSource.from_json(json)
# print the JSON string representation of the object
print(V1PersistentVolumeClaimVolumeSource.to_json())

# convert the object into a dict
v1_persistent_volume_claim_volume_source_dict = v1_persistent_volume_claim_volume_source_instance.to_dict()
# create an instance of V1PersistentVolumeClaimVolumeSource from a dict
v1_persistent_volume_claim_volume_source_from_dict = V1PersistentVolumeClaimVolumeSource.from_dict(v1_persistent_volume_claim_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


