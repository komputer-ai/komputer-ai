# CreateConnectorRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AuthSecretKey** | Pointer to **string** |  | [optional] 
**AuthSecretName** | Pointer to **string** |  | [optional] 
**AuthType** | Pointer to **string** | \&quot;token\&quot;, \&quot;oauth\&quot;, or \&quot;header\&quot; | [optional] 
**DisplayName** | Pointer to **string** |  | [optional] 
**HeaderName** | Pointer to **string** | custom header name when authType is \&quot;header\&quot; | [optional] 
**Name** | **string** |  | 
**Namespace** | Pointer to **string** |  | [optional] 
**OauthClientId** | Pointer to **string** | OAuth client ID (stored in secret) | [optional] 
**OauthClientSecret** | Pointer to **string** | OAuth client secret (stored in secret) | [optional] 
**Service** | **string** |  | 
**Type** | Pointer to **string** |  | [optional] 
**Url** | **string** |  | 

## Methods

### NewCreateConnectorRequest

`func NewCreateConnectorRequest(name string, service string, url string, ) *CreateConnectorRequest`

NewCreateConnectorRequest instantiates a new CreateConnectorRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewCreateConnectorRequestWithDefaults

`func NewCreateConnectorRequestWithDefaults() *CreateConnectorRequest`

NewCreateConnectorRequestWithDefaults instantiates a new CreateConnectorRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAuthSecretKey

`func (o *CreateConnectorRequest) GetAuthSecretKey() string`

GetAuthSecretKey returns the AuthSecretKey field if non-nil, zero value otherwise.

### GetAuthSecretKeyOk

`func (o *CreateConnectorRequest) GetAuthSecretKeyOk() (*string, bool)`

GetAuthSecretKeyOk returns a tuple with the AuthSecretKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthSecretKey

`func (o *CreateConnectorRequest) SetAuthSecretKey(v string)`

SetAuthSecretKey sets AuthSecretKey field to given value.

### HasAuthSecretKey

`func (o *CreateConnectorRequest) HasAuthSecretKey() bool`

HasAuthSecretKey returns a boolean if a field has been set.

### GetAuthSecretName

`func (o *CreateConnectorRequest) GetAuthSecretName() string`

GetAuthSecretName returns the AuthSecretName field if non-nil, zero value otherwise.

### GetAuthSecretNameOk

`func (o *CreateConnectorRequest) GetAuthSecretNameOk() (*string, bool)`

GetAuthSecretNameOk returns a tuple with the AuthSecretName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthSecretName

`func (o *CreateConnectorRequest) SetAuthSecretName(v string)`

SetAuthSecretName sets AuthSecretName field to given value.

### HasAuthSecretName

`func (o *CreateConnectorRequest) HasAuthSecretName() bool`

HasAuthSecretName returns a boolean if a field has been set.

### GetAuthType

`func (o *CreateConnectorRequest) GetAuthType() string`

GetAuthType returns the AuthType field if non-nil, zero value otherwise.

### GetAuthTypeOk

`func (o *CreateConnectorRequest) GetAuthTypeOk() (*string, bool)`

GetAuthTypeOk returns a tuple with the AuthType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthType

`func (o *CreateConnectorRequest) SetAuthType(v string)`

SetAuthType sets AuthType field to given value.

### HasAuthType

`func (o *CreateConnectorRequest) HasAuthType() bool`

HasAuthType returns a boolean if a field has been set.

### GetDisplayName

`func (o *CreateConnectorRequest) GetDisplayName() string`

GetDisplayName returns the DisplayName field if non-nil, zero value otherwise.

### GetDisplayNameOk

`func (o *CreateConnectorRequest) GetDisplayNameOk() (*string, bool)`

GetDisplayNameOk returns a tuple with the DisplayName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisplayName

`func (o *CreateConnectorRequest) SetDisplayName(v string)`

SetDisplayName sets DisplayName field to given value.

### HasDisplayName

`func (o *CreateConnectorRequest) HasDisplayName() bool`

HasDisplayName returns a boolean if a field has been set.

### GetHeaderName

`func (o *CreateConnectorRequest) GetHeaderName() string`

GetHeaderName returns the HeaderName field if non-nil, zero value otherwise.

### GetHeaderNameOk

`func (o *CreateConnectorRequest) GetHeaderNameOk() (*string, bool)`

GetHeaderNameOk returns a tuple with the HeaderName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHeaderName

`func (o *CreateConnectorRequest) SetHeaderName(v string)`

SetHeaderName sets HeaderName field to given value.

### HasHeaderName

`func (o *CreateConnectorRequest) HasHeaderName() bool`

HasHeaderName returns a boolean if a field has been set.

### GetName

`func (o *CreateConnectorRequest) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *CreateConnectorRequest) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *CreateConnectorRequest) SetName(v string)`

SetName sets Name field to given value.


### GetNamespace

`func (o *CreateConnectorRequest) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *CreateConnectorRequest) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *CreateConnectorRequest) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *CreateConnectorRequest) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetOauthClientId

`func (o *CreateConnectorRequest) GetOauthClientId() string`

GetOauthClientId returns the OauthClientId field if non-nil, zero value otherwise.

### GetOauthClientIdOk

`func (o *CreateConnectorRequest) GetOauthClientIdOk() (*string, bool)`

GetOauthClientIdOk returns a tuple with the OauthClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOauthClientId

`func (o *CreateConnectorRequest) SetOauthClientId(v string)`

SetOauthClientId sets OauthClientId field to given value.

### HasOauthClientId

`func (o *CreateConnectorRequest) HasOauthClientId() bool`

HasOauthClientId returns a boolean if a field has been set.

### GetOauthClientSecret

`func (o *CreateConnectorRequest) GetOauthClientSecret() string`

GetOauthClientSecret returns the OauthClientSecret field if non-nil, zero value otherwise.

### GetOauthClientSecretOk

`func (o *CreateConnectorRequest) GetOauthClientSecretOk() (*string, bool)`

GetOauthClientSecretOk returns a tuple with the OauthClientSecret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOauthClientSecret

`func (o *CreateConnectorRequest) SetOauthClientSecret(v string)`

SetOauthClientSecret sets OauthClientSecret field to given value.

### HasOauthClientSecret

`func (o *CreateConnectorRequest) HasOauthClientSecret() bool`

HasOauthClientSecret returns a boolean if a field has been set.

### GetService

`func (o *CreateConnectorRequest) GetService() string`

GetService returns the Service field if non-nil, zero value otherwise.

### GetServiceOk

`func (o *CreateConnectorRequest) GetServiceOk() (*string, bool)`

GetServiceOk returns a tuple with the Service field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetService

`func (o *CreateConnectorRequest) SetService(v string)`

SetService sets Service field to given value.


### GetType

`func (o *CreateConnectorRequest) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *CreateConnectorRequest) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *CreateConnectorRequest) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *CreateConnectorRequest) HasType() bool`

HasType returns a boolean if a field has been set.

### GetUrl

`func (o *CreateConnectorRequest) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *CreateConnectorRequest) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *CreateConnectorRequest) SetUrl(v string)`

SetUrl sets Url field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


