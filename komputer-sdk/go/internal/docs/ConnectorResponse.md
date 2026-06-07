# ConnectorResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AgentNames** | Pointer to **[]string** |  | [optional] 
**AttachedAgents** | Pointer to **int32** |  | [optional] 
**AuthSecretKey** | Pointer to **string** |  | [optional] 
**AuthSecretName** | Pointer to **string** |  | [optional] 
**AuthType** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **string** |  | [optional] 
**DisplayName** | Pointer to **string** |  | [optional] 
**HeaderName** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Namespace** | Pointer to **string** |  | [optional] 
**OauthStatus** | Pointer to **string** | \&quot;pending\&quot;, \&quot;connected\&quot;, \&quot;\&quot; | [optional] 
**Service** | Pointer to **string** |  | [optional] 
**Type** | Pointer to **string** |  | [optional] 
**Url** | Pointer to **string** |  | [optional] 

## Methods

### NewConnectorResponse

`func NewConnectorResponse() *ConnectorResponse`

NewConnectorResponse instantiates a new ConnectorResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewConnectorResponseWithDefaults

`func NewConnectorResponseWithDefaults() *ConnectorResponse`

NewConnectorResponseWithDefaults instantiates a new ConnectorResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAgentNames

`func (o *ConnectorResponse) GetAgentNames() []string`

GetAgentNames returns the AgentNames field if non-nil, zero value otherwise.

### GetAgentNamesOk

`func (o *ConnectorResponse) GetAgentNamesOk() (*[]string, bool)`

GetAgentNamesOk returns a tuple with the AgentNames field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAgentNames

`func (o *ConnectorResponse) SetAgentNames(v []string)`

SetAgentNames sets AgentNames field to given value.

### HasAgentNames

`func (o *ConnectorResponse) HasAgentNames() bool`

HasAgentNames returns a boolean if a field has been set.

### GetAttachedAgents

`func (o *ConnectorResponse) GetAttachedAgents() int32`

GetAttachedAgents returns the AttachedAgents field if non-nil, zero value otherwise.

### GetAttachedAgentsOk

`func (o *ConnectorResponse) GetAttachedAgentsOk() (*int32, bool)`

GetAttachedAgentsOk returns a tuple with the AttachedAgents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAttachedAgents

`func (o *ConnectorResponse) SetAttachedAgents(v int32)`

SetAttachedAgents sets AttachedAgents field to given value.

### HasAttachedAgents

`func (o *ConnectorResponse) HasAttachedAgents() bool`

HasAttachedAgents returns a boolean if a field has been set.

### GetAuthSecretKey

`func (o *ConnectorResponse) GetAuthSecretKey() string`

GetAuthSecretKey returns the AuthSecretKey field if non-nil, zero value otherwise.

### GetAuthSecretKeyOk

`func (o *ConnectorResponse) GetAuthSecretKeyOk() (*string, bool)`

GetAuthSecretKeyOk returns a tuple with the AuthSecretKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthSecretKey

`func (o *ConnectorResponse) SetAuthSecretKey(v string)`

SetAuthSecretKey sets AuthSecretKey field to given value.

### HasAuthSecretKey

`func (o *ConnectorResponse) HasAuthSecretKey() bool`

HasAuthSecretKey returns a boolean if a field has been set.

### GetAuthSecretName

`func (o *ConnectorResponse) GetAuthSecretName() string`

GetAuthSecretName returns the AuthSecretName field if non-nil, zero value otherwise.

### GetAuthSecretNameOk

`func (o *ConnectorResponse) GetAuthSecretNameOk() (*string, bool)`

GetAuthSecretNameOk returns a tuple with the AuthSecretName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthSecretName

`func (o *ConnectorResponse) SetAuthSecretName(v string)`

SetAuthSecretName sets AuthSecretName field to given value.

### HasAuthSecretName

`func (o *ConnectorResponse) HasAuthSecretName() bool`

HasAuthSecretName returns a boolean if a field has been set.

### GetAuthType

`func (o *ConnectorResponse) GetAuthType() string`

GetAuthType returns the AuthType field if non-nil, zero value otherwise.

### GetAuthTypeOk

