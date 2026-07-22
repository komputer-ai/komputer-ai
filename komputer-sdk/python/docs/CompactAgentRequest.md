# CompactAgentRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**instructions** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.compact_agent_request import CompactAgentRequest

# TODO update the JSON string below
json = "{}"
# create an instance of CompactAgentRequest from a JSON string
compact_agent_request_instance = CompactAgentRequest.from_json(json)
# print the JSON string representation of the object
print(CompactAgentRequest.to_json())

# convert the object into a dict
compact_agent_request_dict = compact_agent_request_instance.to_dict()
# create an instance of CompactAgentRequest from a dict
compact_agent_request_from_dict = CompactAgentRequest.from_dict(compact_agent_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


