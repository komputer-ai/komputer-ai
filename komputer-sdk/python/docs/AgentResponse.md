# AgentResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**connectors** | **List[str]** | KomputerConnector names attached to this agent | [optional] 
**created_at** | **str** |  | [optional] 
**errors** | **List[str]** | Errors are non-fatal failures that occurred during the request (e.g. CR was patched but live-pod sync failed). The CR change still took effect; the UI can surface these as toasts so the user knows something didn&#39;t fully apply. | [optional] 
**instructions** | **str** | User task (spec.instructions) | [optional] 
**last_task_cost_usd** | **str** |  | [optional] 
**last_task_message** | **str** |  | [optional] 
**lifecycle** | **str** |  | [optional] 
**memories** | **List[str]** | KomputerMemory names attached to this agent | [optional] 
**model** | **str** |  | [optional] 
**model_context_window** | **int** |  | [optional] 
**name** | **str** |  | [optional] 
**namespace** | **str** |  | [optional] 
**pod_spec** | [**V1PodSpec**](V1PodSpec.md) |  | [optional] 
**priority** | **int** |  | [optional] 
**queue_position** | **int** |  | [optional] 
**queue_reason** | **str** |  | [optional] 
**secrets** | **List[str]** | Key names from K8s Secrets (not values) | [optional] 
**skills** | **List[str]** | KomputerSkill names attached to this agent | [optional] 
**status** | **str** |  | [optional] 
**storage** | [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) |  | [optional] 
**system_prompt** | **str** | Custom system prompt (spec.systemPrompt) | [optional] 
**task_status** | **str** |  | [optional] 
**total_cost_usd** | **str** |  | [optional] 
**total_tokens** | **int** |  | [optional] 

## Example

```python
from komputer_ai.models.agent_response import AgentResponse

# TODO update the JSON string below
json = "{}"
# create an instance of AgentResponse from a JSON string
agent_response_instance = AgentResponse.from_json(json)
# print the JSON string representation of the object
print(AgentResponse.to_json())

# convert the object into a dict
agent_response_dict = agent_response_instance.to_dict()
# create an instance of AgentResponse from a dict
agent_response_from_dict = AgentResponse.from_dict(agent_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