`func (o *ConnectorResponse) GetAuthTypeOk() (*string, bool)`

GetAuthTypeOk returns a tuple with the AuthType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAuthType

`func (o *ConnectorResponse) SetAuthType(v string)`

SetAuthType sets AuthType field to given value.

### HasAuthType

`func (o *ConnectorResponse) HasAuthType() bool`

HasAuthType returns a boolean if a field has been set.

### GetCreatedAt

`func (o *ConnectorResponse) GetCreatedAt() string`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *ConnectorResponse) GetCreatedAtOk() (*string, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *ConnectorResponse) SetCreatedAt(v string)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *ConnectorResponse) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetDisplayName

`func (o *ConnectorResponse) GetDisplayName() string`

GetDisplayName returns the DisplayName field if non-nil, zero value otherwise.

### GetDisplayNameOk

`func (o *ConnectorResponse) GetDisplayNameOk() (*string, bool)`

GetDisplayNameOk returns a tuple with the DisplayName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisplayName

`func (o *ConnectorResponse) SetDisplayName(v string)`

SetDisplayName sets DisplayName field to given value.

### HasDisplayName

`func (o *ConnectorResponse) HasDisplayName() bool`

HasDisplayName returns a boolean if a field has been set.

### GetHeaderName

`func (o *ConnectorResponse) GetHeaderName() string`

GetHeaderName returns the HeaderName field if non-nil, zero value otherwise.

### GetHeaderNameOk

`func (o *ConnectorResponse) GetHeaderNameOk() (*string, bool)`

GetHeaderNameOk returns a tuple with the HeaderName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHeaderName

`func (o *ConnectorResponse) SetHeaderName(v string)`

SetHeaderName sets HeaderName field to given value.

### HasHeaderName

`func (o *ConnectorResponse) HasHeaderName() bool`

HasHeaderName returns a boolean if a field has been set.

### GetName

`func (o *ConnectorResponse) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ConnectorResponse) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ConnectorResponse) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *ConnectorResponse) HasName() bool`

HasName returns a boolean if a field has been set.

### GetNamespace

`func (o *ConnectorResponse) GetNamespace() string`

GetNamespace returns the Namespace field if non-nil, zero value otherwise.

### GetNamespaceOk

`func (o *ConnectorResponse) GetNamespaceOk() (*string, bool)`

GetNamespaceOk returns a tuple with the Namespace field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNamespace

`func (o *ConnectorResponse) SetNamespace(v string)`

SetNamespace sets Namespace field to given value.

### HasNamespace

`func (o *ConnectorResponse) HasNamespace() bool`

HasNamespace returns a boolean if a field has been set.

### GetOauthStatus

`func (o *ConnectorResponse) GetOauthStatus() string`

GetOauthStatus returns the OauthStatus field if non-nil, zero value otherwise.

### GetOauthStatusOk

`func (o *ConnectorResponse) GetOauthStatusOk() (*string, bool)`

GetOauthStatusOk returns a tuple with the OauthStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOauthStatus

`func (o *ConnectorResponse) SetOauthStatus(v string)`

SetOauthStatus sets OauthStatus field to given value.

### HasOauthStatus

`func (o *ConnectorResponse) HasOauthStatus() bool`

HasOauthStatus returns a boolean if a field has been set.

### GetService

`func (o *ConnectorResponse) GetService() string`

GetService returns the Service field if non-nil, zero value otherwise.

### GetServiceOk

`func (o *ConnectorResponse) GetServiceOk() (*string, bool)`

GetServiceOk returns a tuple with the Service field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetService

`func (o *ConnectorResponse) SetService(v string)`

SetService sets Service field to given value.

### HasService

`func (o *ConnectorResponse) HasService() bool`

HasService returns a boolean if a field has been set.

### GetType

`func (o *ConnectorResponse) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *ConnectorResponse) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *ConnectorResponse) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *ConnectorResponse) HasType() bool`

HasType returns a boolean if a field has been set.

### GetUrl

`func (o *ConnectorResponse) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *ConnectorResponse) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *ConnectorResponse) SetUrl(v string)`

SetUrl sets Url field to given value.

### HasUrl

`func (o *ConnectorResponse) HasUrl() bool`

HasUrl returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


