# K8sIoApiCoreV1ResourceClaim


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name must match the name of one entry in pod.spec.resourceClaims of the Pod where this field is used. It makes that resource available inside a container. | [optional] 
**request** | **str** | Request is the name chosen for a request in the referenced claim. If empty, everything from the claim is made available, otherwise only the result of this request.  +optional | [optional] 

## Example

```python
from komputer_ai.models.k8s_io_api_core_v1_resource_claim import K8sIoApiCoreV1ResourceClaim

# TODO update the JSON string below
json = "{}"
# create an instance of K8sIoApiCoreV1ResourceClaim from a JSON string
k8s_io_api_core_v1_resource_claim_instance = K8sIoApiCoreV1ResourceClaim.from_json(json)
# print the JSON string representation of the object
print(K8sIoApiCoreV1ResourceClaim.to_json())

# convert the object into a dict
k8s_io_api_core_v1_resource_claim_dict = k8s_io_api_core_v1_resource_claim_instance.to_dict()
# create an instance of K8sIoApiCoreV1ResourceClaim from a dict
k8s_io_api_core_v1_resource_claim_from_dict = K8sIoApiCoreV1ResourceClaim.from_dict(k8s_io_api_core_v1_resource_claim_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


