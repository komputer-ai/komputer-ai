
# V1alpha1KomputerAgentSpec


## Properties

Name | Type
------------ | -------------
`connectors` | Array&lt;string&gt;
`instructions` | string
`internalSystemPrompt` | string
`lifecycle` | [V1alpha1AgentLifecycle](V1alpha1AgentLifecycle.md)
`memories` | Array&lt;string&gt;
`model` | string
`officeManager` | string
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
import type { V1alpha1KomputerAgentSpec } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "connectors": null,
  "instructions": null,
  "internalSystemPrompt": null,
  "lifecycle": null,
  "memories": null,
  "model": null,
  "officeManager": null,
  "podSpec": null,
  "priority": null,
  "role": null,
  "secrets": null,
  "skills": null,
  "storage": null,
  "systemPrompt": null,
  "templateRef": null,
} satisfies V1alpha1KomputerAgentSpec

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1alpha1KomputerAgentSpec
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


