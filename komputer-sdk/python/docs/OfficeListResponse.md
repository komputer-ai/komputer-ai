# OfficeListResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**offices** | [**List[OfficeResponse]**](OfficeResponse.md) |  | [optional] 

## Example

```python
from komputer_ai.models.office_list_response import OfficeListResponse

# TODO update the JSON string below
json = "{}"
# create an instance of OfficeListResponse from a JSON string
office_list_response_instance = OfficeListResponse.from_json(json)
# print the JSON string representation of the object
print(OfficeListResponse.to_json())

# convert the object into a dict
office_list_response_dict = office_list_response_instance.to_dict()
# create an instance of OfficeListResponse from a dict
office_list_response_from_dict = OfficeListResponse.from_dict(office_list_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


