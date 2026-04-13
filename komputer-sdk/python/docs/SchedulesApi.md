# komputer_ai.SchedulesApi

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_schedule**](SchedulesApi.md#create_schedule) | **POST** /schedules | Create schedule
[**delete_schedule**](SchedulesApi.md#delete_schedule) | **DELETE** /schedules/{name} | Delete schedule
[**get_schedule**](SchedulesApi.md#get_schedule) | **GET** /schedules/{name} | Get schedule details
[**list_schedules**](SchedulesApi.md#list_schedules) | **GET** /schedules | List schedules
[**patch_schedule**](SchedulesApi.md#patch_schedule) | **PATCH** /schedules/{name} | Patch schedule


# **create_schedule**
> ScheduleResponse create_schedule(request)

Create schedule

Creates a new KomputerSchedule CR that triggers agent tasks on a cron schedule.

### Example


```python
import komputer_ai
from komputer_ai.models.create_schedule_request import CreateScheduleRequest
from komputer_ai.models.schedule_response import ScheduleResponse
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
    api_instance = komputer_ai.SchedulesApi(api_client)
    request = komputer_ai.CreateScheduleRequest() # CreateScheduleRequest | Schedule creation request

    try:
        # Create schedule
        api_response = api_instance.create_schedule(request)
        print("The response of SchedulesApi->create_schedule:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SchedulesApi->create_schedule: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateScheduleRequest**](CreateScheduleRequest.md)| Schedule creation request | 

### Return type

[**ScheduleResponse**](ScheduleResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | Schedule created |  -  |
**400** | Bad request |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_schedule**
> Dict[str, str] delete_schedule(name, namespace=namespace)

Delete schedule

Deletes the schedule CR. Does not delete any agents that were created by the schedule.

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
    api_instance = komputer_ai.SchedulesApi(api_client)
    name = 'name_example' # str | Schedule name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Delete schedule
        api_response = api_instance.delete_schedule(name, namespace=namespace)
        print("The response of SchedulesApi->delete_schedule:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SchedulesApi->delete_schedule: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Schedule name | 
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
**200** | Schedule deleted |  -  |
**404** | Schedule not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_schedule**
> ScheduleResponse get_schedule(name, namespace=namespace)

Get schedule details

Returns the current status and run history for a single schedule.

### Example


```python
import komputer_ai
from komputer_ai.models.schedule_response import ScheduleResponse
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
    api_instance = komputer_ai.SchedulesApi(api_client)
    name = 'name_example' # str | Schedule name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Get schedule details
        api_response = api_instance.get_schedule(name, namespace=namespace)
        print("The response of SchedulesApi->get_schedule:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SchedulesApi->get_schedule: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Schedule name | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**ScheduleResponse**](ScheduleResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Schedule details |  -  |
**404** | Schedule not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_schedules**
> ScheduleListResponse list_schedules(namespace=namespace)

List schedules

Returns all schedules with their current status and run history in the specified namespace.

### Example


```python
import komputer_ai
from komputer_ai.models.schedule_list_response import ScheduleListResponse
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
    api_instance = komputer_ai.SchedulesApi(api_client)
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # List schedules
        api_response = api_instance.list_schedules(namespace=namespace)
        print("The response of SchedulesApi->list_schedules:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SchedulesApi->list_schedules: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**ScheduleListResponse**](ScheduleListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | List of schedules |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **patch_schedule**
> ScheduleResponse patch_schedule(name, request, namespace=namespace)

Patch schedule

Updates the cron expression for an existing schedule.

### Example


```python
import komputer_ai
from komputer_ai.models.patch_schedule_request import PatchScheduleRequest
from komputer_ai.models.schedule_response import ScheduleResponse
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
    api_instance = komputer_ai.SchedulesApi(api_client)
    name = 'name_example' # str | Schedule name
    request = komputer_ai.PatchScheduleRequest() # PatchScheduleRequest | Fields to update
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Patch schedule
        api_response = api_instance.patch_schedule(name, request, namespace=namespace)
        print("The response of SchedulesApi->patch_schedule:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SchedulesApi->patch_schedule: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Schedule name | 
 **request** | [**PatchScheduleRequest**](PatchScheduleRequest.md)| Fields to update | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**ScheduleResponse**](ScheduleResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Updated schedule |  -  |
**400** | Bad request |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

