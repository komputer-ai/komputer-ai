# V1alpha1KomputerAgentSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AllowedTools** | Pointer to **[]string** | AllowedTools restricts the agent to exactly these tools. When empty, the default built-in tool set is used and all tools from attached connectors are permitted.  Setting this REPLACES the default set rather than extending it, so an agent given only [\&quot;Read\&quot;] loses Bash, Write, Edit and the rest. Connector tools are not auto-added either — list them explicitly, e.g. \&quot;mcp__figma__*\&quot; for a whole connector or \&quot;mcp__figma__get_design_context\&quot; for a single tool. +optional | [optional] 
**ArchiveTTL** | Pointer to [**V1Duration**](V1Duration.md) | ArchiveTTL archives the agent (pod and PVC deleted, CR kept as a record) once it has been idle for this long. Idle is measured exactly like SleepTTL — from Status.LastActivityAt, reset by any task event or wake, never while a task is in progress, and never on an agent that has never run a task — but it also runs while the agent is Sleeping, so SleepTTL &lt; ArchiveTTL sleeps first, then archives. With the S3 session store it waits until the latest task is saved; without it the archived agent can&#39;t run again. Not applied to squad members. When set, Lifecycle&#x3D;Archive waits for it instead of archiving right after the task. Unset (default) means the agent never auto-archives. Overrides the template&#39;s archiveTTL when set. +optional | [optional] 
**BudgetUSD** | Pointer to **string** | BudgetUSD is a SOFT lifetime spending cap across every task this agent ever runs, as a decimal USD string (e.g. \&quot;10.00\&quot;). Measured against Status.TotalCostUSD: once spent reaches or exceeds this, the API refuses to start any NEW task for the agent. A task already in progress when the cap is crossed is always allowed to finish, even if it overshoots — the cap only gates starting the next one. Raising (or clearing) BudgetUSD unblocks the agent again.  A string rather than a float to match how the rest of the CRD represents USD (see Status.TotalCostUSD) and avoid controller-gen&#39;s allowDangerousTypes for floats. Unset (default) means no lifetime cap. Overrides the template&#39;s budgetUSD when set. +kubebuilder:validation:Pattern&#x3D;&#x60;^[0-9]+(\\.[0-9]+)?$&#x60; +optional | [optional] 
**Connectors** | Pointer to **[]string** | Connectors is a list of KomputerConnector names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**DeleteTTL** | Pointer to [**V1Duration**](V1Duration.md) | DeleteTTL deletes the entire agent (pod + PVC) once this long has elapsed since metadata.creationTimestamp. This is an absolute lifetime cap: unlike SleepTTL it does not reset on wake and applies in every phase, including Sleeping. Unset (default) means the agent never auto-deletes. Overrides the template&#39;s deleteTTL when set. +optional | [optional] 
**DisallowedTools** | Pointer to **[]string** | DisallowedTools removes these tools from the agent. Purely subtractive: the default built-ins and all connector tools remain available except what is named here. Takes precedence over AllowedTools. Supports wildcards, e.g. \&quot;mcp__figma__*\&quot;. +optional | [optional] 
**Effort** | Pointer to **string** | Effort controls the Claude Agent SDK&#39;s reasoning effort: \&quot;low\&quot;, \&quot;medium\&quot;, \&quot;high\&quot;, \&quot;xhigh\&quot;, or \&quot;max\&quot;. Empty (default) leaves it unset, so the CLI uses the model&#39;s own default effort (\&quot;auto\&quot;). +kubebuilder:validation:Enum&#x3D;\&quot;\&quot;;low;medium;high;xhigh;max +optional | [optional] 
**Instructions** | Pointer to **string** | Instructions is the user&#39;s task for the Claude agent. | [optional] 
**InternalSystemPrompt** | Pointer to **string** | InternalSystemPrompt is the built-in system prompt set by the API (role prompt + memories). +optional | [optional] 
**Labels** | Pointer to **map[string]string** | Labels are user-defined key&#x3D;value labels attached to this agent and propagated to all child resources (Pod, PVC, ConfigMap, Service). Keys starting with \&quot;komputer.ai/\&quot; are reserved for system labels and should not be set directly through the API. +optional | [optional] 
**Lifecycle** | Pointer to [**V1alpha1AgentLifecycle**](V1alpha1AgentLifecycle.md) | Lifecycle controls what happens after task completion. Empty (default) keeps the pod running, \&quot;Sleep\&quot; deletes the pod but keeps the PVC, \&quot;AutoDelete\&quot; deletes the entire agent after task completion, and \&quot;Archive\&quot; deletes the pod and PVC but keeps the agent as a read-only record (Phase&#x3D;Archived). +kubebuilder:validation:Enum&#x3D;\&quot;\&quot;;Sleep;AutoDelete;Archive +optional | [optional] 
**Memories** | Pointer to **[]string** | Memories is a list of KomputerMemory names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**Model** | Pointer to **string** | Model is the Claude model to use. +kubebuilder:default&#x3D;\&quot;claude-opus-5-5\&quot; | [optional] 
**OfficeManager** | Pointer to **string** | OfficeManager is the name of the manager agent that created this sub-agent. When set, the operator creates/joins a KomputerOffice for the group. +optional | [optional] 
**PodSpec** | Pointer to [**V1PodSpec**](V1PodSpec.md) | PodSpec, when set, overrides the template&#39;s PodSpec for this agent. Container fields are merged by name; non-zero fields from this PodSpec override the template&#39;s container fields. Takes effect on next pod start (existing pods are not mutated). +optional | [optional] 
**Priority** | Pointer to **int32** | Priority controls admission order when the template&#39;s maxConcurrentAgents limit is reached. Higher number &#x3D; admitted first (matches K8s PodPriority). Ties broken by creationTimestamp (older first). Defaults to 0. +kubebuilder:default&#x3D;0 +optional | [optional] 
**Role** | Pointer to **string** | Role is \&quot;manager\&quot; or \&quot;worker\&quot;. Managers get orchestration tools. Role is \&quot;manager\&quot; or \&quot;worker\&quot;. Defaults to \&quot;manager\&quot; for top-level agents. Sub-agents created by managers are explicitly set to \&quot;worker\&quot;. +kubebuilder:default&#x3D;\&quot;manager\&quot; +kubebuilder:validation:Enum&#x3D;worker;manager +optional | [optional] 
**Secrets** | Pointer to **[]string** | Secrets is a list of K8s Secret names containing agent-specific secrets. Each key in each secret is injected as an env var into the agent pod. +optional | [optional] 
**Skills** | Pointer to **[]string** | Skills is a list of KomputerSkill names to attach to this agent. Names can be \&quot;name\&quot; (same namespace) or \&quot;namespace/name\&quot; (cross-namespace). +optional | [optional] 
**SleepTTL** | Pointer to [**V1Duration**](V1Duration.md) | SleepTTL puts the agent to sleep (pod deleted, PVC preserved) once it has been idle for this long. Idle is measured from Status.LastActivityAt, so the clock only starts once a task has actually started, and any later task event or wake resets it. Never fires while a task is in progress, and an agent that has never run a task is never auto-slept (use DeleteTTL to reclaim those). Unset (default) means the agent never auto-sleeps. Overrides the template&#39;s sleepTTL when set. +optional | [optional] 
**Storage** | Pointer to [**V1alpha1StorageSpec**](V1alpha1StorageSpec.md) | Storage, when set, overrides the template&#39;s storage settings for this agent. Existing PVCs are expanded in place when the storage class supports it. +optional | [optional] 
**SystemPrompt** | Pointer to **string** | SystemPrompt is a custom system prompt provided by the user, appended to the internal prompt. +optional | [optional] 
**TaskBudgetUSD** | Pointer to **string** | TaskBudgetUSD is a HARD per-task spending cap, as a decimal USD string (e.g. \&quot;2.50\&quot;). Passed to the Claude Agent SDK as max_budget_usd for each task the agent runs: the SDK itself stops the task the instant this is exceeded, wherever it is mid-task, returning an error_max_budget_usd result. Unlike BudgetUSD this is enforced inside the running task rather than by refusing to start one.  Unset (default) means tasks run without a per-task cap. Overrides the template&#39;s taskBudgetUSD when set. +kubebuilder:validation:Pattern&#x3D;&#x60;^[0-9]+(\\.[0-9]+)?$&#x60; +optional | [optional] 
**TaskTimeout** | Pointer to [**V1Duration**](V1Duration.md) | TaskTimeout cancels the agent&#39;s running task once it has been running for this long. The clock starts when a task starts (Status.TaskStartedAt) and never resets — steering a task does not extend it — so this is a hard wall-clock cap on a single task, not an idle timeout.  Only the task is cancelled; the agent itself stays alive and any configured Lifecycle (Sleep / AutoDelete) then applies as it would after any other task end. Unset (default) means tasks run without a time limit. Overrides the template&#39;s taskTimeout when set. +optional | [optional] 
**TemplateRef** | Pointer to **string** | TemplateRef is the name of the KomputerAgentTemplate to use. +kubebuilder:default&#x3D;\&quot;default\&quot; | [optional] 

## Methods

### NewV1alpha1KomputerAgentSpec

`func NewV1alpha1KomputerAgentSpec() *V1alpha1KomputerAgentSpec`

NewV1alpha1KomputerAgentSpec instantiates a new V1alpha1KomputerAgentSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewV1alpha1KomputerAgentSpecWithDefaults

`func NewV1alpha1KomputerAgentSpecWithDefaults() *V1alpha1KomputerAgentSpec`

NewV1alpha1KomputerAgentSpecWithDefaults instantiates a new V1alpha1KomputerAgentSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAllowedTools

`func (o *V1alpha1KomputerAgentSpec) GetAllowedTools() []string`

GetAllowedTools returns the AllowedTools field if non-nil, zero value otherwise.

### GetAllowedToolsOk

`func (o *V1alpha1KomputerAgentSpec) GetAllowedToolsOk() (*[]string, bool)`

GetAllowedToolsOk returns a tuple with the AllowedTools field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowedTools

`func (o *V1alpha1KomputerAgentSpec) SetAllowedTools(v []string)`

SetAllowedTools sets AllowedTools field to given value.

### HasAllowedTools

`func (o *V1alpha1KomputerAgentSpec) HasAllowedTools() bool`

HasAllowedTools returns a boolean if a field has been set.

### GetArchiveTTL

`func (o *V1alpha1KomputerAgentSpec) GetArchiveTTL() V1Duration`

GetArchiveTTL returns the ArchiveTTL field if non-nil, zero value otherwise.

### GetArchiveTTLOk

`func (o *V1alpha1KomputerAgentSpec) GetArchiveTTLOk() (*V1Duration, bool)`

GetArchiveTTLOk returns a tuple with the ArchiveTTL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArchiveTTL

`func (o *V1alpha1KomputerAgentSpec) SetArchiveTTL(v V1Duration)`

SetArchiveTTL sets ArchiveTTL field to given value.

### HasArchiveTTL

`func (o *V1alpha1KomputerAgentSpec) HasArchiveTTL() bool`

HasArchiveTTL returns a boolean if a field has been set.

### GetBudgetUSD

`func (o *V1alpha1KomputerAgentSpec) GetBudgetUSD() string`

GetBudgetUSD returns the BudgetUSD field if non-nil, zero value otherwise.

### GetBudgetUSDOk

`func (o *V1alpha1KomputerAgentSpec) GetBudgetUSDOk() (*string, bool)`

GetBudgetUSDOk returns a tuple with the BudgetUSD field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBudgetUSD

`func (o *V1alpha1KomputerAgentSpec) SetBudgetUSD(v string)`

SetBudgetUSD sets BudgetUSD field to given value.

### HasBudgetUSD

`func (o *V1alpha1KomputerAgentSpec) HasBudgetUSD() bool`

HasBudgetUSD returns a boolean if a field has been set.

### GetConnectors

`func (o *V1alpha1KomputerAgentSpec) GetConnectors() []string`

GetConnectors returns the Connectors field if non-nil, zero value otherwise.

### GetConnectorsOk

`func (o *V1alpha1KomputerAgentSpec) GetConnectorsOk() (*[]string, bool)`

GetConnectorsOk returns a tuple with the Connectors field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConnectors

`func (o *V1alpha1KomputerAgentSpec) SetConnectors(v []string)`

SetConnectors sets Connectors field to given value.

### HasConnectors

`func (o *V1alpha1KomputerAgentSpec) HasConnectors() bool`

HasConnectors returns a boolean if a field has been set.

### GetDeleteTTL

`func (o *V1alpha1KomputerAgentSpec) GetDeleteTTL() V1Duration`

GetDeleteTTL returns the DeleteTTL field if non-nil, zero value otherwise.

### GetDeleteTTLOk

`func (o *V1alpha1KomputerAgentSpec) GetDeleteTTLOk() (*V1Duration, bool)`

GetDeleteTTLOk returns a tuple with the DeleteTTL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeleteTTL

`func (o *V1alpha1KomputerAgentSpec) SetDeleteTTL(v V1Duration)`

SetDeleteTTL sets DeleteTTL field to given value.

### HasDeleteTTL

`func (o *V1alpha1KomputerAgentSpec) HasDeleteTTL() bool`

HasDeleteTTL returns a boolean if a field has been set.

### GetDisallowedTools

`func (o *V1alpha1KomputerAgentSpec) GetDisallowedTools() []string`

GetDisallowedTools returns the DisallowedTools field if non-nil, zero value otherwise.

### GetDisallowedToolsOk

`func (o *V1alpha1KomputerAgentSpec) GetDisallowedToolsOk() (*[]string, bool)`

GetDisallowedToolsOk returns a tuple with the DisallowedTools field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisallowedTools

`func (o *V1alpha1KomputerAgentSpec) SetDisallowedTools(v []string)`

SetDisallowedTools sets DisallowedTools field to given value.

### HasDisallowedTools

`func (o *V1alpha1KomputerAgentSpec) HasDisallowedTools() bool`

HasDisallowedTools returns a boolean if a field has been set.

### GetEffort

`func (o *V1alpha1KomputerAgentSpec) GetEffort() string`

GetEffort returns the Effort field if non-nil, zero value otherwise.

### GetEffortOk

`func (o *V1alpha1KomputerAgentSpec) GetEffortOk() (*string, bool)`

GetEffortOk returns a tuple with the Effort field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEffort

`func (o *V1alpha1KomputerAgentSpec) SetEffort(v string)`

SetEffort sets Effort field to given value.

### HasEffort

`func (o *V1alpha1KomputerAgentSpec) HasEffort() bool`

HasEffort returns a boolean if a field has been set.

### GetInstructions

`func (o *V1alpha1KomputerAgentSpec) GetInstructions() string`

GetInstructions returns the Instructions field if non-nil, zero value otherwise.

### GetInstructionsOk

`func (o *V1alpha1KomputerAgentSpec) GetInstructionsOk() (*string, bool)`

GetInstructionsOk returns a tuple with the Instructions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstructions

`func (o *V1alpha1KomputerAgentSpec) SetInstructions(v string)`

SetInstructions sets Instructions field to given value.

### HasInstructions

`func (o *V1alpha1KomputerAgentSpec) HasInstructions() bool`

HasInstructions returns a boolean if a field has been set.

### GetInternalSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) GetInternalSystemPrompt() string`

GetInternalSystemPrompt returns the InternalSystemPrompt field if non-nil, zero value otherwise.

### GetInternalSystemPromptOk

`func (o *V1alpha1KomputerAgentSpec) GetInternalSystemPromptOk() (*string, bool)`

GetInternalSystemPromptOk returns a tuple with the InternalSystemPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInternalSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) SetInternalSystemPrompt(v string)`

SetInternalSystemPrompt sets InternalSystemPrompt field to given value.

### HasInternalSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) HasInternalSystemPrompt() bool`

