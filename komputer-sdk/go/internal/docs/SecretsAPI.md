# \SecretsAPI

All URIs are relative to *http://localhost:8080/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateSecret**](SecretsAPI.md#CreateSecret) | **Post** /secrets | Create managed secret
[**DeleteSecret**](SecretsAPI.md#DeleteSecret) | **Delete** /secrets/{name} | Delete managed secret
[**ListSecrets**](SecretsAPI.md#ListSecrets) | **Get** /secrets | List secrets
[**UpdateSecret**](SecretsAPI.md#UpdateSecret) | **Patch** /secrets/{name} | Update managed secret



## CreateSecret

> SecretResponse CreateSecret(ctx).Request(request).Execute()

Create managed secret



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
	request := *openapiclient.NewCreateSecretRequest(map[string]string{"key": "Inner_example"}, "Name_example") // CreateSecretRequest | Secret creation request

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SecretsAPI.CreateSecret(context.Background()).Request(request).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SecretsAPI.CreateSecret``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSecret`: SecretResponse
	fmt.Fprintf(os.Stdout, "Response from `SecretsAPI.CreateSecret`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateSecretRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **request** | [**CreateSecretRequest**](CreateSecretRequest.md) | Secret creation request | 

### Return type

[**SecretResponse**](SecretResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteSecret

> map[string]string DeleteSecret(ctx, name).Namespace(namespace).Execute()

Delete managed secret



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
	name := "name_example" // string | Secret name
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SecretsAPI.DeleteSecret(context.Background(), name).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SecretsAPI.DeleteSecret``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DeleteSecret`: map[string]string
	fmt.Fprintf(os.Stdout, "Response from `SecretsAPI.DeleteSecret`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Secret name | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteSecretRequest struct via the builder pattern


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


## ListSecrets

> SecretListResponse ListSecrets(ctx).Namespace(namespace).All(all).Execute()

List secrets



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
	all := true // bool | Include all secrets, not just managed ones (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SecretsAPI.ListSecrets(context.Background()).Namespace(namespace).All(all).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SecretsAPI.ListSecrets``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSecrets`: SecretListResponse
	fmt.Fprintf(os.Stdout, "Response from `SecretsAPI.ListSecrets`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListSecretsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **string** | Kubernetes namespace | 
 **all** | **bool** | Include all secrets, not just managed ones | 

### Return type

[**SecretListResponse**](SecretListResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateSecret

> map[string]string UpdateSecret(ctx, name).Request(request).Namespace(namespace).Execute()

Update managed secret



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
	name := "name_example" // string | Secret name
	request := *openapiclient.NewUpdateSecretRequest(map[string]string{"key": "Inner_example"}) // UpdateSecretRequest | Updated secret data
	namespace := "namespace_example" // string | Kubernetes namespace (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SecretsAPI.UpdateSecret(context.Background(), name).Request(request).Namespace(namespace).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SecretsAPI.UpdateSecret``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateSecret`: map[string]string
	fmt.Fprintf(os.Stdout, "Response from `SecretsAPI.UpdateSecret`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** | Secret name | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateSecretRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **request** | [**UpdateSecretRequest**](UpdateSecretRequest.md) | Updated secret data | 
 **namespace** | **string** | Kubernetes namespace | 

### Return type

**map[string]string**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

