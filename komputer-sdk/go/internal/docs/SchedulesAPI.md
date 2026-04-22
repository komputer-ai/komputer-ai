# \SchedulesAPI

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateSchedule**](SchedulesAPI.md#CreateSchedule) | **Post** /schedules | Create schedule
[**DeleteSchedule**](SchedulesAPI.md#DeleteSchedule) | **Delete** /schedules/{name} | Delete schedule
[**GetSchedule**](SchedulesAPI.md#GetSchedule) | **Get** /schedules/{name} | Get schedule details
[**ListSchedules**](SchedulesAPI.md#ListSchedules) | **Get** /schedules | List schedules
[**PatchSchedule**](SchedulesAPI.md#PatchSchedule) | **Patch** /schedules/{name} | Patch schedule



## CreateSchedule

> ScheduleResponse CreateSchedule(ctx).Request(request).Execute()

Create schedule



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/komputer-ai/komputer-ai/komputer"
)

func main() {
	request := *openapiclient.NewCreateScheduleRequest("Instructions_example", "Name_example", "Schedule_example") // CreateScheduleRequest | Schedule creation request

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SchedulesAPI.CreateSchedule(context.Background()).Request(request).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SchedulesAPI.CreateSchedule``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSchedule`: ScheduleResponse
	fmt.Fprintf(os.Stdout, "Response from `SchedulesAPI.CreateSchedule`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateScheduleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateScheduleRequest**](CreateScheduleRequest.md) | Schedule creation request | 

### Return type

[**ScheduleResponse**](ScheduleResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteSchedule

> map[string]string DeleteSchedule(ctx, name).Namespace(namespace).Execute()

Delete schedule



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/komputer-ai/komputer-ai/komputer"
)

func main() {
	name := "name_example" // string | Schedule name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SchedulesAPI.DeleteSchedule(context.Background(), name).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SchedulesAPI.DeleteSchedule``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DeleteSchedule`: map[string]string
	fmt.Fprintf(os.Stdout, "Response from `SchedulesAPI.DeleteSchedule`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Schedule name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteScheduleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **namespace** | **string** | Kubernetes namespace | 

### Return type

**map[string]string**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetSchedule

> ScheduleResponse GetSchedule(ctx, name).Namespace(namespace).Execute()

Get schedule details



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/komputer-ai/komputer-ai/komputer"
)

func main() {
	name := "name_example" // string | Schedule name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SchedulesAPI.GetSchedule(context.Background(), name).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SchedulesAPI.GetSchedule``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetSchedule`: ScheduleResponse
	fmt.Fprintf(os.Stdout, "Response from `SchedulesAPI.GetSchedule`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Schedule name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetScheduleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**ScheduleResponse**](ScheduleResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListSchedules

> ScheduleListResponse ListSchedules(ctx).Namespace(namespace).Execute()

List schedules



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/komputer-ai/komputer-ai/komputer"
)

func main() {
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SchedulesAPI.ListSchedules(context.Background()).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SchedulesAPI.ListSchedules``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSchedules`: ScheduleListResponse
	fmt.Fprintf(os.Stdout, "Response from `SchedulesAPI.ListSchedules`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListSchedulesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**ScheduleListResponse**](ScheduleListResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchSchedule

> ScheduleResponse PatchSchedule(ctx, name).Request(request).Namespace(namespace).Execute()

Patch schedule



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/komputer-ai/komputer-ai/komputer"
)

func main() {
	name := "name_example" // string | Schedule name
	request := *openapiclient.NewPatchScheduleRequest() // PatchScheduleRequest | Fields to update
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SchedulesAPI.PatchSchedule(context.Background(), name).Request(request).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SchedulesAPI.PatchSchedule``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchSchedule`: ScheduleResponse
	fmt.Fprintf(os.Stdout, "Response from `SchedulesAPI.PatchSchedule`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Schedule name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchScheduleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **request** | [**PatchScheduleRequest**](PatchScheduleRequest.md) | Fields to update | 
 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**ScheduleResponse**](ScheduleResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

