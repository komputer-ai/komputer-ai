# TemplatesApi

All URIs are relative to *http://localhost:8080/api/v1*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**listNamespaces**](TemplatesApi.md#listnamespaces) | **GET** /namespaces | List namespaces |
| [**listTemplates**](TemplatesApi.md#listtemplates) | **GET** /templates | List agent templates |



## listNamespaces

> { [key: string]: any; } listNamespaces()

List namespaces

Returns all Kubernetes namespaces the API has access to.

### Example

```ts
import {
  Configuration,
  TemplatesApi,
} from '@komputer-ai/sdk';
import type { ListNamespacesRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new TemplatesApi();

  try {
    const data = await api.listNamespaces();
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

This endpoint does not need any parameter.

### Return type

**{ [key: string]: any; }**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | List of namespaces |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## listTemplates

> { [key: string]: any; } listTemplates(namespace)

List agent templates

Returns all agent templates (both namespace-scoped and cluster-scoped) available in the specified namespace.

### Example

```ts
import {
  Configuration,
  TemplatesApi,
} from '@komputer-ai/sdk';
import type { ListTemplatesRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new TemplatesApi();

  const body = {
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies ListTemplatesRequest;

  try {
    const data = await api.listTemplates(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

**{ [key: string]: any; }**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | List of templates |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

