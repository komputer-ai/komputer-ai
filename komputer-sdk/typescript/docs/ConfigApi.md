# ConfigApi

All URIs are relative to *http://localhost:8080/api/v1*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**getConfig**](ConfigApi.md#getconfig) | **GET** /config | Get platform configuration |



## getConfig

> ConfigResponse getConfig()

Get platform configuration

Returns platform settings that change what clients can offer, such as whether the S3 session store is enabled. Never returns credentials.

### Example

```ts
import {
  Configuration,
  ConfigApi,
} from '@komputer-ai/sdk';
import type { GetConfigRequest } from '@komputer-ai/sdk';

async function example() {
  console.log("🚀 Testing @komputer-ai/sdk SDK...");
  const api = new ConfigApi();

  try {
    const data = await api.getConfig();
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

[**ConfigResponse**](ConfigResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Platform configuration |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

