# komputer_ai.ConfigApi

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_config**](ConfigApi.md#get_config) | **GET** /config | Get platform configuration


# **get_config**
> ConfigResponse get_config()

Get platform configuration

Returns platform settings that change what clients can offer, such as whether the S3 session store is enabled. Never returns credentials.

### Example


```python
import komputer_ai
from komputer_ai.models.config_response import ConfigResponse
from komputer_ai.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to http://localhost:8080/api/v1
# See configuration.py for a list of all supported configuration parameters.
configuration = komputer_ai.Configuration(
    host = "http://localhost:8080/api/v1"
)


# Enter a context with an instance of the API client
with komputer_ai.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = komputer_ai.ConfigApi(api_client)

    try:
        # Get platform configuration
        api_response = api_instance.get_config()
        print("The response of ConfigApi->get_config:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ConfigApi->get_config: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**ConfigResponse**](ConfigResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Platform configuration |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

