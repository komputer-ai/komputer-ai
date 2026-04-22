# V1alpha1StorageSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Size** | Pointer to **string** | Size is the PVC storage size (e.g. \&quot;5Gi\&quot;). +kubebuilder:default&#x3D;\&quot;5Gi\&quot; | [optional] 
**StorageClassName** | Pointer to **string** | StorageClassName is the optional storage class name. +optional | [optional] 

## Methods

### NewV1alpha1StorageSpec

`func NewV1alpha1StorageSpec() *V1alpha1StorageSpec`

NewV1alpha1StorageSpec instantiates a new V1alpha1StorageSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1alpha1StorageSpecWithDefaults

`func NewV1alpha1StorageSpecWithDefaults() *V1alpha1StorageSpec`

NewV1alpha1StorageSpecWithDefaults instantiates a new V1alpha1StorageSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSize

`func (o *V1alpha1StorageSpec) GetSize() string`

GetSize returns the Size field if non-nil, zero value otherwise.

### GetSizeOk

`func (o *V1alpha1StorageSpec) GetSizeOk() (*string, bool)`

GetSizeOk returns a tuple with the Size field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSize

`func (o *V1alpha1StorageSpec) SetSize(v string)`

SetSize sets Size field to given value.

### HasSize

`func (o *V1alpha1StorageSpec) HasSize() bool`

HasSize returns a boolean if a field has been set.

### GetStorageClassName

`func (o *V1alpha1StorageSpec) GetStorageClassName() string`

GetStorageClassName returns the StorageClassName field if non-nil, zero value otherwise.

### GetStorageClassNameOk

`func (o *V1alpha1StorageSpec) GetStorageClassNameOk() (*string, bool)`

GetStorageClassNameOk returns a tuple with the StorageClassName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStorageClassName

`func (o *V1alpha1StorageSpec) SetStorageClassName(v string)`

SetStorageClassName sets StorageClassName field to given value.

### HasStorageClassName

`func (o *V1alpha1StorageSpec) HasStorageClassName() bool`

HasStorageClassName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


