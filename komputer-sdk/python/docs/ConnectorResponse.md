# ConnectorResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**agent_names** | **List[str]** |  | [optional] 
**attached_agents** | **int** |  | [optional] 
**auth_secret_key** | **str** |  | [optional] 
**auth_secret_name** | **str** |  | [optional] 
**auth_type** | **str** |  | [optional] 
**created_at** | **str** |  | [optional] 
**display_name** | **str** |  | [optional] 
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**oauth_status** | **str** | \&quot;pending\&quot;, \&quot;connected\&quot;, \&quot;\&quot; | [optional] 
**service** | **str** |  | [optional] 
**type** | **str** |  | [optional] 
**url** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.connector_response import ConnectorResponse

# TODO update the JSON string below
json = "{}"
# create an instance of ConnectorResponse from a JSON string
connector_response_instance = ConnectorResponse.from_json(json)
# print the JSON string representation of the object
print(ConnectorResponse.to_json())

# convert the object into a dict
connector_response_dict = connector_response_instance.to_dict()
# create an instance of ConnectorResponse from a dict
connector_response_from_dict = ConnectorResponse.from_dict(connector_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


