# AgentsApi

All URIs are relative to *http://localhost:8080/api/v1*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**agentsNameWsGet**](AgentsApi.md#agentsnamewsget) | **GET** /agents/{name}/ws | Stream agent events (WebSocket) |
| [**cancelAgentTask**](AgentsApi.md#cancelagenttask) | **POST** /agents/{name}/cancel | Cancel agent task |
| [**compactAgentTask**](AgentsApi.md#compactagenttask) | **POST** /agents/{name}/compact | Compact agent conversation |
| [**createAgent**](AgentsApi.md#createagentoperation) | **POST** /agents | Create agent or send task |
| [**deleteAgent**](AgentsApi.md#deleteagent) | **DELETE** /agents/{name} | Delete agent |
| [**getAgent**](AgentsApi.md#getagent) | **GET** /agents/{name} | Get agent details |
| [**getAgentCostBreakdown**](AgentsApi.md#getagentcostbreakdown) | **GET** /agents/{name}/cost | Get agent cost breakdown |
| [**getAgentEvents**](AgentsApi.md#getagentevents) | **GET** /agents/{name}/events | Get agent events |
| [**listAgents**](AgentsApi.md#listagents) | **GET** /agents | List agents |
| [**patchAgent**](AgentsApi.md#patchagentoperation) | **PATCH** /agents/{name} | Patch agent |



## agentsNameWsGet

> agentsNameWsGet(name, group)

Stream agent events (WebSocket)

Upgrades to a WebSocket connection to stream real-time agent events. Pass &#x60;?group&#x3D;&lt;name&gt;&#x60; to join a consumer group: each event in the stream is delivered to exactly one client per group across all API replicas (useful for distributed consumers). Without &#x60;group&#x60;, every connected client receives every event (legacy broadcast).

### Example

```ts
import {
  Configuration,
  AgentsApi,
} from '@komputer-ai/sdk';
import type { AgentsNameWsGetRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new AgentsApi();

  const body = {
    // string | Agent name
    name: name_example,
    // string | Consumer group name. When set, this connection joins a group and Redis-coordinated single-delivery applies. (optional)
    group: group_example,
  } satisfies AgentsNameWsGetRequest;

  try {
    const data = await api.agentsNameWsGet(body);
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
| **name** | `string` | Agent name | [Defaults to `undefined`] |
| **group** | `string` | Consumer group name. When set, this connection joins a group and Redis-coordinated single-delivery applies. | [Optional] [Defaults to `undefined`] |

### Return type

`void` (Empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## cancelAgentTask

> { [key: string]: string; } cancelAgentTask(name, namespace)

Cancel agent task

Gracefully cancels the currently running task. The agent pod stays alive for future tasks.

### Example

```ts
import {
  Configuration,
  AgentsApi,
} from '@komputer-ai/sdk';
import type { CancelAgentTaskRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new AgentsApi();

  const body = {
    // string | Agent name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies CancelAgentTaskRequest;

  try {
    const data = await api.cancelAgentTask(body);
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
| **name** | `string` | Agent name | [Defaults to `undefined`] |
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
| **200** | Task cancelling |  -  |
| **404** | Agent not found |  -  |
| **409** | Agent has no running pod |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## compactAgentTask

> { [key: string]: string; } compactAgentTask(name, namespace, request)

Compact agent conversation

Triggers manual conversation compaction. Older turns are summarized to free context space. Requires an active task — returns 409 if the agent is idle. Optional &#x60;instructions&#x60; field passes custom guidance to the compactor.

### Example

```ts
import {
  Configuration,
  AgentsApi,
} from '@komputer-ai/sdk';
import type { CompactAgentTaskRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new AgentsApi();

  const body = {
    // string | Agent name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
    // CompactAgentRequest | Optional compaction instructions (optional)
    request: ...,
  } satisfies CompactAgentTaskRequest;

  try {
    const data = await api.compactAgentTask(body);
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
| **name** | `string` | Agent name | [Defaults to `undefined`] |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |
| **request** | [CompactAgentRequest](CompactAgentRequest.md) | Optional compaction instructions | [Optional] |

### Return type

**{ [key: string]: string; }**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Compaction triggered |  -  |
| **404** | Agent not found |  -  |
| **409** | Agent has no running pod or no active task |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## createAgent

> AgentResponse createAgent(request)

Create agent or send task

Creates a new agent or sends a task to an existing idle agent (upsert by name). If the agent doesn\&#39;t exist, it is created. If it exists and is idle, the task is forwarded.

### Example

```ts
import {
  Configuration,
  AgentsApi,
} from '@komputer-ai/sdk';
import type { CreateAgentOperationRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new AgentsApi();

  const body = {
    // CreateAgentRequest | Agent creation request
    request: ...,
  } satisfies CreateAgentOperationRequest;

  try {
    const data = await api.createAgent(body);
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
| **request** | [CreateAgentRequest](CreateAgentRequest.md) | Agent creation request | |

### Return type

[**AgentResponse**](AgentResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Agent created or task forwarded |  -  |
| **400** | Bad request |  -  |
| **409** | Agent is busy or has no running pod |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## deleteAgent

> { [key: string]: string; } deleteAgent(name, namespace, recreatePod)

Delete agent

Deletes the agent CR, pod, PVC, secrets, and Redis event stream.

### Example

```ts
import {
  Configuration,
  AgentsApi,
} from '@komputer-ai/sdk';
import type { DeleteAgentRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new AgentsApi();

  const body = {
    // string | Agent name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
    // boolean | If the agent is a squad member, also delete the squad pod so it is rebuilt without this member (optional)
    recreatePod: true,
  } satisfies DeleteAgentRequest;

  try {
    const data = await api.deleteAgent(body);
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
| **name** | `string` | Agent name | [Defaults to `undefined`] |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |
| **recreatePod** | `boolean` | If the agent is a squad member, also delete the squad pod so it is rebuilt without this member | [Optional] [Defaults to `undefined`] |

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
| **200** | Agent deleted |  -  |
| **404** | Agent not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## getAgent

> AgentResponse getAgent(name, namespace)

Get agent details

Returns the current status and metadata for a single agent.

### Example

```ts
import {
  Configuration,
  AgentsApi,
} from '@komputer-ai/sdk';
import type { GetAgentRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new AgentsApi();

  const body = {
    // string | Agent name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies GetAgentRequest;

  try {
    const data = await api.getAgent(body);
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
| **name** | `string` | Agent name | [Defaults to `undefined`] |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**AgentResponse**](AgentResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Agent details |  -  |
| **404** | Agent not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## getAgentCostBreakdown

> CostBreakdownResponse getAgentCostBreakdown(name, refresh)

Get agent cost breakdown

Returns per-task costs for an agent, computed from its event history. Cached for 5 minutes unless refresh&#x3D;true.

### Example

```ts
import {
  Configuration,
  AgentsApi,
} from '@komputer-ai/sdk';
import type { GetAgentCostBreakdownRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new AgentsApi();

  const body = {
    // string | Agent name
    name: name_example,
    // boolean | Bypass the 5-minute cache (optional)
    refresh: true,
  } satisfies GetAgentCostBreakdownRequest;

  try {
    const data = await api.getAgentCostBreakdown(body);
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
| **name** | `string` | Agent name | [Defaults to `undefined`] |
| **refresh** | `boolean` | Bypass the 5-minute cache | [Optional] [Defaults to `undefined`] |

### Return type

[**CostBreakdownResponse**](CostBreakdownResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Cost breakdown |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## getAgentEvents

> { [key: string]: any; } getAgentEvents(name, namespace, limit, before, after, around)

Get agent events

Returns recent events from the agent\&#39;s Redis stream in chronological order.

### Example

```ts
import {
  Configuration,
  AgentsApi,
} from '@komputer-ai/sdk';
import type { GetAgentEventsRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new AgentsApi();

  const body = {
    // string | Agent name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
    // number | Max events to return (1-200) (optional)
    limit: 56,
    // string | RFC-3339 cursor: return events strictly before this timestamp (pagination) (optional)
    before: before_example,
    // string | RFC-3339 cursor: return events strictly after this timestamp (polling for new events) (optional)
    after: after_example,
    // string | RFC-3339 timestamp to center results on (optional)
    around: around_example,
  } satisfies GetAgentEventsRequest;

  try {
    const data = await api.getAgentEvents(body);
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
| **name** | `string` | Agent name | [Defaults to `undefined`] |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |
| **limit** | `number` | Max events to return (1-200) | [Optional] [Defaults to `50`] |
| **before** | `string` | RFC-3339 cursor: return events strictly before this timestamp (pagination) | [Optional] [Defaults to `undefined`] |
| **after** | `string` | RFC-3339 cursor: return events strictly after this timestamp (polling for new events) | [Optional] [Defaults to `undefined`] |
| **around** | `string` | RFC-3339 timestamp to center results on | [Optional] [Defaults to `undefined`] |

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
| **200** | Agent events |  -  |
| **400** | Invalid limit parameter |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## listAgents

> AgentListResponse listAgents(namespace, status, label)

List agents

Returns all agents with their current status in the specified namespace.

### Example

```ts
import {
  Configuration,
  AgentsApi,
} from '@komputer-ai/sdk';
import type { ListAgentsRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new AgentsApi();

  const body = {
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
    // string | Filter by agent phase (case-insensitive), e.g. Running, Sleeping (optional)
    status: status_example,
    // Array<string> | Label filter key=value; repeat for multiple (AND) (optional)
    label: ...,
  } satisfies ListAgentsRequest;

  try {
    const data = await api.listAgents(body);
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
| **status** | `string` | Filter by agent phase (case-insensitive), e.g. Running, Sleeping | [Optional] [Defaults to `undefined`] |
| **label** | `Array<string>` | Label filter key&#x3D;value; repeat for multiple (AND) | [Optional] |

### Return type

[**AgentListResponse**](AgentListResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | List of agents |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## patchAgent

> AgentResponse patchAgent(name, request, namespace)

Patch agent

Updates model, lifecycle, instructions, secretRefs, memories, skills, or connectors on an existing agent.

### Example

```ts
import {
  Configuration,
  AgentsApi,
} from '@komputer-ai/sdk';
import type { PatchAgentOperationRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new AgentsApi();

  const body = {
    // string | Agent name
    name: name_example,
    // PatchAgentRequest | Fields to update
    request: ...,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies PatchAgentOperationRequest;

  try {
    const data = await api.patchAgent(body);
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
| **name** | `string` | Agent name | [Defaults to `undefined`] |
| **request** | [PatchAgentRequest](PatchAgentRequest.md) | Fields to update | |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**AgentResponse**](AgentResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Updated agent |  -  |
| **400** | Bad request |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

