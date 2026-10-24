# ConfigResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**S3** | Pointer to [**S3ConfigResponse**](S3ConfigResponse.md) |  | [optional] 

## Methods

### NewConfigResponse

`func NewConfigResponse() *ConfigResponse`

NewConfigResponse instantiates a new ConfigResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewConfigResponseWithDefaults

`func NewConfigResponseWithDefaults() *ConfigResponse`

NewConfigResponseWithDefaults instantiates a new ConfigResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetS3

`func (o *ConfigResponse) GetS3() S3ConfigResponse`

GetS3 returns the S3 field if non-nil, zero value otherwise.

### GetS3Ok

`func (o *ConfigResponse) GetS3Ok() (*S3ConfigResponse, bool)`

GetS3Ok returns a tuple with the S3 field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetS3

`func (o *ConfigResponse) SetS3(v S3ConfigResponse)`

SetS3 sets S3 field to given value.

### HasS3

`func (o *ConfigResponse) HasS3() bool`

HasS3 returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


