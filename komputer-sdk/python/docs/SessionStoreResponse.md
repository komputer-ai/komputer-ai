# SessionStoreResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**error** | **str** |  | [optional] 
**last_saved_at** | **str** |  | [optional] 
**ok** | **bool** |  | [optional] 

## Example

```python
from komputer_ai.models.session_store_response import SessionStoreResponse

# TODO update the JSON string below
json = "{}"
# create an instance of SessionStoreResponse from a JSON string
session_store_response_instance = SessionStoreResponse.from_json(json)
# print the JSON string representation of the object
print(SessionStoreResponse.to_json())

# convert the object into a dict
session_store_response_dict = session_store_response_instance.to_dict()
# create an instance of SessionStoreResponse from a dict
session_store_response_from_dict = SessionStoreResponse.from_dict(session_store_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


