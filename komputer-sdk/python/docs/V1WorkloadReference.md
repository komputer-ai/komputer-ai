# V1WorkloadReference


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name defines the name of the Workload object this Pod belongs to. Workload must be in the same namespace as the Pod. If it doesn&#39;t match any existing Workload, the Pod will remain unschedulable until a Workload object is created and observed by the kube-scheduler. It must be a DNS subdo  +required | [optional] 
**pod_group** | **str** | PodGroup is the name of the PodGroup within the Workload that this Pod belongs to. If it doesn&#39;t match any existing PodGroup within the Workload, the Pod will remain unschedulable until the Workload object is recreated and observed by the kube-scheduler. It must be a DNS label.  +required | [optional] 
**pod_group_replica_key** | **str** | PodGroupReplicaKey specifies the replica key of the PodGroup to which this Pod belongs. It is used to distinguish pods belonging to different replicas of the same pod group. The pod group policy is applied separately to each replica. When set, it must be a DNS label.  +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_workload_reference import V1WorkloadReference

# TODO update the JSON string below
json = "{}"
# create an instance of V1WorkloadReference from a JSON string
v1_workload_reference_instance = V1WorkloadReference.from_json(json)
# print the JSON string representation of the object
print(V1WorkloadReference.to_json())

# convert the object into a dict
v1_workload_reference_dict = v1_workload_reference_instance.to_dict()
# create an instance of V1WorkloadReference from a dict
v1_workload_reference_from_dict = V1WorkloadReference.from_dict(v1_workload_reference_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


