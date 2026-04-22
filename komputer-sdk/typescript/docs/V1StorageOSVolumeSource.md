
# V1StorageOSVolumeSource


## Properties

Name | Type
------------ | -------------
`fsType` | string
`readOnly` | boolean
`secretRef` | [V1LocalObjectReference](V1LocalObjectReference.md)
`volumeName` | string
`volumeNamespace` | string

## Example

```typescript
import type { V1StorageOSVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "fsType": null,
  "readOnly": null,
  "secretRef": null,
  "volumeName": null,
  "volumeNamespace": null,
} satisfies V1StorageOSVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1StorageOSVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