HasInternalSystemPrompt returns a boolean if a field has been set.

### GetLabels

`func (o *V1alpha1KomputerAgentSpec) GetLabels() map[string]string`

GetLabels returns the Labels field if non-nil, zero value otherwise.

### GetLabelsOk

`func (o *V1alpha1KomputerAgentSpec) GetLabelsOk() (*map[string]string, bool)`

GetLabelsOk returns a tuple with the Labels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabels

`func (o *V1alpha1KomputerAgentSpec) SetLabels(v map[string]string)`

SetLabels sets Labels field to given value.

### HasLabels

`func (o *V1alpha1KomputerAgentSpec) HasLabels() bool`

HasLabels returns a boolean if a field has been set.

### GetLifecycle

`func (o *V1alpha1KomputerAgentSpec) GetLifecycle() V1alpha1AgentLifecycle`

GetLifecycle returns the Lifecycle field if non-nil, zero value otherwise.

### GetLifecycleOk

`func (o *V1alpha1KomputerAgentSpec) GetLifecycleOk() (*V1alpha1AgentLifecycle, bool)`

GetLifecycleOk returns a tuple with the Lifecycle field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLifecycle

`func (o *V1alpha1KomputerAgentSpec) SetLifecycle(v V1alpha1AgentLifecycle)`

