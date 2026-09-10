# CreateAgentRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**allowed_tools** | **List[str]** | AllowedTools restricts the agent to exactly these tools. REPLACES the default built-in set rather than extending it, and connector tools are not auto-added. Supports wildcards, e.g. \&quot;mcp__figma__*\&quot;. Empty keeps default behavior. | [optional] 
**connectors** | **List[str]** | optional KomputerConnector names to attach | [optional] 
**delete_ttl** | **str** | DeleteTTL deletes the agent this long after creation, as a Go duration string (e.g. \&quot;24h\&quot;). Absolute — it does not reset on wake. Empty means never auto-delete. | [optional] 
**disallowed_tools** | **List[str]** | DisallowedTools removes these tools; everything else stays available. Takes precedence over AllowedTools. Supports wildcards. | [optional] 
**instructions** | **str** |  | 
**labels** | **Dict[str, str]** | Labels are user-defined key&#x3D;value labels passed through to the agent CR. Reserved-prefix keys (komputer.ai/*) are rejected except for \&quot;komputer.ai/personal-agent\&quot; which is allow-listed. | [optional] 
**lifecycle** | **str** | \&quot;\&quot;, \&quot;Sleep\&quot;, or \&quot;AutoDelete\&quot; | [optional] 
**memories** | **List[str]** | optional KomputerMemory names to attach | [optional] 
**model** | **str** |  | [optional] 
**name** | **str** |  | 
**namespace** | **str** | optional, defaults to server default | [optional] 
**office_manager** | **str** | set by manager MCP tool | [optional] 
**pod_spec** | [**V1PodSpec**](V1PodSpec.md) |  | [optional] 
**priority** | **int** | queue priority; higher &#x3D; admitted first | [optional] 
**role** | **str** | \&quot;manager\&quot; or \&quot;\&quot; (default manager) | [optional] 
**secret_refs** | **List[str]** | names of existing K8s Secrets to attach | [optional] 
**skills** | **List[str]** | optional KomputerSkill names to attach | [optional] 
**sleep_ttl** | **str** | SleepTTL puts the agent to sleep after this long with no activity, as a Go duration string (e.g. \&quot;30m\&quot;, \&quot;2h\&quot;). Empty means never auto-sleep. | [optional] 
**storage** | [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) |  | [optional] 
**system_prompt** | **str** | optional custom system prompt | [optional] 
**task_timeout** | **str** | TaskTimeout cancels a running task once it has run this long, as a Go duration string (e.g. \&quot;30m\&quot;). A hard wall-clock cap per task — steering does not extend it. Empty means tasks run without a time limit. | [optional] 
**template_ref** | **str** |  | [optional] 

## Example

```python
from komputer_ai.models.create_agent_request import CreateAgentRequest

# TODO update the JSON string below
json = "{}"
# create an instance of CreateAgentRequest from a JSON string
create_agent_request_instance = CreateAgentRequest.from_json(json)
# print the JSON string representation of the object
print(CreateAgentRequest.to_json())

# convert the object into a dict
create_agent_request_dict = create_agent_request_instance.to_dict()
# create an instance of CreateAgentRequest from a dict
create_agent_request_from_dict = CreateAgentRequest.from_dict(create_agent_request_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


