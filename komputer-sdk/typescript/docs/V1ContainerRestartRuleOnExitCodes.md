
# V1ContainerRestartRuleOnExitCodes


## Properties

Name | Type
------------ | -------------
`operator` | [V1ContainerRestartRuleOnExitCodesOperator](V1ContainerRestartRuleOnExitCodesOperator.md)
`values` | Array&lt;number&gt;

## Example

```typescript
import type { V1ContainerRestartRuleOnExitCodes } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "operator": null,
  "values": null,
} satisfies V1ContainerRestartRuleOnExitCodes

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ContainerRestartRuleOnExitCodes
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