SetLifecycle sets Lifecycle field to given value.

### HasLifecycle

`func (o *V1alpha1KomputerAgentSpec) HasLifecycle() bool`

HasLifecycle returns a boolean if a field has been set.

### GetMemories

`func (o *V1alpha1KomputerAgentSpec) GetMemories() []string`

GetMemories returns the Memories field if non-nil, zero value otherwise.

### GetMemoriesOk

`func (o *V1alpha1KomputerAgentSpec) GetMemoriesOk() (*[]string, bool)`

GetMemoriesOk returns a tuple with the Memories field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemories

`func (o *V1alpha1KomputerAgentSpec) SetMemories(v []string)`

SetMemories sets Memories field to given value.

### HasMemories

`func (o *V1alpha1KomputerAgentSpec) HasMemories() bool`

HasMemories returns a boolean if a field has been set.

### GetModel

`func (o *V1alpha1KomputerAgentSpec) GetModel() string`

GetModel returns the Model field if non-nil, zero value otherwise.

### GetModelOk

`func (o *V1alpha1KomputerAgentSpec) GetModelOk() (*string, bool)`

GetModelOk returns a tuple with the Model field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetModel

`func (o *V1alpha1KomputerAgentSpec) SetModel(v string)`

SetModel sets Model field to given value.

