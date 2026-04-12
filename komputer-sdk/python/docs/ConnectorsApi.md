# komputer_ai.ConnectorsApi

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_connector**](ConnectorsApi.md#create_connector) | **POST** /connectors | Create connector
[**delete_connector**](ConnectorsApi.md#delete_connector) | **DELETE** /connectors/{name} | Delete connector
[**get_connector**](ConnectorsApi.md#get_connector) | **GET** /connectors/{name} | Get connector details
[**list_connector_tools**](ConnectorsApi.md#list_connector_tools) | **GET** /connectors/{name}/tools | List connector tools
[**list_connectors**](ConnectorsApi.md#list_connectors) | **GET** /connectors | List connectors


# **create_connector**
> ConnectorResponse create_connector(request)

Create connector

Creates a new KomputerConnector CR pointing to an MCP server that can be attached to agents.

### Example


```python
import komputer_ai
from komputer_ai.models.connector_response import ConnectorResponse
from komputer_ai.models.create_connector_request import CreateConnectorRequest
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
    api_instance = komputer_ai.ConnectorsApi(api_client)
    request = komputer_ai.CreateConnectorRequest() # CreateConnectorRequest | Connector creation request

    try:
        # Create connector
        api_response = api_instance.create_connector(request)
        print("The response of ConnectorsApi->create_connector:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ConnectorsApi->create_connector: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateConnectorRequest**](CreateConnectorRequest.md)| Connector creation request | 

### Return type

[**ConnectorResponse**](ConnectorResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | Connector created |  -  |
**400** | Bad request |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_connector**
> Dict[str, str] delete_connector(name, namespace=namespace)

Delete connector

Deletes the connector CR.

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
    api_instance = komputer_ai.ConnectorsApi(api_client)
    name = 'name_example' # str | Connector name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Delete connector
        api_response = api_instance.delete_connector(name, namespace=namespace)
        print("The response of ConnectorsApi->delete_connector:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ConnectorsApi->delete_connector: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Connector name | 
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
**200** | Connector deleted |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_connector**
> ConnectorResponse get_connector(name, namespace=namespace)

Get connector details

Returns the URL, service, type, and auth config for a single connector.

### Example


```python
import komputer_ai
from komputer_ai.models.connector_response import ConnectorResponse
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
    api_instance = komputer_ai.ConnectorsApi(api_client)
    name = 'name_example' # str | Connector name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Get connector details
        api_response = api_instance.get_connector(name, namespace=namespace)
        print("The response of ConnectorsApi->get_connector:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ConnectorsApi->get_connector: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Connector name | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**ConnectorResponse**](ConnectorResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Connector details |  -  |
**404** | Connector not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_connector_tools**
> Dict[str, object] list_connector_tools(name, namespace=namespace)

List connector tools

Calls the MCP server's tools/list endpoint and returns the available tools.

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
    api_instance = komputer_ai.ConnectorsApi(api_client)
    name = 'name_example' # str | Connector name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # List connector tools
        api_response = api_instance.list_connector_tools(name, namespace=namespace)
        print("The response of ConnectorsApi->list_connector_tools:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ConnectorsApi->list_connector_tools: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Connector name | 
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
**200** | List of MCP tools |  -  |
**404** | Connector not found |  -  |
**500** | Internal error |  -  |
**502** | Failed to reach MCP server |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_connectors**
> Dict[str, object] list_connectors(namespace=namespace)

List connectors

Returns all connectors with attached agent counts in the specified namespace.

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
    api_instance = komputer_ai.ConnectorsApi(api_client)
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # List connectors
        api_response = api_instance.list_connectors(namespace=namespace)
        print("The response of ConnectorsApi->list_connectors:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling ConnectorsApi->list_connectors: %s\n" % e)
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
**200** | List of connectors |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

