# K8sIoApiCoreV1ResourceClaim

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container. | [optional] 
**Request** | Pointer to **string** | Request is the name chosen for a request in the referenced claim. If empty, everything from the claim is made available, otherwise only the result of this request.  +optional | [optional] 

## Methods

### NewK8sIoApiCoreV1ResourceClaim

`func NewK8sIoApiCoreV1ResourceClaim() *K8sIoApiCoreV1ResourceClaim`

NewK8sIoApiCoreV1ResourceClaim instantiates a new K8sIoApiCoreV1ResourceClaim object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewK8sIoApiCoreV1ResourceClaimWithDefaults

`func NewK8sIoApiCoreV1ResourceClaimWithDefaults() *K8sIoApiCoreV1ResourceClaim`

NewK8sIoApiCoreV1ResourceClaimWithDefaults instantiates a new K8sIoApiCoreV1ResourceClaim object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *K8sIoApiCoreV1ResourceClaim) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *K8sIoApiCoreV1ResourceClaim) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *K8sIoApiCoreV1ResourceClaim) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *K8sIoApiCoreV1ResourceClaim) HasName() bool`

HasName returns a boolean if a field has been set.

### GetRequest

`func (o *K8sIoApiCoreV1ResourceClaim) GetRequest() string`

GetRequest returns the Request field if non-nil, zero value otherwise.

### GetRequestOk

`func (o *K8sIoApiCoreV1ResourceClaim) GetRequestOk() (*string, bool)`

GetRequestOk returns a tuple with the Request field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequest

`func (o *K8sIoApiCoreV1ResourceClaim) SetRequest(v string)`

SetRequest sets Request field to given value.

### HasRequest

`func (o *K8sIoApiCoreV1ResourceClaim) HasRequest() bool`

HasRequest returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