### HasModel

`func (o *V1alpha1KomputerAgentSpec) HasModel() bool`

HasModel returns a boolean if a field has been set.

### GetOfficeManager

`func (o *V1alpha1KomputerAgentSpec) GetOfficeManager() string`

GetOfficeManager returns the OfficeManager field if non-nil, zero value otherwise.

### GetOfficeManagerOk

`func (o *V1alpha1KomputerAgentSpec) GetOfficeManagerOk() (*string, bool)`

GetOfficeManagerOk returns a tuple with the OfficeManager field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOfficeManager

`func (o *V1alpha1KomputerAgentSpec) SetOfficeManager(v string)`

SetOfficeManager sets OfficeManager field to given value.

### HasOfficeManager

`func (o *V1alpha1KomputerAgentSpec) HasOfficeManager() bool`

HasOfficeManager returns a boolean if a field has been set.

### GetPodSpec

`func (o *V1alpha1KomputerAgentSpec) GetPodSpec() V1PodSpec`

GetPodSpec returns the PodSpec field if non-nil, zero value otherwise.

### GetPodSpecOk

`func (o *V1alpha1KomputerAgentSpec) GetPodSpecOk() (*V1PodSpec, bool)`

GetPodSpecOk returns a tuple with the PodSpec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPodSpec

