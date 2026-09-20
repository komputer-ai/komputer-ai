# ConnectorsApi

All URIs are relative to *http://localhost:8080/api/v1*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**createConnector**](ConnectorsApi.md#createconnectoroperation) | **POST** /connectors | Create connector |
| [**deleteConnector**](ConnectorsApi.md#deleteconnector) | **DELETE** /connectors/{name} | Delete connector |
| [**getConnector**](ConnectorsApi.md#getconnector) | **GET** /connectors/{name} | Get connector details |
| [**listConnectorTools**](ConnectorsApi.md#listconnectortools) | **GET** /connectors/{name}/tools | List connector tools |
| [**listConnectors**](ConnectorsApi.md#listconnectors) | **GET** /connectors | List connectors |
| [**updateConnector**](ConnectorsApi.md#updateconnectoroperation) | **PATCH** /connectors/{name} | Update connector token |



## createConnector

> ConnectorResponse createConnector(request)

Create connector

Creates a new KomputerConnector CR pointing to an MCP server that can be attached to agents.

### Example

```ts
import {
  Configuration,
  ConnectorsApi,
} from '@komputer-ai/sdk';
import type { CreateConnectorOperationRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new ConnectorsApi();

  const body = {
    // CreateConnectorRequest | Connector creation request
    request: ...,
  } satisfies CreateConnectorOperationRequest;

  try {
    const data = await api.createConnector(body);
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
| **request** | [CreateConnectorRequest](CreateConnectorRequest.md) | Connector creation request | |

### Return type

[**ConnectorResponse**](ConnectorResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Connector created |  -  |
| **400** | Bad request |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## deleteConnector

> { [key: string]: string; } deleteConnector(name, namespace)

Delete connector

Deletes the connector CR.

### Example

```ts
import {
  Configuration,
  ConnectorsApi,
} from '@komputer-ai/sdk';
import type { DeleteConnectorRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new ConnectorsApi();

  const body = {
    // string | Connector name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies DeleteConnectorRequest;

  try {
    const data = await api.deleteConnector(body);
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
| **name** | `string` | Connector name | [Defaults to `undefined`] |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

**{ [key: string]: string; }**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Connector deleted |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## getConnector

> ConnectorResponse getConnector(name, namespace)

Get connector details

Returns the URL, service, type, and auth config for a single connector.

### Example

```ts
import {
  Configuration,
  ConnectorsApi,
} from '@komputer-ai/sdk';
import type { GetConnectorRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new ConnectorsApi();

  const body = {
    // string | Connector name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies GetConnectorRequest;

  try {
    const data = await api.getConnector(body);
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
| **name** | `string` | Connector name | [Defaults to `undefined`] |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**ConnectorResponse**](ConnectorResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Connector details |  -  |
| **404** | Connector not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## listConnectorTools

> { [key: string]: any; } listConnectorTools(name, namespace)

List connector tools

Calls the MCP server\&#39;s tools/list endpoint and returns the available tools.

### Example

```ts
import {
  Configuration,
  ConnectorsApi,
} from '@komputer-ai/sdk';
import type { ListConnectorToolsRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new ConnectorsApi();

  const body = {
    // string | Connector name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies ListConnectorToolsRequest;

  try {
    const data = await api.listConnectorTools(body);
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
| **name** | `string` | Connector name | [Defaults to `undefined`] |
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
| **200** | List of MCP tools |  -  |
| **404** | Connector not found |  -  |
| **500** | Internal error |  -  |
| **502** | Failed to reach MCP server |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## listConnectors

> { [key: string]: any; } listConnectors(namespace)

List connectors

Returns all connectors with attached agent counts in the specified namespace.

### Example

```ts
import {
  Configuration,
  ConnectorsApi,
} from '@komputer-ai/sdk';
import type { ListConnectorsRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new ConnectorsApi();

  const body = {
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies ListConnectorsRequest;

  try {
    const data = await api.listConnectors(body);
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
| **200** | List of connectors |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## updateConnector

> ConnectorResponse updateConnector(name, request, namespace)

Update connector token

Replaces the auth token of a token/header connector. Writes the new value into the connector\&#39;s existing secret, or creates a managed &lt;name&gt;-credentials secret if the connector has none. OAuth connectors are rejected; reconnect them via the OAuth flow instead.

### Example

```ts
import {
  Configuration,
  ConnectorsApi,
} from '@komputer-ai/sdk';
import type { UpdateConnectorOperationRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new ConnectorsApi();

  const body = {
    // string | Connector name
    name: name_example,
    // UpdateConnectorRequest | New token
    request: ...,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies UpdateConnectorOperationRequest;

  try {
    const data = await api.updateConnector(body);
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
| **name** | `string` | Connector name | [Defaults to `undefined`] |
| **request** | [UpdateConnectorRequest](UpdateConnectorRequest.md) | New token | |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**ConnectorResponse**](ConnectorResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Connector updated |  -  |
| **400** | Bad request |  -  |
| **404** | Connector not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

