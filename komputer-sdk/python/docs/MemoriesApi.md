# komputer_ai.MemoriesApi

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_memory**](MemoriesApi.md#create_memory) | **POST** /memories | Create memory
[**delete_memory**](MemoriesApi.md#delete_memory) | **DELETE** /memories/{name} | Delete memory
[**get_memory**](MemoriesApi.md#get_memory) | **GET** /memories/{name} | Get memory details
[**list_memories**](MemoriesApi.md#list_memories) | **GET** /memories | List memories
[**patch_memory**](MemoriesApi.md#patch_memory) | **PATCH** /memories/{name} | Patch memory


# **create_memory**
> MemoryResponse create_memory(request)

Create memory

Creates a new KomputerMemory CR that can be attached to agents as persistent context.

### Example


```python
import komputer_ai
from komputer_ai.models.create_memory_request import CreateMemoryRequest
from komputer_ai.models.memory_response import MemoryResponse
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
    api_instance = komputer_ai.MemoriesApi(api_client)
    request = komputer_ai.CreateMemoryRequest() # CreateMemoryRequest | Memory creation request

    try:
        # Create memory
        api_response = api_instance.create_memory(request)
        print("The response of MemoriesApi->create_memory:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MemoriesApi->create_memory: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateMemoryRequest**](CreateMemoryRequest.md)| Memory creation request | 

### Return type

[**MemoryResponse**](MemoryResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | Memory created |  -  |
**400** | Bad request |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_memory**
> Dict[str, str] delete_memory(name, namespace=namespace)

Delete memory

Deletes the memory CR.

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
    api_instance = komputer_ai.MemoriesApi(api_client)
    name = 'name_example' # str | Memory name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Delete memory
        api_response = api_instance.delete_memory(name, namespace=namespace)
        print("The response of MemoriesApi->delete_memory:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MemoriesApi->delete_memory: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Memory name | 
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
**200** | Memory deleted |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_memory**
> MemoryResponse get_memory(name, namespace=namespace)

Get memory details

Returns the content and attached agent count for a single memory.

### Example


```python
import komputer_ai
from komputer_ai.models.memory_response import MemoryResponse
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
    api_instance = komputer_ai.MemoriesApi(api_client)
    name = 'name_example' # str | Memory name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Get memory details
        api_response = api_instance.get_memory(name, namespace=namespace)
        print("The response of MemoriesApi->get_memory:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MemoriesApi->get_memory: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Memory name | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**MemoryResponse**](MemoryResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Memory details |  -  |
**404** | Memory not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_memories**
> Dict[str, object] list_memories(namespace=namespace)

List memories

Returns all memories with content and attached agent counts in the specified namespace.

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
    api_instance = komputer_ai.MemoriesApi(api_client)
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # List memories
        api_response = api_instance.list_memories(namespace=namespace)
        print("The response of MemoriesApi->list_memories:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MemoriesApi->list_memories: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| Kubernetes namespace | [optional] 

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
**200** | List of memories |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **patch_memory**
> MemoryResponse patch_memory(name, request, namespace=namespace)

Patch memory

Updates the content or description of an existing memory.

### Example


```python
import komputer_ai
from komputer_ai.models.memory_response import MemoryResponse
from komputer_ai.models.patch_memory_request import PatchMemoryRequest
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
    api_instance = komputer_ai.MemoriesApi(api_client)
    name = 'name_example' # str | Memory name
    request = komputer_ai.PatchMemoryRequest() # PatchMemoryRequest | Fields to update
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Patch memory
        api_response = api_instance.patch_memory(name, request, namespace=namespace)
        print("The response of MemoriesApi->patch_memory:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MemoriesApi->patch_memory: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Memory name | 
 **request** | [**PatchMemoryRequest**](PatchMemoryRequest.md)| Fields to update | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**MemoryResponse**](MemoryResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Updated memory |  -  |
**400** | Bad request |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

