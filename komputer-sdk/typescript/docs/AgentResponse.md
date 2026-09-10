
# AgentResponse


## Properties

Name | Type
------------ | -------------
`allowedTools` | Array&lt;string&gt;
`completionTime` | string
`connectors` | Array&lt;string&gt;
`createdAt` | string
`deleteExpiresAt` | string
`deleteTTL` | string
`disallowedTools` | Array&lt;string&gt;
`errors` | Array&lt;string&gt;
`instructions` | string
`labels` | { [key: string]: string; }
`lastActivityAt` | string
`lastTaskCostUSD` | string
`lastTaskMessage` | string
`lifecycle` | string
`memories` | Array&lt;string&gt;
`model` | string
`modelContextWindow` | number
`name` | string
`namespace` | string
`podSpec` | [V1PodSpec](V1PodSpec.md)
`priority` | number
`queuePosition` | number
`queueReason` | string
`secrets` | Array&lt;string&gt;
`skills` | Array&lt;string&gt;
`sleepExpiresAt` | string
`sleepTTL` | string
`squad` | boolean
`squadName` | string
`status` | string
`storage` | [V1alpha1StorageSpec](V1alpha1StorageSpec.md)
`systemPrompt` | string
`taskExpiresAt` | string
`taskStartedAt` | string
`taskStatus` | string
`taskTimeout` | string
`totalCostUSD` | string
`totalTokens` | number

## Example

```typescript
import type { AgentResponse } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "allowedTools": null,
  "completionTime": null,
  "connectors": null,
  "createdAt": null,
  "deleteExpiresAt": null,
  "deleteTTL": null,
  "disallowedTools": null,
  "errors": null,
  "instructions": null,
  "labels": null,
  "lastActivityAt": null,
  "lastTaskCostUSD": null,
  "lastTaskMessage": null,
  "lifecycle": null,
  "memories": null,
  "model": null,
  "modelContextWindow": null,
  "name": null,
  "namespace": null,
  "podSpec": null,
  "priority": null,
  "queuePosition": null,
  "queueReason": null,
  "secrets": null,
  "skills": null,
  "sleepExpiresAt": null,
  "sleepTTL": null,
  "squad": null,
  "squadName": null,
  "status": null,
  "storage": null,
  "systemPrompt": null,
  "taskExpiresAt": null,
  "taskStartedAt": null,
  "taskStatus": null,
  "taskTimeout": null,
  "totalCostUSD": null,
  "totalTokens": null,
} satisfies AgentResponse

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as AgentResponse
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


