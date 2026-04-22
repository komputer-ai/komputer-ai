
# AgentResponse


## Properties

Name | Type
------------ | -------------
`connectors` | Array&lt;string&gt;
`createdAt` | string
`instructions` | string
`lastTaskCostUSD` | string
`lastTaskMessage` | string
`lifecycle` | string
`memories` | Array&lt;string&gt;
`model` | string
`modelContextWindow` | number
`name` | string
`namespace` | string
`podSpec` | [V1PodSpec](V1PodSpec.md)
`secrets` | Array&lt;string&gt;
`skills` | Array&lt;string&gt;
`status` | string
`storage` | [V1alpha1StorageSpec](V1alpha1StorageSpec.md)
`systemPrompt` | string
`taskStatus` | string
`totalCostUSD` | string
`totalTokens` | number

## Example

```typescript
import type { AgentResponse } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "connectors": null,
  "createdAt": null,
  "instructions": null,
  "lastTaskCostUSD": null,
  "lastTaskMessage": null,
  "lifecycle": null,
  "memories": null,
  "model": null,
  "modelContextWindow": null,
  "name": null,
  "namespace": null,
  "podSpec": null,
  "secrets": null,
  "skills": null,
  "status": null,
  "storage": null,
  "systemPrompt": null,
  "taskStatus": null,
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


