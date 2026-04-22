
# V1EmptyDirVolumeSource


## Properties

Name | Type
------------ | -------------
`medium` | [V1StorageMedium](V1StorageMedium.md)
`sizeLimit` | [ResourceQuantity](ResourceQuantity.md)

## Example

```typescript
import type { V1EmptyDirVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "medium": null,
  "sizeLimit": null,
} satisfies V1EmptyDirVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1EmptyDirVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


