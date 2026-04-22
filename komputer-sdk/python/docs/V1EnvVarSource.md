# V1EnvVarSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map_key_ref** | [**V1ConfigMapKeySelector**](V1ConfigMapKeySelector.md) | Selects a key of a ConfigMap. +optional | [optional] 
**field_ref** | [**V1ObjectFieldSelector**](V1ObjectFieldSelector.md) | Selects a field of the pod: supports metadata.name, metadata.namespace, &#x60;metadata.labels[&#39;&lt;KEY&gt;&#39;]&#x60;, &#x60;metadata.annotations[&#39;&lt;KEY&gt;&#39;]&#x60;, spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs. +optional | [optional] 
**file_key_ref** | [**V1FileKeySelector**](V1FileKeySelector.md) | FileKeyRef selects a key of the env file. Requires the EnvFiles feature gate to be enabled.  +featureGate&#x3D;EnvFiles +optional | [optional] 
**resource_field_ref** | [**V1ResourceFieldSelector**](V1ResourceFieldSelector.md) | Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported. +optional | [optional] 
**secret_key_ref** | [**V1SecretKeySelector**](V1SecretKeySelector.md) | Selects a key of a secret in the pod&#39;s namespace +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_env_var_source import V1EnvVarSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1EnvVarSource from a JSON string
v1_env_var_source_instance = V1EnvVarSource.from_json(json)
# print the JSON string representation of the object
print(V1EnvVarSource.to_json())

# convert the object into a dict
v1_env_var_source_dict = v1_env_var_source_instance.to_dict()
# create an instance of V1EnvVarSource from a dict
v1_env_var_source_from_dict = V1EnvVarSource.from_dict(v1_env_var_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


