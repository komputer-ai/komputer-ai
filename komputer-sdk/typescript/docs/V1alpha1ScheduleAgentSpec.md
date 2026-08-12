
# V1alpha1ScheduleAgentSpec


## Properties

Name | Type
------------ | -------------
`allowedTools` | Array&lt;string&gt;
`connectors` | Array&lt;string&gt;
`disallowedTools` | Array&lt;string&gt;
`labels` | { [key: string]: string; }
`lifecycle` | [V1alpha1AgentLifecycle](V1alpha1AgentLifecycle.md)
`memories` | Array&lt;string&gt;
`model` | string
`podSpec` | [V1PodSpec](V1PodSpec.md)
`priority` | number
`role` | string
`secrets` | Array&lt;string&gt;
`skills` | Array&lt;string&gt;
`storage` | [V1alpha1StorageSpec](V1alpha1StorageSpec.md)
`systemPrompt` | string
`templateRef` | string

## Example

```typescript
import type { V1alpha1ScheduleAgentSpec } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "allowedTools": null,
  "connectors": null,
  "disallowedTools": null,
  "labels": null,
  "lifecycle": null,
  "memories": null,
  "model": null,
  "podSpec": null,
  "priority": null,
  "role": null,
  "secrets": null,
  "skills": null,
  "storage": null,
  "systemPrompt": null,
  "templateRef": null,
} satisfies V1alpha1ScheduleAgentSpec

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1alpha1ScheduleAgentSpec
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


