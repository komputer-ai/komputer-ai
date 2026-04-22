
# V1ResourceFieldSelector


## Properties

Name | Type
------------ | -------------
`containerName` | string
`divisor` | [ResourceQuantity](ResourceQuantity.md)
`resource` | string

## Example

```typescript
import type { V1ResourceFieldSelector } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "containerName": null,
  "divisor": null,
  "resource": null,
} satisfies V1ResourceFieldSelector

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ResourceFieldSelector
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


