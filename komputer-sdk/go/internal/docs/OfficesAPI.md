# \OfficesAPI

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**DeleteOffice**](OfficesAPI.md#DeleteOffice) | **Delete** /offices/{name} | Delete office
[**GetOffice**](OfficesAPI.md#GetOffice) | **Get** /offices/{name} | Get office details
[**GetOfficeEvents**](OfficesAPI.md#GetOfficeEvents) | **Get** /offices/{name}/events | Get office events
[**ListOffices**](OfficesAPI.md#ListOffices) | **Get** /offices | List offices



## DeleteOffice

> map[string]string DeleteOffice(ctx, name).Namespace(namespace).Execute()

Delete office



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
	name := "name_example" // string | Office name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OfficesAPI.DeleteOffice(context.Background(), name).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OfficesAPI.DeleteOffice``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DeleteOffice`: map[string]string
	fmt.Fprintf(os.Stdout, "Response from `OfficesAPI.DeleteOffice`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Office name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteOfficeRequest struct via the builder pattern


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


## GetOffice

> OfficeResponse GetOffice(ctx, name).Namespace(namespace).Execute()

Get office details



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
	name := "name_example" // string | Office name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OfficesAPI.GetOffice(context.Background(), name).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OfficesAPI.GetOffice``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetOffice`: OfficeResponse
	fmt.Fprintf(os.Stdout, "Response from `OfficesAPI.GetOffice`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Office name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetOfficeRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**OfficeResponse**](OfficeResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetOfficeEvents

> map[string]interface{} GetOfficeEvents(ctx, name).Namespace(namespace).Limit(limit).Execute()

Get office events



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
	name := "name_example" // string | Office name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)
	limit := int32(56) // int32 | Max events to return (1-200) (optional) (default to 50)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OfficesAPI.GetOfficeEvents(context.Background(), name).Namespace(namespace).Limit(limit).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OfficesAPI.GetOfficeEvents``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetOfficeEvents`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `OfficesAPI.GetOfficeEvents`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Office name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetOfficeEventsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **namespace** | **string** | Kubernetes namespace | 
 **limit** | **int32** | Max events to return (1-200) | [default to 50]

### Return type

**map[string]interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListOffices

> OfficeListResponse ListOffices(ctx).Namespace(namespace).Execute()

List offices



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
	resp, r, err := apiClient.OfficesAPI.ListOffices(context.Background()).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OfficesAPI.ListOffices``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListOffices`: OfficeListResponse
	fmt.Fprintf(os.Stdout, "Response from `OfficesAPI.ListOffices`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListOfficesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**OfficeListResponse**](OfficeListResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

