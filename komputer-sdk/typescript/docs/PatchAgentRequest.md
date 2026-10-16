
# PatchAgentRequest


## Properties

Name | Type
------------ | -------------
`allowedTools` | Array&lt;string&gt;
`budgetUSD` | string
`connectors` | Array&lt;string&gt;
`deleteTTL` | string
`disallowedTools` | Array&lt;string&gt;
`effort` | string
`instructions` | string
`labels` | { [key: string]: string; }
`lifecycle` | string
`memories` | Array&lt;string&gt;
`model` | string
`podSpec` | [V1PodSpec](V1PodSpec.md)
`priority` | number
`removeLabels` | Array&lt;string&gt;
`secretRefs` | Array&lt;string&gt;
`skills` | Array&lt;string&gt;
`sleepTTL` | string
`storage` | [V1alpha1StorageSpec](V1alpha1StorageSpec.md)
`systemPrompt` | string
`taskBudgetUSD` | string
`taskTimeout` | string
`templateRef` | string

## Example

```typescript
import type { PatchAgentRequest } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "allowedTools": null,
  "budgetUSD": null,
  "connectors": null,
  "deleteTTL": null,
  "disallowedTools": null,
  "effort": null,
  "instructions": null,
  "labels": null,
  "lifecycle": null,
  "memories": null,
  "model": null,
  "podSpec": null,
  "priority": null,
  "removeLabels": null,
  "secretRefs": null,
  "skills": null,
  "sleepTTL": null,
  "storage": null,
  "systemPrompt": null,
  "taskBudgetUSD": null,
  "taskTimeout": null,
  "templateRef": null,
} satisfies PatchAgentRequest

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as PatchAgentRequest
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


