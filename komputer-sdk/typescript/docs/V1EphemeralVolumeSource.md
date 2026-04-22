
# V1EphemeralVolumeSource


## Properties

Name | Type
------------ | -------------
`volumeClaimTemplate` | [V1PersistentVolumeClaimTemplate](V1PersistentVolumeClaimTemplate.md)

## Example

```typescript
import type { V1EphemeralVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "volumeClaimTemplate": null,
} satisfies V1EphemeralVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1EphemeralVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


