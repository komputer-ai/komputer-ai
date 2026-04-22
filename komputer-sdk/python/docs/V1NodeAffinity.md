# V1NodeAffinity


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**preferred_during_scheduling_ignored_during_execution** | [**List[V1PreferredSchedulingTerm]**](V1PreferredSchedulingTerm.md) | The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding \&quot;weight\&quot; to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred. +optional +listType&#x3D;atomic | [optional] 
**required_during_scheduling_ignored_during_execution** | [**V1NodeSelector**](V1NodeSelector.md) | If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node. +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_node_affinity import V1NodeAffinity

# TODO update the JSON string below
json = "{}"
# create an instance of V1NodeAffinity from a JSON string
v1_node_affinity_instance = V1NodeAffinity.from_json(json)
# print the JSON string representation of the object
print(V1NodeAffinity.to_json())

# convert the object into a dict
v1_node_affinity_dict = v1_node_affinity_instance.to_dict()
# create an instance of V1NodeAffinity from a dict
v1_node_affinity_from_dict = V1NodeAffinity.from_dict(v1_node_affinity_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


