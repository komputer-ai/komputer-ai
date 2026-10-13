# AgentResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**allowed_tools** | **List[str]** | Tools this agent is restricted to (empty &#x3D; defaults) | [optional] 
**budget_exhausted** | **bool** | BudgetExhausted is true when status.totalCostUSD has reached or passed the EFFECTIVE lifetime budget (this agent&#39;s own BudgetUSD, else its template&#39;s default) — new tasks are refused with 409 until it&#39;s raised. Always false when no lifetime budget applies anywhere. Computed by the API so clients don&#39;t have to re-derive the template resolution themselves. | [optional] 
**budget_usd** | **str** | BudgetUSD is this agent&#39;s own SOFT lifetime spending cap (spec.budgetUSD), as a decimal USD string. Empty when unset on the agent itself — note the agent may still be bound by a template default; BudgetExhausted reflects that effective value even when this field is empty. | [optional] 
**completion_time** | **str** |  | [optional] 
**connectors** | **List[str]** | KomputerConnector names attached to this agent | [optional] 
**created_at** | **str** |  | [optional] 
**delete_expires_at** | **str** |  | [optional] 
**delete_ttl** | **str** | absolute lifetime before auto-delete, e.g. \&quot;24h\&quot; | [optional] 
**disallowed_tools** | **List[str]** | Tools removed from this agent | [optional] 
**errors** | **List[str]** | Errors are non-fatal failures that occurred during the request (e.g. CR was patched but live-pod sync failed). The CR change still took effect; the UI can surface these as toasts so the user knows something didn&#39;t fully apply. | [optional] 
**instructions** | **str** | User task (spec.instructions) | [optional] 
**labels** | **Dict[str, str]** |  | [optional] 
**last_activity_at** | **str** | LastActivityAt is when the agent last saw task activity (RFC3339). The idle clock sleepTTL is measured against. | [optional] 
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
**sleep_expires_at** | **str** | SleepExpiresAt / DeleteExpiresAt are when the TTLs will fire (RFC3339). Empty when the matching TTL is unset or its countdown isn&#39;t currently running. | [optional] 
**sleep_ttl** | **str** | idle timeout before auto-sleep, e.g. \&quot;30m\&quot; | [optional] 
**squad** | **bool** | True when this agent is managed by a KomputerSquad | [optional] 
**squad_name** | **str** | Name of the squad managing this agent (when Squad&#x3D;true) | [optional] 
**status** | **str** |  | [optional] 
**storage** | [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) |  | [optional] 
**system_prompt** | **str** | Custom system prompt (spec.systemPrompt) | [optional] 
**task_budget_usd** | **str** | TaskBudgetUSD is this agent&#39;s own HARD per-task spending cap (spec.taskBudgetUSD), as a decimal USD string. Empty when unset on the agent itself (it may still inherit a template default). | [optional] 
**task_expires_at** | **str** |  | [optional] 
**task_started_at** | **str** | TaskStartedAt is when the current (or most recent) task started (RFC3339). TaskExpiresAt is when taskTimeout will cancel it; empty when no task is running. | [optional] 
**task_status** | **str** |  | [optional] 
**task_timeout** | **str** | per-task wall-clock cap, e.g. \&quot;30m\&quot; | [optional] 
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


