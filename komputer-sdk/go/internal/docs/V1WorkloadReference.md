# V1WorkloadReference

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** | Name defines the name of the Workload object this Pod belongs to. Workload must be in the same namespace as the Pod. If it doesn&#39;t match any existing Workload, the Pod will remain unschedulable until a Workload object is created and observed by the kube-scheduler. It must be a DNS subdo  +required | [optional] 
**PodGroup** | Pointer to **string** | PodGroup is the name of the PodGroup within the Workload that this Pod belongs to. If it doesn&#39;t match any existing PodGroup within the Workload, the Pod will remain unschedulable until the Workload object is recreated and observed by the kube-scheduler. It must be a DNS label.  +required | [optional] 
**PodGroupReplicaKey** | Pointer to **string** | PodGroupReplicaKey specifies the replica key of the PodGroup to which this Pod belongs. It is used to distinguish pods belonging to different replicas of the same pod group. The pod group policy is applied separately to each replica. When set, it must be a DNS label.  +optional | [optional] 

## Methods

### NewV1WorkloadReference

`func NewV1WorkloadReference() *V1WorkloadReference`

NewV1WorkloadReference instantiates a new V1WorkloadReference object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1WorkloadReferenceWithDefaults

`func NewV1WorkloadReferenceWithDefaults() *V1WorkloadReference`

NewV1WorkloadReferenceWithDefaults instantiates a new V1WorkloadReference object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *V1WorkloadReference) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *V1WorkloadReference) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *V1WorkloadReference) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *V1WorkloadReference) HasName() bool`

HasName returns a boolean if a field has been set.

### GetPodGroup

`func (o *V1WorkloadReference) GetPodGroup() string`

GetPodGroup returns the PodGroup field if non-nil, zero value otherwise.

### GetPodGroupOk

`func (o *V1WorkloadReference) GetPodGroupOk() (*string, bool)`

GetPodGroupOk returns a tuple with the PodGroup field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodGroup

`func (o *V1WorkloadReference) SetPodGroup(v string)`

SetPodGroup sets PodGroup field to given value.

### HasPodGroup

`func (o *V1WorkloadReference) HasPodGroup() bool`

HasPodGroup returns a boolean if a field has been set.

### GetPodGroupReplicaKey

`func (o *V1WorkloadReference) GetPodGroupReplicaKey() string`

GetPodGroupReplicaKey returns the PodGroupReplicaKey field if non-nil, zero value otherwise.

### GetPodGroupReplicaKeyOk

`func (o *V1WorkloadReference) GetPodGroupReplicaKeyOk() (*string, bool)`

GetPodGroupReplicaKeyOk returns a tuple with the PodGroupReplicaKey field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodGroupReplicaKey

`func (o *V1WorkloadReference) SetPodGroupReplicaKey(v string)`

SetPodGroupReplicaKey sets PodGroupReplicaKey field to given value.

### HasPodGroupReplicaKey

`func (o *V1WorkloadReference) HasPodGroupReplicaKey() bool`

HasPodGroupReplicaKey returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


