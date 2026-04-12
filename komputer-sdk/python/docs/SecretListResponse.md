# SecretListResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**secrets** | [**List[SecretResponse]**](SecretResponse.md) |  | [optional] 

## Example

```python
from komputer_ai.models.secret_list_response import SecretListResponse

# TODO update the JSON string below
json = "{}"
# create an instance of SecretListResponse from a JSON string
secret_list_response_instance = SecretListResponse.from_json(json)
# print the JSON string representation of the object
print(SecretListResponse.to_json())

# convert the object into a dict
secret_list_response_dict = secret_list_response_instance.to_dict()
# create an instance of SecretListResponse from a dict
secret_list_response_from_dict = SecretListResponse.from_dict(secret_list_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


