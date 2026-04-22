
# V1PhotonPersistentDiskVolumeSource


## Properties

Name | Type
------------ | -------------
`fsType` | string
`pdID` | string

## Example

```typescript
import type { V1PhotonPersistentDiskVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "fsType": null,
  "pdID": null,
} satisfies V1PhotonPersistentDiskVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PhotonPersistentDiskVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


