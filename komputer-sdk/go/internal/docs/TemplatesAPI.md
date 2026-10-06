# \TemplatesAPI

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ListTemplates**](TemplatesAPI.md#ListTemplates) | **Get** /templates | List agent templates
[**NamespacesGet**](TemplatesAPI.md#NamespacesGet) | **Get** /namespaces | List namespaces



## ListTemplates

> map[string]interface{} ListTemplates(ctx).Namespace(namespace).Execute()

List agent templates



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
	resp, r, err := apiClient.TemplatesAPI.ListTemplates(context.Background()).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `TemplatesAPI.ListTemplates``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListTemplates`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `TemplatesAPI.ListTemplates`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListTemplatesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **string** | Kubernetes namespace | 

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


## NamespacesGet

> map[string]interface{} NamespacesGet(ctx).Execute()

List namespaces



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

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.TemplatesAPI.NamespacesGet(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `TemplatesAPI.NamespacesGet``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `NamespacesGet`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `TemplatesAPI.NamespacesGet`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiNamespacesGetRequest struct via the builder pattern


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

