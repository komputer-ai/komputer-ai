# V1SecretEnvSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name of the referent. This field is effectively required, but due to backwards compatibility is allowed to be empty. Instances of this type with an empty value here are almost certainly wrong. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names +optional +default&#x3D;\&quot;\&quot; +kubebuilder:default&#x3D;\&quot;\&quot; TODO: Drop &#x60;kubebuilder:default&#x60; when controller-gen doesn&#39;t need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. | [optional] 
**optional** | **bool** | Specify whether the Secret must be defined +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_secret_env_source import V1SecretEnvSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1SecretEnvSource from a JSON string
v1_secret_env_source_instance = V1SecretEnvSource.from_json(json)
# print the JSON string representation of the object
print(V1SecretEnvSource.to_json())

# convert the object into a dict
v1_secret_env_source_dict = v1_secret_env_source_instance.to_dict()
# create an instance of V1SecretEnvSource from a dict
v1_secret_env_source_from_dict = V1SecretEnvSource.from_dict(v1_secret_env_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


