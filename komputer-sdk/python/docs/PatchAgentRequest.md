# PatchAgentRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**connectors** | **List[str]** | connector names to attach | [optional] 
**instructions** | **str** |  | [optional] 
**lifecycle** | **str** |  | [optional] 
**memories** | **List[str]** | memory names to attach | [optional] 
**model** | **str** |  | [optional] 
**pod_spec** | [**V1PodSpec**](V1PodSpec.md) |  | [optional] 
**secret_refs** | **List[str]** | full replacement list of K8s secret names | [optional] 
**skills** | **List[str]** | skill names to attach | [optional] 
**storage** | [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) |  | [optional] 
**system_prompt** | **str** | custom system prompt | [optional] 
**template_ref** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.patch_agent_request import PatchAgentRequest

# TODO update the JSON string below
json = "{}"
# create an instance of PatchAgentRequest from a JSON string
patch_agent_request_instance = PatchAgentRequest.from_json(json)
# print the JSON string representation of the object
print(PatchAgentRequest.to_json())

# convert the object into a dict
patch_agent_request_dict = patch_agent_request_instance.to_dict()
# create an instance of PatchAgentRequest from a dict
patch_agent_request_from_dict = PatchAgentRequest.from_dict(patch_agent_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


