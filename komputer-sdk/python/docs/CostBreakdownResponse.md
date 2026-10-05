# CostBreakdownResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent** | **str** |  | [optional] 
**cached_at** | **str** |  | [optional] 
**task_count** | **int** |  | [optional] 
**tasks** | [**List[TaskBreakdown]**](TaskBreakdown.md) |  | [optional] 
**total_cost** | **float** |  | [optional] 

## Example

```python
from komputer_ai.models.cost_breakdown_response import CostBreakdownResponse

# TODO update the JSON string below
json = "{}"
# create an instance of CostBreakdownResponse from a JSON string
cost_breakdown_response_instance = CostBreakdownResponse.from_json(json)
# print the JSON string representation of the object
print(CostBreakdownResponse.to_json())

# convert the object into a dict
cost_breakdown_response_dict = cost_breakdown_response_instance.to_dict()
# create an instance of CostBreakdownResponse from a dict
cost_breakdown_response_from_dict = CostBreakdownResponse.from_dict(cost_breakdown_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


