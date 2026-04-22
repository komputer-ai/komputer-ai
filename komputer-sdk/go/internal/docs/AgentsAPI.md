# \AgentsAPI

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AgentsNameWsGet**](AgentsAPI.md#AgentsNameWsGet) | **Get** /agents/{name}/ws | Stream agent events (WebSocket)
[**CancelAgentTask**](AgentsAPI.md#CancelAgentTask) | **Post** /agents/{name}/cancel | Cancel agent task
[**CreateAgent**](AgentsAPI.md#CreateAgent) | **Post** /agents | Create agent or send task
[**DeleteAgent**](AgentsAPI.md#DeleteAgent) | **Delete** /agents/{name} | Delete agent
[**GetAgent**](AgentsAPI.md#GetAgent) | **Get** /agents/{name} | Get agent details
[**GetAgentEvents**](AgentsAPI.md#GetAgentEvents) | **Get** /agents/{name}/events | Get agent events
[**ListAgents**](AgentsAPI.md#ListAgents) | **Get** /agents | List agents
[**PatchAgent**](AgentsAPI.md#PatchAgent) | **Patch** /agents/{name} | Patch agent



## AgentsNameWsGet

> AgentsNameWsGet(ctx, name).Execute()

Stream agent events (WebSocket)



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
	name := "name_example" // string | Agent name

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.AgentsAPI.AgentsNameWsGet(context.Background(), name).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AgentsAPI.AgentsNameWsGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Agent name | 

### Other Parameters

Other parameters are passed through a pointer to a apiAgentsNameWsGetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CancelAgentTask

> map[string]string CancelAgentTask(ctx, name).Namespace(namespace).Execute()

Cancel agent task



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
	name := "name_example" // string | Agent name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AgentsAPI.CancelAgentTask(context.Background(), name).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AgentsAPI.CancelAgentTask``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CancelAgentTask`: map[string]string
	fmt.Fprintf(os.Stdout, "Response from `AgentsAPI.CancelAgentTask`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Agent name | 

### Other Parameters

Other parameters are passed through a pointer to a apiCancelAgentTaskRequest struct via the builder pattern


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


## CreateAgent

> AgentResponse CreateAgent(ctx).Request(request).Execute()

Create agent or send task



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
	request := *openapiclient.NewCreateAgentRequest("Instructions_example", "Name_example") // CreateAgentRequest | Agent creation request

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AgentsAPI.CreateAgent(context.Background()).Request(request).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AgentsAPI.CreateAgent``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateAgent`: AgentResponse
	fmt.Fprintf(os.Stdout, "Response from `AgentsAPI.CreateAgent`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateAgentRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateAgentRequest**](CreateAgentRequest.md) | Agent creation request | 

### Return type

[**AgentResponse**](AgentResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteAgent

> map[string]string DeleteAgent(ctx, name).Namespace(namespace).Execute()

Delete agent



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
	name := "name_example" // string | Agent name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AgentsAPI.DeleteAgent(context.Background(), name).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AgentsAPI.DeleteAgent``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DeleteAgent`: map[string]string
	fmt.Fprintf(os.Stdout, "Response from `AgentsAPI.DeleteAgent`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Agent name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteAgentRequest struct via the builder pattern


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


## GetAgent

> AgentResponse GetAgent(ctx, name).Namespace(namespace).Execute()

Get agent details



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
	name := "name_example" // string | Agent name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AgentsAPI.GetAgent(context.Background(), name).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AgentsAPI.GetAgent``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetAgent`: AgentResponse
	fmt.Fprintf(os.Stdout, "Response from `AgentsAPI.GetAgent`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Agent name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetAgentRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**AgentResponse**](AgentResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetAgentEvents

> map[string]interface{} GetAgentEvents(ctx, name).Namespace(namespace).Limit(limit).Execute()

Get agent events



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
	name := "name_example" // string | Agent name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)
	limit := int32(56) // int32 | Max events to return (1-200) (optional) (default to 50)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AgentsAPI.GetAgentEvents(context.Background(), name).Namespace(namespace).Limit(limit).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AgentsAPI.GetAgentEvents``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetAgentEvents`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `AgentsAPI.GetAgentEvents`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Agent name | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetAgentEventsRequest struct via the builder pattern


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


## ListAgents

> AgentListResponse ListAgents(ctx).Namespace(namespace).Execute()

List agents



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
	resp, r, err := apiClient.AgentsAPI.ListAgents(context.Background()).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AgentsAPI.ListAgents``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListAgents`: AgentListResponse
	fmt.Fprintf(os.Stdout, "Response from `AgentsAPI.ListAgents`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListAgentsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**AgentListResponse**](AgentListResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PatchAgent

> AgentResponse PatchAgent(ctx, name).Request(request).Namespace(namespace).Execute()

Patch agent



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
	name := "name_example" // string | Agent name
	request := *openapiclient.NewPatchAgentRequest() // PatchAgentRequest | Fields to update
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AgentsAPI.PatchAgent(context.Background(), name).Request(request).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AgentsAPI.PatchAgent``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PatchAgent`: AgentResponse
	fmt.Fprintf(os.Stdout, "Response from `AgentsAPI.PatchAgent`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Agent name | 

### Other Parameters

Other parameters are passed through a pointer to a apiPatchAgentRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **request** | [**PatchAgentRequest**](PatchAgentRequest.md) | Fields to update | 
 **namespace** | **string** | Kubernetes namespace | 

### Return type

[**AgentResponse**](AgentResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

