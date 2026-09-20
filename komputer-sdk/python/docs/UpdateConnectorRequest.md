# UpdateConnectorRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**namespace** | **str** |  | [optional] 
**token** | **str** | new auth token; replaces the value in the connector&#39;s secret | 

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


