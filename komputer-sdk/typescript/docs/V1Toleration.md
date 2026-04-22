
# V1Toleration


## Properties

Name | Type
------------ | -------------
`effect` | [V1TaintEffect](V1TaintEffect.md)
`key` | string
`operator` | [V1TolerationOperator](V1TolerationOperator.md)
`tolerationSeconds` | number
`value` | string

## Example

```typescript
import type { V1Toleration } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "effect": null,
  "key": null,
  "operator": null,
  "tolerationSeconds": null,
  "value": null,
} satisfies V1Toleration

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1Toleration
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