`func (o *V1alpha1KomputerAgentSpec) SetPodSpec(v V1PodSpec)`

SetPodSpec sets PodSpec field to given value.

### HasPodSpec

`func (o *V1alpha1KomputerAgentSpec) HasPodSpec() bool`

HasPodSpec returns a boolean if a field has been set.

### GetPriority

`func (o *V1alpha1KomputerAgentSpec) GetPriority() int32`

GetPriority returns the Priority field if non-nil, zero value otherwise.

### GetPriorityOk

`func (o *V1alpha1KomputerAgentSpec) GetPriorityOk() (*int32, bool)`

GetPriorityOk returns a tuple with the Priority field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPriority

`func (o *V1alpha1KomputerAgentSpec) SetPriority(v int32)`

SetPriority sets Priority field to given value.

### HasPriority

`func (o *V1alpha1KomputerAgentSpec) HasPriority() bool`

HasPriority returns a boolean if a field has been set.

### GetRole

`func (o *V1alpha1KomputerAgentSpec) GetRole() string`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *V1alpha1KomputerAgentSpec) GetRoleOk() (*string, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *V1alpha1KomputerAgentSpec) SetRole(v string)`

SetRole sets Role field to given value.

### HasRole

`func (o *V1alpha1KomputerAgentSpec) HasRole() bool`

HasRole returns a boolean if a field has been set.

### GetSecrets

`func (o *V1alpha1KomputerAgentSpec) GetSecrets() []string`

GetSecrets returns the Secrets field if non-nil, zero value otherwise.

### GetSecretsOk

`func (o *V1alpha1KomputerAgentSpec) GetSecretsOk() (*[]string, bool)`

GetSecretsOk returns a tuple with the Secrets field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecrets

`func (o *V1alpha1KomputerAgentSpec) SetSecrets(v []string)`

SetSecrets sets Secrets field to given value.

### HasSecrets

`func (o *V1alpha1KomputerAgentSpec) HasSecrets() bool`

HasSecrets returns a boolean if a field has been set.

### GetSkills

`func (o *V1alpha1KomputerAgentSpec) GetSkills() []string`

GetSkills returns the Skills field if non-nil, zero value otherwise.

### GetSkillsOk

`func (o *V1alpha1KomputerAgentSpec) GetSkillsOk() (*[]string, bool)`

GetSkillsOk returns a tuple with the Skills field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSkills

`func (o *V1alpha1KomputerAgentSpec) SetSkills(v []string)`

SetSkills sets Skills field to given value.

### HasSkills

`func (o *V1alpha1KomputerAgentSpec) HasSkills() bool`

HasSkills returns a boolean if a field has been set.

### GetSleepTTL

`func (o *V1alpha1KomputerAgentSpec) GetSleepTTL() V1Duration`

GetSleepTTL returns the SleepTTL field if non-nil, zero value otherwise.

### GetSleepTTLOk

`func (o *V1alpha1KomputerAgentSpec) GetSleepTTLOk() (*V1Duration, bool)`

GetSleepTTLOk returns a tuple with the SleepTTL field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSleepTTL

`func (o *V1alpha1KomputerAgentSpec) SetSleepTTL(v V1Duration)`

SetSleepTTL sets SleepTTL field to given value.

### HasSleepTTL

`func (o *V1alpha1KomputerAgentSpec) HasSleepTTL() bool`

HasSleepTTL returns a boolean if a field has been set.

### GetStorage

`func (o *V1alpha1KomputerAgentSpec) GetStorage() V1alpha1StorageSpec`

GetStorage returns the Storage field if non-nil, zero value otherwise.

### GetStorageOk

`func (o *V1alpha1KomputerAgentSpec) GetStorageOk() (*V1alpha1StorageSpec, bool)`

GetStorageOk returns a tuple with the Storage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStorage

`func (o *V1alpha1KomputerAgentSpec) SetStorage(v V1alpha1StorageSpec)`

SetStorage sets Storage field to given value.

### HasStorage

`func (o *V1alpha1KomputerAgentSpec) HasStorage() bool`

HasStorage returns a boolean if a field has been set.

### GetSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) GetSystemPrompt() string`

