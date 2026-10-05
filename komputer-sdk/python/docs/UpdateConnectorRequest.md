# UpdateConnectorRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**disabled** | **bool** | Disabled toggles whether the connector can be newly attached to agents and whether its tools are resolved for agent pods. true disables, false re-enables. Any caller may toggle this — there is no additional authorization check. | [optional] 
**namespace** | **str** |  | [optional] 
**token** | **str** | new auth token; replaces the value in the connector&#39;s secret | [optional] 

## Example

```python
from komputer_ai.models.update_connector_request import UpdateConnectorRequest

# TODO update the JSON string below
json = "{}"
# create an instance of UpdateConnectorRequest from a JSON string
update_connector_request_instance = UpdateConnectorRequest.from_json(json)
# print the JSON string representation of the object
print(UpdateConnectorRequest.to_json())

# convert the object into a dict
update_connector_request_dict = update_connector_request_instance.to_dict()
# create an instance of UpdateConnectorRequest from a dict
update_connector_request_from_dict = UpdateConnectorRequest.from_dict(update_connector_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


