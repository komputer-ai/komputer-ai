
# V1VsphereVirtualDiskVolumeSource


## Properties

Name | Type
------------ | -------------
`fsType` | string
`storagePolicyID` | string
`storagePolicyName` | string
`volumePath` | string

## Example

```typescript
import type { V1VsphereVirtualDiskVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "fsType": null,
  "storagePolicyID": null,
  "storagePolicyName": null,
  "volumePath": null,
} satisfies V1VsphereVirtualDiskVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1VsphereVirtualDiskVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


