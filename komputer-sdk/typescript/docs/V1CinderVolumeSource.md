
# V1CinderVolumeSource


## Properties

Name | Type
------------ | -------------
`fsType` | string
`readOnly` | boolean
`secretRef` | [V1LocalObjectReference](V1LocalObjectReference.md)
`volumeID` | string

## Example

```typescript
import type { V1CinderVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "fsType": null,
  "readOnly": null,
  "secretRef": null,
  "volumeID": null,
} satisfies V1CinderVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1CinderVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