GetSystemPrompt returns the SystemPrompt field if non-nil, zero value otherwise.

### GetSystemPromptOk

`func (o *V1alpha1KomputerAgentSpec) GetSystemPromptOk() (*string, bool)`

GetSystemPromptOk returns a tuple with the SystemPrompt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) SetSystemPrompt(v string)`

SetSystemPrompt sets SystemPrompt field to given value.

### HasSystemPrompt

`func (o *V1alpha1KomputerAgentSpec) HasSystemPrompt() bool`

HasSystemPrompt returns a boolean if a field has been set.

### GetTaskBudgetUSD

`func (o *V1alpha1KomputerAgentSpec) GetTaskBudgetUSD() string`

GetTaskBudgetUSD returns the TaskBudgetUSD field if non-nil, zero value otherwise.

### GetTaskBudgetUSDOk

`func (o *V1alpha1KomputerAgentSpec) GetTaskBudgetUSDOk() (*string, bool)`

GetTaskBudgetUSDOk returns a tuple with the TaskBudgetUSD field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTaskBudgetUSD

`func (o *V1alpha1KomputerAgentSpec) SetTaskBudgetUSD(v string)`

SetTaskBudgetUSD sets TaskBudgetUSD field to given value.

### HasTaskBudgetUSD

`func (o *V1alpha1KomputerAgentSpec) HasTaskBudgetUSD() bool`

HasTaskBudgetUSD returns a boolean if a field has been set.

### GetTaskTimeout

`func (o *V1alpha1KomputerAgentSpec) GetTaskTimeout() V1Duration`

GetTaskTimeout returns the TaskTimeout field if non-nil, zero value otherwise.

### GetTaskTimeoutOk

`func (o *V1alpha1KomputerAgentSpec) GetTaskTimeoutOk() (*V1Duration, bool)`

GetTaskTimeoutOk returns a tuple with the TaskTimeout field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTaskTimeout

`func (o *V1alpha1KomputerAgentSpec) SetTaskTimeout(v V1Duration)`

SetTaskTimeout sets TaskTimeout field to given value.

### HasTaskTimeout

`func (o *V1alpha1KomputerAgentSpec) HasTaskTimeout() bool`

HasTaskTimeout returns a boolean if a field has been set.

### GetTemplateRef

`func (o *V1alpha1KomputerAgentSpec) GetTemplateRef() string`

GetTemplateRef returns the TemplateRef field if non-nil, zero value otherwise.

### GetTemplateRefOk

`func (o *V1alpha1KomputerAgentSpec) GetTemplateRefOk() (*string, bool)`

GetTemplateRefOk returns a tuple with the TemplateRef field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTemplateRef

`func (o *V1alpha1KomputerAgentSpec) SetTemplateRef(v string)`

SetTemplateRef sets TemplateRef field to given value.

### HasTemplateRef

`func (o *V1alpha1KomputerAgentSpec) HasTemplateRef() bool`

HasTemplateRef returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


