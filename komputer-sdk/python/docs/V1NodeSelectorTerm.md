# V1NodeSelectorTerm


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**match_expressions** | [**List[V1NodeSelectorRequirement]**](V1NodeSelectorRequirement.md) | A list of node selector requirements by node&#39;s labels. +optional +listType&#x3D;atomic | [optional] 
**match_fields** | [**List[V1NodeSelectorRequirement]**](V1NodeSelectorRequirement.md) | A list of node selector requirements by node&#39;s fields. +optional +listType&#x3D;atomic | [optional] 

## Example

```python
from komputer_ai.models.v1_node_selector_term import V1NodeSelectorTerm

# TODO update the JSON string below
json = "{}"
# create an instance of V1NodeSelectorTerm from a JSON string
v1_node_selector_term_instance = V1NodeSelectorTerm.from_json(json)
# print the JSON string representation of the object
print(V1NodeSelectorTerm.to_json())

# convert the object into a dict
v1_node_selector_term_dict = v1_node_selector_term_instance.to_dict()
# create an instance of V1NodeSelectorTerm from a dict
v1_node_selector_term_from_dict = V1NodeSelectorTerm.from_dict(v1_node_selector_term_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


