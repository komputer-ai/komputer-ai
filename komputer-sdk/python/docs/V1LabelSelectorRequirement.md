# V1LabelSelectorRequirement


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** | key is the label key that the selector applies to. | [optional] 
**operator** | [**V1LabelSelectorOperator**](V1LabelSelectorOperator.md) | operator represents a key&#39;s relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist. | [optional] 
**values** | **List[str]** | values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch. +optional +listType&#x3D;atomic | [optional] 

## Example

```python
from komputer_ai.models.v1_label_selector_requirement import V1LabelSelectorRequirement

# TODO update the JSON string below
json = "{}"
# create an instance of V1LabelSelectorRequirement from a JSON string
v1_label_selector_requirement_instance = V1LabelSelectorRequirement.from_json(json)
# print the JSON string representation of the object
print(V1LabelSelectorRequirement.to_json())

# convert the object into a dict
v1_label_selector_requirement_dict = v1_label_selector_requirement_instance.to_dict()
# create an instance of V1LabelSelectorRequirement from a dict
v1_label_selector_requirement_from_dict = V1LabelSelectorRequirement.from_dict(v1_label_selector_requirement_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


