# V1Affinity


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**node_affinity** | [**V1NodeAffinity**](V1NodeAffinity.md) | Describes node affinity scheduling rules for the pod. +optional | [optional] 
**pod_affinity** | [**V1PodAffinity**](V1PodAffinity.md) | Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)). +optional | [optional] 
**pod_anti_affinity** | [**V1PodAntiAffinity**](V1PodAntiAffinity.md) | Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)). +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_affinity import V1Affinity

# TODO update the JSON string below
json = "{}"
# create an instance of V1Affinity from a JSON string
v1_affinity_instance = V1Affinity.from_json(json)
# print the JSON string representation of the object
print(V1Affinity.to_json())

# convert the object into a dict
v1_affinity_dict = v1_affinity_instance.to_dict()
# create an instance of V1Affinity from a dict
v1_affinity_from_dict = V1Affinity.from_dict(v1_affinity_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


