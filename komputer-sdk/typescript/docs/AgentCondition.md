
# AgentCondition


## Properties

Name | Type
------------ | -------------
`lastTransitionTime` | string
`message` | string
`reason` | string
`status` | string
`type` | string

## Example

```typescript
import type { AgentCondition } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "lastTransitionTime": null,
  "message": null,
  "reason": null,
  "status": null,
  "type": null,
} satisfies AgentCondition

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as AgentCondition
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


