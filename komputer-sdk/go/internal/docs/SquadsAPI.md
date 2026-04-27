# \SquadsAPI

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddSquadMember**](SquadsAPI.md#AddSquadMember) | **Post** /squads/{name}/members | Add squad member
[**CreateSquad**](SquadsAPI.md#CreateSquad) | **Post** /squads | Create squad
[**DeleteSquad**](SquadsAPI.md#DeleteSquad) | **Delete** /squads/{name} | Delete squad
[**GetSquad**](SquadsAPI.md#GetSquad) | **Get** /squads/{name} | Get squad details
[**ListSquads**](SquadsAPI.md#ListSquads) | **Get** /squads | List squads
[**PatchSquad**](SquadsAPI.md#PatchSquad) | **Patch** /squads/{name} | Patch squad
[**RemoveSquadMember**](SquadsAPI.md#RemoveSquadMember) | **Delete** /squads/{name}/members/{agent} | Remove squad member



## AddSquadMember

> SquadResponse AddSquadMember(ctx, name).Request(request).Namespace(namespace).Execute()

Add squad member



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
	name := "name_example" // string | Squad name
	request := *openapiclient.NewAddSquadMemberRequest() // AddSquadMemberRequest | Member to add
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SquadsAPI.AddSquadMember(context.Background(), name).Request(request).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SquadsAPI.AddSquadMember``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AddSquadMember`: SquadResponse
	fmt.Fprintf(os.Stdout, "Response from `SquadsAPI.AddSquadMember`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Squad name | 

### Other Parameters

Other parameters are passed through a pointer to a apiAddSquadMemberRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **request** | [**AddSquadMemberRequest**](AddSquadMemberRequest.md) | Member to add | 
 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateSquad

> SquadResponse CreateSquad(ctx).Request(request).Execute()

Create squad



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
	request := *openapiclient.NewCreateSquadRequest([]openapiclient.V1alpha1KomputerSquadMember{*openapiclient.NewV1alpha1KomputerSquadMember()}, "Name_example") // CreateSquadRequest | Squad creation request

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SquadsAPI.CreateSquad(context.Background()).Request(request).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SquadsAPI.CreateSquad``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSquad`: SquadResponse
	fmt.Fprintf(os.Stdout, "Response from `SquadsAPI.CreateSquad`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateSquadRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateSquadRequest**](CreateSquadRequest.md) | Squad creation request | 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteSquad

> map[string]string DeleteSquad(ctx, name).Namespace(namespace).Execute()

Delete squad



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
	name := "name_example" // string | Squad name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SquadsAPI.DeleteSquad(context.Background(), name).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SquadsAPI.DeleteSquad``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DeleteSquad`: map[string]string
	fmt.Fprintf(os.Stdout, "Response from `SquadsAPI.DeleteSquad`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Squad name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteSquadRequest struct via the builder pattern


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


## GetSquad

> SquadResponse GetSquad(ctx, name).Namespace(namespace).Execute()

Get squad details



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
	name := "name_example" // string | Squad name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SquadsAPI.GetSquad(context.Background(), name).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SquadsAPI.GetSquad``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetSquad`: SquadResponse
	fmt.Fprintf(os.Stdout, "Response from `SquadsAPI.GetSquad`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Squad name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetSquadRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListSquads

> SquadListResponse ListSquads(ctx).Namespace(namespace).Execute()

List squads



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
	resp, r, err := apiClient.SquadsAPI.ListSquads(context.Background()).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SquadsAPI.ListSquads``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSquads`: SquadListResponse
	fmt.Fprintf(os.Stdout, "Response from `SquadsAPI.ListSquads`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListSquadsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**SquadListResponse**](SquadListResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchSquad

> SquadResponse PatchSquad(ctx, name).Request(request).Namespace(namespace).Execute()

Patch squad



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
	name := "name_example" // string | Squad name
	request := *openapiclient.NewPatchSquadRequest() // PatchSquadRequest | Fields to update
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SquadsAPI.PatchSquad(context.Background(), name).Request(request).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SquadsAPI.PatchSquad``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchSquad`: SquadResponse
	fmt.Fprintf(os.Stdout, "Response from `SquadsAPI.PatchSquad`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Squad name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchSquadRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **request** | [**PatchSquadRequest**](PatchSquadRequest.md) | Fields to update | 
 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RemoveSquadMember

> SquadResponse RemoveSquadMember(ctx, name, agent).Namespace(namespace).Execute()

Remove squad member



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
	name := "name_example" // string | Squad name
	agent := "agent_example" // string | Agent name to remove
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SquadsAPI.RemoveSquadMember(context.Background(), name, agent).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SquadsAPI.RemoveSquadMember``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RemoveSquadMember`: SquadResponse
	fmt.Fprintf(os.Stdout, "Response from `SquadsAPI.RemoveSquadMember`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Squad name | 
**agent** | **string** | Agent name to remove | 

### Other Parameters

Other parameters are passed through a pointer to a apiRemoveSquadMemberRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**SquadResponse**](SquadResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

