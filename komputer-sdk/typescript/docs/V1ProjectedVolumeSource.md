
# V1ProjectedVolumeSource


## Properties

Name | Type
------------ | -------------
`defaultMode` | number
`sources` | [Array&lt;V1VolumeProjection&gt;](V1VolumeProjection.md)

## Example

```typescript
import type { V1ProjectedVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "defaultMode": null,
  "sources": null,
} satisfies V1ProjectedVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ProjectedVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


