
# V1TypedLocalObjectReference


## Properties

Name | Type
------------ | -------------
`apiGroup` | string
`kind` | string
`name` | string

## Example

```typescript
import type { V1TypedLocalObjectReference } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "apiGroup": null,
  "kind": null,
  "name": null,
} satisfies V1TypedLocalObjectReference

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1TypedLocalObjectReference
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


