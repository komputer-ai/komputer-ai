
# V1ContainerRestartRule


## Properties

Name | Type
------------ | -------------
`action` | [V1ContainerRestartRuleAction](V1ContainerRestartRuleAction.md)
`exitCodes` | [V1ContainerRestartRuleOnExitCodes](V1ContainerRestartRuleOnExitCodes.md)

## Example

```typescript
import type { V1ContainerRestartRule } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "action": null,
  "exitCodes": null,
} satisfies V1ContainerRestartRule

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ContainerRestartRule
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


