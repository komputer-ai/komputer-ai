# V1PreferredSchedulingTerm


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**preference** | [**V1NodeSelectorTerm**](V1NodeSelectorTerm.md) | A node selector term, associated with the corresponding weight. | [optional] 
**weight** | **int** | Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100. | [optional] 

## Example

```python
from komputer_ai.models.v1_preferred_scheduling_term import V1PreferredSchedulingTerm

# TODO update the JSON string below
json = "{}"
# create an instance of V1PreferredSchedulingTerm from a JSON string
v1_preferred_scheduling_term_instance = V1PreferredSchedulingTerm.from_json(json)
# print the JSON string representation of the object
print(V1PreferredSchedulingTerm.to_json())

# convert the object into a dict
v1_preferred_scheduling_term_dict = v1_preferred_scheduling_term_instance.to_dict()
# create an instance of V1PreferredSchedulingTerm from a dict
v1_preferred_scheduling_term_from_dict = V1PreferredSchedulingTerm.from_dict(v1_preferred_scheduling_term_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


