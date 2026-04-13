# komputer_ai.SecretsApi

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_secret**](SecretsApi.md#create_secret) | **POST** /secrets | Create managed secret
[**delete_secret**](SecretsApi.md#delete_secret) | **DELETE** /secrets/{name} | Delete managed secret
[**list_secrets**](SecretsApi.md#list_secrets) | **GET** /secrets | List secrets
[**update_secret**](SecretsApi.md#update_secret) | **PATCH** /secrets/{name} | Update managed secret


# **create_secret**
> SecretResponse create_secret(request)

Create managed secret

Creates a new Kubernetes secret managed by komputer.ai that can be attached to agents.

### Example


```python
import komputer_ai
from komputer_ai.models.create_secret_request import CreateSecretRequest
from komputer_ai.models.secret_response import SecretResponse
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
    api_instance = komputer_ai.SecretsApi(api_client)
    request = komputer_ai.CreateSecretRequest() # CreateSecretRequest | Secret creation request

    try:
        # Create managed secret
        api_response = api_instance.create_secret(request)
        print("The response of SecretsApi->create_secret:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SecretsApi->create_secret: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateSecretRequest**](CreateSecretRequest.md)| Secret creation request | 

### Return type

[**SecretResponse**](SecretResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | Secret created |  -  |
**400** | Bad request |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_secret**
> Dict[str, str] delete_secret(name, namespace=namespace)

Delete managed secret

Deletes a managed Kubernetes secret.

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
    api_instance = komputer_ai.SecretsApi(api_client)
    name = 'name_example' # str | Secret name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Delete managed secret
        api_response = api_instance.delete_secret(name, namespace=namespace)
        print("The response of SecretsApi->delete_secret:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SecretsApi->delete_secret: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Secret name | 
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
**200** | Secret deleted |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_secrets**
> SecretListResponse list_secrets(namespace=namespace, all=all)

List secrets

Returns all secrets with key names (not values) and attached agent counts in the specified namespace.

### Example


```python
import komputer_ai
from komputer_ai.models.secret_list_response import SecretListResponse
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
    api_instance = komputer_ai.SecretsApi(api_client)
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)
    all = True # bool | Include all secrets, not just managed ones (optional)

    try:
        # List secrets
        api_response = api_instance.list_secrets(namespace=namespace, all=all)
        print("The response of SecretsApi->list_secrets:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SecretsApi->list_secrets: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| Kubernetes namespace | [optional] 
 **all** | **bool**| Include all secrets, not just managed ones | [optional] 

### Return type

[**SecretListResponse**](SecretListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | List of secrets |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **update_secret**
> Dict[str, str] update_secret(name, request, namespace=namespace)

Update managed secret

Replaces the key-value pairs in a managed Kubernetes secret.

### Example


```python
import komputer_ai
from komputer_ai.models.update_secret_request import UpdateSecretRequest
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
    api_instance = komputer_ai.SecretsApi(api_client)
    name = 'name_example' # str | Secret name
    request = komputer_ai.UpdateSecretRequest() # UpdateSecretRequest | Updated secret data
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Update managed secret
        api_response = api_instance.update_secret(name, request, namespace=namespace)
        print("The response of SecretsApi->update_secret:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SecretsApi->update_secret: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Secret name | 
 **request** | [**UpdateSecretRequest**](UpdateSecretRequest.md)| Updated secret data | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

**Dict[str, str]**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Secret updated |  -  |
**400** | Bad request |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

