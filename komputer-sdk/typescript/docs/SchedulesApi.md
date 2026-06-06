# SchedulesApi

All URIs are relative to *http://localhost:8080/api/v1*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**createSchedule**](SchedulesApi.md#createscheduleoperation) | **POST** /schedules | Create schedule |
| [**deleteSchedule**](SchedulesApi.md#deleteschedule) | **DELETE** /schedules/{name} | Delete schedule |
| [**getSchedule**](SchedulesApi.md#getschedule) | **GET** /schedules/{name} | Get schedule details |
| [**listSchedules**](SchedulesApi.md#listschedules) | **GET** /schedules | List schedules |
| [**patchSchedule**](SchedulesApi.md#patchscheduleoperation) | **PATCH** /schedules/{name} | Patch schedule |
| [**triggerSchedule**](SchedulesApi.md#triggerschedule) | **POST** /schedules/{name}/trigger | Trigger schedule now |



## createSchedule

> ScheduleResponse createSchedule(request)

Create schedule

Creates a new KomputerSchedule CR that triggers agent tasks on a cron schedule.

### Example

```ts
import {
  Configuration,
  SchedulesApi,
} from '@komputer-ai/sdk';
import type { CreateScheduleOperationRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SchedulesApi();

  const body = {
    // CreateScheduleRequest | Schedule creation request
    request: ...,
  } satisfies CreateScheduleOperationRequest;

  try {
    const data = await api.createSchedule(body);
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
| **request** | [CreateScheduleRequest](CreateScheduleRequest.md) | Schedule creation request | |

### Return type

[**ScheduleResponse**](ScheduleResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Schedule created |  -  |
| **400** | Bad request |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## deleteSchedule

> { [key: string]: string; } deleteSchedule(name, namespace)

Delete schedule

Deletes the schedule CR. Does not delete any agents that were created by the schedule.

### Example

```ts
import {
  Configuration,
  SchedulesApi,
} from '@komputer-ai/sdk';
import type { DeleteScheduleRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SchedulesApi();

  const body = {
    // string | Schedule name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies DeleteScheduleRequest;

  try {
    const data = await api.deleteSchedule(body);
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
| **name** | `string` | Schedule name | [Defaults to `undefined`] |
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
| **200** | Schedule deleted |  -  |
| **404** | Schedule not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## getSchedule

> ScheduleResponse getSchedule(name, namespace)

Get schedule details

Returns the current status and run history for a single schedule.

### Example

```ts
import {
  Configuration,
  SchedulesApi,
} from '@komputer-ai/sdk';
import type { GetScheduleRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SchedulesApi();

  const body = {
    // string | Schedule name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies GetScheduleRequest;

  try {
    const data = await api.getSchedule(body);
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
| **name** | `string` | Schedule name | [Defaults to `undefined`] |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**ScheduleResponse**](ScheduleResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Schedule details |  -  |
| **404** | Schedule not found |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## listSchedules

> ScheduleListResponse listSchedules(namespace)

List schedules

Returns all schedules with their current status and run history in the specified namespace.

### Example

```ts
import {
  Configuration,
  SchedulesApi,
} from '@komputer-ai/sdk';
import type { ListSchedulesRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SchedulesApi();

  const body = {
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies ListSchedulesRequest;

  try {
    const data = await api.listSchedules(body);
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

[**ScheduleListResponse**](ScheduleListResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | List of schedules |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## patchSchedule

> ScheduleResponse patchSchedule(name, request, namespace)

Patch schedule

Updates the cron expression for an existing schedule.

### Example

```ts
import {
  Configuration,
  SchedulesApi,
} from '@komputer-ai/sdk';
import type { PatchScheduleOperationRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SchedulesApi();

  const body = {
    // string | Schedule name
    name: name_example,
    // PatchScheduleRequest | Fields to update
    request: ...,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies PatchScheduleOperationRequest;

  try {
    const data = await api.patchSchedule(body);
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
| **name** | `string` | Schedule name | [Defaults to `undefined`] |
| **request** | [PatchScheduleRequest](PatchScheduleRequest.md) | Fields to update | |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**ScheduleResponse**](ScheduleResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Updated schedule |  -  |
| **400** | Bad request |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## triggerSchedule

> TriggerScheduleResponse triggerSchedule(name, namespace)

Trigger schedule now

Fires the schedule immediately, independent of its cron cadence. Returns 409 if a run is already in progress.

### Example

```ts
import {
  Configuration,
  SchedulesApi,
} from '@komputer-ai/sdk';
import type { TriggerScheduleRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new SchedulesApi();

  const body = {
    // string | Schedule name
    name: name_example,
    // string | Kubernetes namespace (optional)
    namespace: namespace_example,
  } satisfies TriggerScheduleRequest;

  try {
    const data = await api.triggerSchedule(body);
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
| **name** | `string` | Schedule name | [Defaults to `undefined`] |
| **namespace** | `string` | Kubernetes namespace | [Optional] [Defaults to `undefined`] |

### Return type

[**TriggerScheduleResponse**](TriggerScheduleResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Triggered |  -  |
| **404** | Schedule not found |  -  |
| **409** | A run is already in progress |  -  |
| **500** | Internal error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

