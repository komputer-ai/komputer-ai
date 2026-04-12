# komputer_ai.SkillsApi

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_skill**](SkillsApi.md#create_skill) | **POST** /skills | Create skill
[**delete_skill**](SkillsApi.md#delete_skill) | **DELETE** /skills/{name} | Delete skill
[**get_skill**](SkillsApi.md#get_skill) | **GET** /skills/{name} | Get skill details
[**list_skills**](SkillsApi.md#list_skills) | **GET** /skills | List skills
[**patch_skill**](SkillsApi.md#patch_skill) | **PATCH** /skills/{name} | Patch skill


# **create_skill**
> SkillResponse create_skill(request)

Create skill

Creates a new KomputerSkill CR with script content that can be attached to agents.

### Example


```python
import komputer_ai
from komputer_ai.models.create_skill_request import CreateSkillRequest
from komputer_ai.models.skill_response import SkillResponse
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
    api_instance = komputer_ai.SkillsApi(api_client)
    request = komputer_ai.CreateSkillRequest() # CreateSkillRequest | Skill creation request

    try:
        # Create skill
        api_response = api_instance.create_skill(request)
        print("The response of SkillsApi->create_skill:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SkillsApi->create_skill: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateSkillRequest**](CreateSkillRequest.md)| Skill creation request | 

### Return type

[**SkillResponse**](SkillResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | Skill created |  -  |
**400** | Bad request |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_skill**
> Dict[str, str] delete_skill(name, namespace=namespace)

Delete skill

Deletes the skill CR.

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
    api_instance = komputer_ai.SkillsApi(api_client)
    name = 'name_example' # str | Skill name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Delete skill
        api_response = api_instance.delete_skill(name, namespace=namespace)
        print("The response of SkillsApi->delete_skill:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SkillsApi->delete_skill: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Skill name | 
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
**200** | Skill deleted |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_skill**
> SkillResponse get_skill(name, namespace=namespace)

Get skill details

Returns the content, description, and attached agent count for a single skill.

### Example


```python
import komputer_ai
from komputer_ai.models.skill_response import SkillResponse
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
    api_instance = komputer_ai.SkillsApi(api_client)
    name = 'name_example' # str | Skill name
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Get skill details
        api_response = api_instance.get_skill(name, namespace=namespace)
        print("The response of SkillsApi->get_skill:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SkillsApi->get_skill: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Skill name | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**SkillResponse**](SkillResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Skill details |  -  |
**404** | Skill not found |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_skills**
> Dict[str, object] list_skills(namespace=namespace)

List skills

Returns all skills with content and attached agent counts in the specified namespace.

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
    api_instance = komputer_ai.SkillsApi(api_client)
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # List skills
        api_response = api_instance.list_skills(namespace=namespace)
        print("The response of SkillsApi->list_skills:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SkillsApi->list_skills: %s\n" % e)
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
**200** | List of skills |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **patch_skill**
> SkillResponse patch_skill(name, request, namespace=namespace)

Patch skill

Updates the description or script content of an existing skill.

### Example


```python
import komputer_ai
from komputer_ai.models.patch_skill_request import PatchSkillRequest
from komputer_ai.models.skill_response import SkillResponse
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
    api_instance = komputer_ai.SkillsApi(api_client)
    name = 'name_example' # str | Skill name
    request = komputer_ai.PatchSkillRequest() # PatchSkillRequest | Fields to update
    namespace = 'namespace_example' # str | Kubernetes namespace (optional)

    try:
        # Patch skill
        api_response = api_instance.patch_skill(name, request, namespace=namespace)
        print("The response of SkillsApi->patch_skill:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling SkillsApi->patch_skill: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **str**| Skill name | 
 **request** | [**PatchSkillRequest**](PatchSkillRequest.md)| Fields to update | 
 **namespace** | **str**| Kubernetes namespace | [optional] 

### Return type

[**SkillResponse**](SkillResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | Updated skill |  -  |
**400** | Bad request |  -  |
**500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

