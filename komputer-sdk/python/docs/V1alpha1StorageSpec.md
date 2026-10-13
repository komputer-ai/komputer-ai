# V1alpha1StorageSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ephemeral** | **bool** | Ephemeral, when true, runs the agent&#39;s workspace as an emptyDir instead of a PVC: no PersistentVolumeClaim is created. The workspace lives exactly as long as the pod — files, installed packages and the Claude session are permanently lost whenever the pod is deleted, evicted, rescheduled or restarted. Incompatible with Lifecycle&#x3D;Sleep and with sleepTTL, since both would sleep the agent by deleting its pod (the operator refuses to sleep an ephemeral agent for this reason).  Unset (default) or false keeps today&#39;s behavior: every agent gets a PVC. Set it on a KomputerAgentTemplate to make ephemeral the default for agents using it, or on an individual agent to override its template either way. Immutable after agent creation. An agent that already has a PVC keeps it even if later marked ephemeral — the operator never deletes an existing PVC. +optional | [optional] 
**size** | **str** | Size is the PVC storage size (e.g. \&quot;5Gi\&quot;). When Ephemeral is true, this is instead used as the emptyDir&#39;s SizeLimit. +kubebuilder:default&#x3D;\&quot;5Gi\&quot; | [optional] 
**storage_class_name** | **str** | StorageClassName is the optional storage class name. Ignored when Ephemeral is true. +optional | [optional] 

## Example

```python
from komputer_ai.models.v1alpha1_storage_spec import V1alpha1StorageSpec

# TODO update the JSON string below
json = "{}"
# create an instance of V1alpha1StorageSpec from a JSON string
v1alpha1_storage_spec_instance = V1alpha1StorageSpec.from_json(json)
# print the JSON string representation of the object
print(V1alpha1StorageSpec.to_json())

# convert the object into a dict
v1alpha1_storage_spec_dict = v1alpha1_storage_spec_instance.to_dict()
# create an instance of V1alpha1StorageSpec from a dict
v1alpha1_storage_spec_from_dict = V1alpha1StorageSpec.from_dict(v1alpha1_storage_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


