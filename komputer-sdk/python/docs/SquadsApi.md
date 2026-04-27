# komputer_ai.SquadsApi

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**add_squad_member**](SquadsApi.md#add_squad_member) | **POST** /squads/{name}/members | Add squad member
[**break_up_squad**](SquadsApi.md#break_up_squad) | **POST** /squads/{name}/break-up | Request squad break-up
[**delete_squad**](SquadsApi.md#delete_squad) | **DELETE** /squads/{name} | Delete squad
[**get_squad**](SquadsApi.md#get_squad) | **GET** /squads/{name} | Get squad details
[**list_squads**](SquadsApi.md#list_squads) | **GET** /squads | List squads
[**patch_squad**](SquadsApi.md#patch_squad) | **PATCH** /squads/{name} | Patch squad
[**remove_squad_member**](SquadsApi.md#remove_squad_member) | **DELETE** /squads/{name}/members/{agent} | Remove squad member
[**squads_post**](SquadsApi.md#squads_post) | **POST** /squads | 


# **add_squad_member**
> SquadResponse add_squad_member(name, request, namespace=namespace)

Add squad member

Appends a member (by ref or inline spec) to the squad's member list.

### Example


```python
import komputer_ai
from komputer_ai.models.add_squad_member_request import AddSquadMemberRequest
from komputer_ai.models.squad_response import SquadResponse
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
    api_instance = komputer_ai.SquadsApi(api_client)
    name = 'name_example' # str | Squad name
    request = komputer_ai.AddSquadMemberRequest() # AddSquadMemberRequest | Member to add
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Add squad member
        api_response = api_instance.add_squad_member(name, request, namespace=namespace)
        print("The response of SquadsApi->add_squad_member:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SquadsApi->add_squad_member: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Squad name | 
 **request** | [**AddSquadMemberRequest**](AddSquadMemberRequest.md)| Member to add | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Updated squad |  -  |
**400** | Bad request |  -  |
**404** | Squad not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **break_up_squad**
> SquadResponse break_up_squad(name, namespace=namespace)

Request squad break-up

Marks the squad for dissolution once all members are asleep.

### Example


```python
import komputer_ai
from komputer_ai.models.squad_response import SquadResponse
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
    api_instance = komputer_ai.SquadsApi(api_client)
    name = 'name_example' # str | Squad name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Request squad break-up
        api_response = api_instance.break_up_squad(name, namespace=namespace)
        print("The response of SquadsApi->break_up_squad:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SquadsApi->break_up_squad: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Squad name | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Squad with break-up flag set |  -  |
**404** | Squad not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_squad**
> Dict[str, str] delete_squad(name, namespace=namespace)

Delete squad

Deletes the squad CR. The operator will clean up the shared pod.

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
    api_instance = komputer_ai.SquadsApi(api_client)
    name = 'name_example' # str | Squad name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Delete squad
        api_response = api_instance.delete_squad(name, namespace=namespace)
        print("The response of SquadsApi->delete_squad:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SquadsApi->delete_squad: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Squad name | 
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
**200** | Squad deleted |  -  |
**404** | Squad not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_squad**
> SquadResponse get_squad(name, namespace=namespace)

Get squad details

Returns the current status and member list for a single squad.

### Example


```python
import komputer_ai
from komputer_ai.models.squad_response import SquadResponse
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
    api_instance = komputer_ai.SquadsApi(api_client)
    name = 'name_example' # str | Squad name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Get squad details
        api_response = api_instance.get_squad(name, namespace=namespace)
        print("The response of SquadsApi->get_squad:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SquadsApi->get_squad: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Squad name | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Squad details |  -  |
**404** | Squad not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_squads**
> SquadListResponse list_squads(namespace=namespace)

List squads

Returns all squads with their current status. Pass ?namespace= to filter; omit for all namespaces.

### Example


```python
import komputer_ai
from komputer_ai.models.squad_list_response import SquadListResponse
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
    api_instance = komputer_ai.SquadsApi(api_client)
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # List squads
        api_response = api_instance.list_squads(namespace=namespace)
        print("The response of SquadsApi->list_squads:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SquadsApi->list_squads: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**SquadListResponse**](SquadListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | List of squads |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **patch_squad**
> SquadResponse patch_squad(name, request, namespace=namespace)

Patch squad

Replaces the member list and/or orphanTTL on an existing squad. Retries once on 409 conflict.

### Example


```python
import komputer_ai
from komputer_ai.models.patch_squad_request import PatchSquadRequest
from komputer_ai.models.squad_response import SquadResponse
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
    api_instance = komputer_ai.SquadsApi(api_client)
    name = 'name_example' # str | Squad name
    request = komputer_ai.PatchSquadRequest() # PatchSquadRequest | Fields to update
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Patch squad
        api_response = api_instance.patch_squad(name, request, namespace=namespace)
        print("The response of SquadsApi->patch_squad:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SquadsApi->patch_squad: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Squad name | 
 **request** | [**PatchSquadRequest**](PatchSquadRequest.md)| Fields to update | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Updated squad |  -  |
**400** | Bad request |  -  |
**404** | Squad not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **remove_squad_member**
> SquadResponse remove_squad_member(name, agent, namespace=namespace)

Remove squad member

Removes the named member from the squad's member list (matched by ref.name or spec-based name).

### Example


```python
import komputer_ai
from komputer_ai.models.squad_response import SquadResponse
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
    api_instance = komputer_ai.SquadsApi(api_client)
    name = 'name_example' # str | Squad name
    agent = 'agent_example' # str | Agent name to remove
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Remove squad member
        api_response = api_instance.remove_squad_member(name, agent, namespace=namespace)
        print("The response of SquadsApi->remove_squad_member:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SquadsApi->remove_squad_member: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Squad name | 
 **agent** | **str**| Agent name to remove | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Updated squad |  -  |
**404** | Squad or member not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **squads_post**
> SquadResponse squads_post(request)

### Example


```python
import komputer_ai
from komputer_ai.models.create_squad_request import CreateSquadRequest
from komputer_ai.models.squad_response import SquadResponse
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
    api_instance = komputer_ai.SquadsApi(api_client)
    request = komputer_ai.CreateSquadRequest() # CreateSquadRequest | Squad creation request

    try:
        api_response = api_instance.squads_post(request)
        print("The response of SquadsApi->squads_post:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SquadsApi->squads_post: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateSquadRequest**](CreateSquadRequest.md)| Squad creation request | 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Created squad |  -  |
**400** | Bad request |  -  |
**409** | Squad already exists |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

