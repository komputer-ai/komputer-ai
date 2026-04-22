# V1alpha1StorageSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**size** | **str** | Size is the PVC storage size (e.g. \&quot;5Gi\&quot;). +kubebuilder:default&#x3D;\&quot;5Gi\&quot; | [optional] 
**storage_class_name** | **str** | StorageClassName is the optional storage class name. +optional | [optional] 

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


