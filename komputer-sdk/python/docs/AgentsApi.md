# komputer_ai.AgentsApi

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**agents_name_ws_get**](AgentsApi.md#agents_name_ws_get) | **GET** /agents/{name}/ws | Stream agent events (WebSocket)
[**cancel_agent_task**](AgentsApi.md#cancel_agent_task) | **POST** /agents/{name}/cancel | Cancel agent task
[**create_agent**](AgentsApi.md#create_agent) | **POST** /agents | Create agent or send task
[**delete_agent**](AgentsApi.md#delete_agent) | **DELETE** /agents/{name} | Delete agent
[**get_agent**](AgentsApi.md#get_agent) | **GET** /agents/{name} | Get agent details
[**get_agent_events**](AgentsApi.md#get_agent_events) | **GET** /agents/{name}/events | Get agent events
[**list_agents**](AgentsApi.md#list_agents) | **GET** /agents | List agents
[**patch_agent**](AgentsApi.md#patch_agent) | **PATCH** /agents/{name} | Patch agent


# **agents_name_ws_get**
> agents_name_ws_get(name)

Stream agent events (WebSocket)

Upgrades to a WebSocket connection to stream real-time agent events. Events include task_started, thinking, tool_call, tool_result, text, task_completed, task_cancelled, and error.

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
    api_instance = komputer_ai.AgentsApi(api_client)
    name = 'name_example' # str | Agent name

    try:
        # Stream agent events (WebSocket)
        api_instance.agents_name_ws_get(name)
    except Exception as e:
        print("Exception when calling AgentsApi->agents_name_ws_get: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Agent name | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined


[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **cancel_agent_task**
> Dict[str, str] cancel_agent_task(name, namespace=namespace)

Cancel agent task

Gracefully cancels the currently running task. The agent pod stays alive for future tasks.

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
    api_instance = komputer_ai.AgentsApi(api_client)
    name = 'name_example' # str | Agent name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Cancel agent task
        api_response = api_instance.cancel_agent_task(name, namespace=namespace)
        print("The response of AgentsApi->cancel_agent_task:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AgentsApi->cancel_agent_task: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Agent name | 
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
**200** | Task cancelling |  -  |
**404** | Agent not found |  -  |
**409** | Agent has no running pod |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_agent**
> AgentResponse create_agent(request)

Create agent or send task

Creates a new agent or sends a task to an existing idle agent (upsert by name).
If the agent doesn't exist, it is created. If it exists and is idle, the task is forwarded.

### Example


```python
import komputer_ai
from komputer_ai.models.agent_response import AgentResponse
from komputer_ai.models.create_agent_request import CreateAgentRequest
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
    api_instance = komputer_ai.AgentsApi(api_client)
    request = komputer_ai.CreateAgentRequest() # CreateAgentRequest | Agent creation request

    try:
        # Create agent or send task
        api_response = api_instance.create_agent(request)
        print("The response of AgentsApi->create_agent:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AgentsApi->create_agent: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateAgentRequest**](CreateAgentRequest.md)| Agent creation request | 

### Return type

[**AgentResponse**](AgentResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Agent created or task forwarded |  -  |
**400** | Bad request |  -  |
**409** | Agent is busy or has no running pod |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_agent**
> Dict[str, str] delete_agent(name, namespace=namespace)

Delete agent

Deletes the agent CR, pod, PVC, secrets, and Redis event stream.

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
    api_instance = komputer_ai.AgentsApi(api_client)
    name = 'name_example' # str | Agent name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Delete agent
        api_response = api_instance.delete_agent(name, namespace=namespace)
        print("The response of AgentsApi->delete_agent:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AgentsApi->delete_agent: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Agent name | 
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
**200** | Agent deleted |  -  |
**404** | Agent not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_agent**
> AgentResponse get_agent(name, namespace=namespace)

Get agent details

Returns the current status and metadata for a single agent.

### Example


```python
import komputer_ai
from komputer_ai.models.agent_response import AgentResponse
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
    api_instance = komputer_ai.AgentsApi(api_client)
    name = 'name_example' # str | Agent name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Get agent details
        api_response = api_instance.get_agent(name, namespace=namespace)
        print("The response of AgentsApi->get_agent:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AgentsApi->get_agent: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Agent name | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**AgentResponse**](AgentResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Agent details |  -  |
**404** | Agent not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_agent_events**
> Dict[str, object] get_agent_events(name, namespace=namespace, limit=limit)

Get agent events

Returns recent events from the agent's Redis stream in chronological order.

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
    api_instance = komputer_ai.AgentsApi(api_client)
    name = 'name_example' # str | Agent name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)
    limit = 50 # int | Max events to return (1-200) (optional) (default to 50)

    try:
        # Get agent events
        api_response = api_instance.get_agent_events(name, namespace=namespace, limit=limit)
        print("The response of AgentsApi->get_agent_events:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AgentsApi->get_agent_events: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Agent name | 
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
**200** | Agent events |  -  |
**400** | Invalid limit parameter |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_agents**
> AgentListResponse list_agents(namespace=namespace)

List agents

Returns all agents with their current status in the specified namespace.

### Example


```python
import komputer_ai
from komputer_ai.models.agent_list_response import AgentListResponse
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
    api_instance = komputer_ai.AgentsApi(api_client)
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # List agents
        api_response = api_instance.list_agents(namespace=namespace)
        print("The response of AgentsApi->list_agents:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AgentsApi->list_agents: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**AgentListResponse**](AgentListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | List of agents |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **patch_agent**
> AgentResponse patch_agent(name, request, namespace=namespace)

Patch agent

Updates model, lifecycle, instructions, secretRefs, memories, skills, or connectors on an existing agent.

### Example


```python
import komputer_ai
from komputer_ai.models.agent_response import AgentResponse
from komputer_ai.models.patch_agent_request import PatchAgentRequest
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
    api_instance = komputer_ai.AgentsApi(api_client)
    name = 'name_example' # str | Agent name
    request = komputer_ai.PatchAgentRequest() # PatchAgentRequest | Fields to update
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Patch agent
        api_response = api_instance.patch_agent(name, request, namespace=namespace)
        print("The response of AgentsApi->patch_agent:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AgentsApi->patch_agent: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Agent name | 
 **request** | [**PatchAgentRequest**](PatchAgentRequest.md)| Fields to update | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**AgentResponse**](AgentResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Updated agent |  -  |
**400** | Bad request |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

