# V1PersistentVolumeClaimTemplate


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**metadata** | [**V1ObjectMeta**](V1ObjectMeta.md) | May contain labels and annotations that will be copied into the PVC when creating it. No other fields are allowed and will be rejected during validation.  +optional | [optional] 
**spec** | [**V1PersistentVolumeClaimSpec**](V1PersistentVolumeClaimSpec.md) | The specification for the PersistentVolumeClaim. The entire content is copied unchanged into the PVC that gets created from this template. The same fields as in a PersistentVolumeClaim are also valid here. | [optional] 

## Example

```python
from komputer_ai.models.v1_persistent_volume_claim_template import V1PersistentVolumeClaimTemplate

# TODO update the JSON string below
json = "{}"
# create an instance of V1PersistentVolumeClaimTemplate from a JSON string
v1_persistent_volume_claim_template_instance = V1PersistentVolumeClaimTemplate.from_json(json)
# print the JSON string representation of the object
print(V1PersistentVolumeClaimTemplate.to_json())

# convert the object into a dict
v1_persistent_volume_claim_template_dict = v1_persistent_volume_claim_template_instance.to_dict()
# create an instance of V1PersistentVolumeClaimTemplate from a dict
v1_persistent_volume_claim_template_from_dict = V1PersistentVolumeClaimTemplate.from_dict(v1_persistent_volume_claim_template_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


