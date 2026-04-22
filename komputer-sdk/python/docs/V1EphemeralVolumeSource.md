# V1EphemeralVolumeSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**volume_claim_template** | [**V1PersistentVolumeClaimTemplate**](V1PersistentVolumeClaimTemplate.md) | Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC, i.e. the PVC will be deleted together with the pod.  The name of the PVC will be &#x60;&lt;pod name&gt;-&lt;volume name&gt;&#x60; where &#x60;&lt;volume name&gt;&#x60; is the name from the &#x60;PodSpec.Volumes&#x60; array entry. Pod validation will reject the pod if the concatenated name is not valid for a PVC (for example, too long).  An existing PVC with that name that is not owned by the pod will *not* be used for the pod to avoid using an unrelated volume by mistake. Starting the pod is then blocked until the unrelated PVC is removed. If such a pre-created PVC is meant to be used by the pod, the PVC has to updated with an owner reference to the pod once the pod exists. Normally this should not be necessary, but it may be useful when manually reconstructing a broken cluster.  This field is read-only and no changes will be made by Kubernetes to the PVC after it has been created.  Required, must not be nil. | [optional] 

## Example

```python
from komputer_ai.models.v1_ephemeral_volume_source import V1EphemeralVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1EphemeralVolumeSource from a JSON string
v1_ephemeral_volume_source_instance = V1EphemeralVolumeSource.from_json(json)
# print the JSON string representation of the object
print(V1EphemeralVolumeSource.to_json())

# convert the object into a dict
v1_ephemeral_volume_source_dict = v1_ephemeral_volume_source_instance.to_dict()
# create an instance of V1EphemeralVolumeSource from a dict
v1_ephemeral_volume_source_from_dict = V1EphemeralVolumeSource.from_dict(v1_ephemeral_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


