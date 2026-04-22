
# V1OwnerReference


## Properties

Name | Type
------------ | -------------
`apiVersion` | string
`blockOwnerDeletion` | boolean
`controller` | boolean
`kind` | string
`name` | string
`uid` | string

## Example

```typescript
import type { V1OwnerReference } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "apiVersion": null,
  "blockOwnerDeletion": null,
  "controller": null,
  "kind": null,
  "name": null,
  "uid": null,
} satisfies V1OwnerReference

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1OwnerReference
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


