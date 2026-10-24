# V1alpha1StorageSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Ephemeral** | Pointer to **bool** | Ephemeral, when true, runs the agent&#39;s workspace as an emptyDir instead of a PVC: no PersistentVolumeClaim is created. The workspace lives exactly as long as the pod — files, installed packages and the Claude session are permanently lost whenever the pod is deleted, evicted, rescheduled or restarted. Sleeping (Lifecycle&#x3D;Sleep, sleepTTL, sleep-now) deletes the pod, so without the S3 session store the API rejects those combinations and the operator never sleeps an ephemeral agent. With the store, the operator sleeps it once the latest task&#39;s session is saved, and the wake restores the session into a fresh, empty workspace.  Unset (default) or false keeps today&#39;s behavior: every agent gets a PVC. Set it on a KomputerAgentTemplate to make ephemeral the default for agents using it, or on an individual agent to override its template either way. Immutable after agent creation. An agent that already has a PVC keeps it even if later marked ephemeral — the operator never deletes an existing PVC. +optional | [optional] 
**Size** | Pointer to **string** | Size is the PVC storage size (e.g. \&quot;5Gi\&quot;). When Ephemeral is true, this is instead used as the emptyDir&#39;s SizeLimit. +kubebuilder:default&#x3D;\&quot;5Gi\&quot; | [optional] 
**StorageClassName** | Pointer to **string** | StorageClassName is the optional storage class name. Ignored when Ephemeral is true. +optional | [optional] 

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

### GetEphemeral

`func (o *V1alpha1StorageSpec) GetEphemeral() bool`

GetEphemeral returns the Ephemeral field if non-nil, zero value otherwise.

### GetEphemeralOk

`func (o *V1alpha1StorageSpec) GetEphemeralOk() (*bool, bool)`

GetEphemeralOk returns a tuple with the Ephemeral field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEphemeral

`func (o *V1alpha1StorageSpec) SetEphemeral(v bool)`

SetEphemeral sets Ephemeral field to given value.

### HasEphemeral

`func (o *V1alpha1StorageSpec) HasEphemeral() bool`

HasEphemeral returns a boolean if a field has been set.

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


