# SquadsApi

All URIs are relative to *http://localhost:8080/api/v1*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**addSquadMember**](SquadsApi.md#addsquadmemberoperation) | **POST** /squads/{name}/members | Add squad member |
| [**createSquad**](SquadsApi.md#createsquadoperation) | **POST** /squads | Create squad |
| [**deleteSquad**](SquadsApi.md#deletesquad) | **DELETE** /squads/{name} | Delete squad |
| [**getSquad**](SquadsApi.md#getsquad) | **GET** /squads/{name} | Get squad details |
| [**listSquads**](SquadsApi.md#listsquads) | **GET** /squads | List squads |
| [**patchSquad**](SquadsApi.md#patchsquadoperation) | **PATCH** /squads/{name} | Patch squad |
| [**removeSquadMember**](SquadsApi.md#removesquadmember) | **DELETE** /squads/{name}/members/{agent} | Remove squad member |



## addSquadMember

> SquadResponse addSquadMember(name, request, namespace)

Add squad member

Appends a member (by ref or inline spec) to the squad\&#39;s member list.

### Example

```ts
import {
  Configuration,
  SquadsApi,
} from '@komputer-ai/sdk';
import type { AddSquadMemberOperationRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SquadsApi();

  const body = {
    // string | Squad name
    name: name_example,
    // AddSquadMemberRequest | Member to add
    request: ...,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies AddSquadMemberOperationRequest;

  try {
    const data = await api.addSquadMember(body);
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
| **name** | `string` | Squad name | [Defaults to `undefined`] |
| **request** | [AddSquadMemberRequest](AddSquadMemberRequest.md) | Member to add | |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Updated squad |  -  |
| **400** | Bad request |  -  |
| **404** | Squad not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## createSquad

> SquadResponse createSquad(request)

Create squad

Creates a new squad with the given members.

### Example

```ts
import {
  Configuration,
  SquadsApi,
} from '@komputer-ai/sdk';
import type { CreateSquadOperationRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SquadsApi();

  const body = {
    // CreateSquadRequest | Squad creation request
    request: ...,
  } satisfies CreateSquadOperationRequest;

  try {
    const data = await api.createSquad(body);
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
| **request** | [CreateSquadRequest](CreateSquadRequest.md) | Squad creation request | |

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Created squad |  -  |
| **400** | Bad request |  -  |
| **409** | Squad already exists |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## deleteSquad

> { [key: string]: string; } deleteSquad(name, namespace)

Delete squad

Deletes the squad CR. The operator will clean up the shared pod.

### Example

```ts
import {
  Configuration,
  SquadsApi,
} from '@komputer-ai/sdk';
import type { DeleteSquadRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SquadsApi();

  const body = {
    // string | Squad name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies DeleteSquadRequest;

  try {
    const data = await api.deleteSquad(body);
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
| **name** | `string` | Squad name | [Defaults to `undefined`] |
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
| **200** | Squad deleted |  -  |
| **404** | Squad not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## getSquad

> SquadResponse getSquad(name, namespace)

Get squad details

Returns the current status and member list for a single squad.

### Example

```ts
import {
  Configuration,
  SquadsApi,
} from '@komputer-ai/sdk';
import type { GetSquadRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SquadsApi();

  const body = {
    // string | Squad name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies GetSquadRequest;

  try {
    const data = await api.getSquad(body);
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
| **name** | `string` | Squad name | [Defaults to `undefined`] |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Squad details |  -  |
| **404** | Squad not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## listSquads

> SquadListResponse listSquads(namespace)

List squads

Returns all squads with their current status. Pass ?namespace&#x3D; to filter; omit for all namespaces.

### Example

```ts
import {
  Configuration,
  SquadsApi,
} from '@komputer-ai/sdk';
import type { ListSquadsRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SquadsApi();

  const body = {
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies ListSquadsRequest;

  try {
    const data = await api.listSquads(body);
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

[**SquadListResponse**](SquadListResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | List of squads |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## patchSquad

> SquadResponse patchSquad(name, request, namespace)

Patch squad

Replaces the member list and/or orphanTTL on an existing squad. Retries once on 409 conflict.

### Example

```ts
import {
  Configuration,
  SquadsApi,
} from '@komputer-ai/sdk';
import type { PatchSquadOperationRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SquadsApi();

  const body = {
    // string | Squad name
    name: name_example,
    // PatchSquadRequest | Fields to update
    request: ...,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies PatchSquadOperationRequest;

  try {
    const data = await api.patchSquad(body);
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
| **name** | `string` | Squad name | [Defaults to `undefined`] |
| **request** | [PatchSquadRequest](PatchSquadRequest.md) | Fields to update | |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Updated squad |  -  |
| **400** | Bad request |  -  |
| **404** | Squad not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## removeSquadMember

> SquadResponse removeSquadMember(name, agent, namespace)

Remove squad member

Removes the named member from the squad\&#39;s member list (matched by ref.name or spec-based name).

### Example

```ts
import {
  Configuration,
  SquadsApi,
} from '@komputer-ai/sdk';
import type { RemoveSquadMemberRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SquadsApi();

  const body = {
    // string | Squad name
    name: name_example,
    // string | Agent name to remove
    agent: agent_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies RemoveSquadMemberRequest;

  try {
    const data = await api.removeSquadMember(body);
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
| **name** | `string` | Squad name | [Defaults to `undefined`] |
| **agent** | `string` | Agent name to remove | [Defaults to `undefined`] |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Updated squad |  -  |
| **404** | Squad or member not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

