# komputer_ai.OfficesApi

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**delete_office**](OfficesApi.md#delete_office) | **DELETE** /offices/{name} | Delete office
[**get_office**](OfficesApi.md#get_office) | **GET** /offices/{name} | Get office details
[**get_office_events**](OfficesApi.md#get_office_events) | **GET** /offices/{name}/events | Get office events
[**list_offices**](OfficesApi.md#list_offices) | **GET** /offices | List offices


# **delete_office**
> Dict[str, str] delete_office(name, namespace=namespace)

Delete office

Deletes the office CR and cleans up Redis event streams for all member agents.

### Example


```python
import komputer_ai
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
    api_instance = komputer_ai.OfficesApi(api_client)
    name = 'name_example' # str | Office name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Delete office
        api_response = api_instance.delete_office(name, namespace=namespace)
        print("The response of OfficesApi->delete_office:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling OfficesApi->delete_office: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Office name | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

**Dict[str, str]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Office deleted |  -  |
**404** | Office not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_office**
> OfficeResponse get_office(name, namespace=namespace)

Get office details

Returns the current status and member list for a single office.

### Example


```python
import komputer_ai
from komputer_ai.models.office_response import OfficeResponse
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
    api_instance = komputer_ai.OfficesApi(api_client)
    name = 'name_example' # str | Office name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Get office details
        api_response = api_instance.get_office(name, namespace=namespace)
        print("The response of OfficesApi->get_office:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling OfficesApi->get_office: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Office name | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**OfficeResponse**](OfficeResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Office details |  -  |
**404** | Office not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_office_events**
> Dict[str, object] get_office_events(name, namespace=namespace, limit=limit)

Get office events

Returns merged events from all member agent Redis streams, sorted chronologically.

### Example


```python
import komputer_ai
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
    api_instance = komputer_ai.OfficesApi(api_client)
    name = 'name_example' # str | Office name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)
    limit = 50 # int | Max events to return (1-200) (optional) (default to 50)

    try:
        # Get office events
        api_response = api_instance.get_office_events(name, namespace=namespace, limit=limit)
        print("The response of OfficesApi->get_office_events:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling OfficesApi->get_office_events: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Office name | 
 **namespace** | **str**| Kubernetes namespace | [optional] 
 **limit** | **int**| Max events to return (1-200) | [optional] [default to 50]

### Return type

**Dict[str, object]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Office events |  -  |
**400** | Invalid limit parameter |  -  |
**404** | Office not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_offices**
> OfficeListResponse list_offices(namespace=namespace)

List offices

Returns all offices with their current status in the specified namespace.

### Example


```python
import komputer_ai
from komputer_ai.models.office_list_response import OfficeListResponse
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
    api_instance = komputer_ai.OfficesApi(api_client)
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # List offices
        api_response = api_instance.list_offices(namespace=namespace)
        print("The response of OfficesApi->list_offices:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling OfficesApi->list_offices: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**OfficeListResponse**](OfficeListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | List of offices |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

