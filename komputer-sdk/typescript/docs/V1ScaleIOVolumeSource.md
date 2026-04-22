
# V1ScaleIOVolumeSource


## Properties

Name | Type
------------ | -------------
`fsType` | string
`gateway` | string
`protectionDomain` | string
`readOnly` | boolean
`secretRef` | [V1LocalObjectReference](V1LocalObjectReference.md)
`sslEnabled` | boolean
`storageMode` | string
`storagePool` | string
`system` | string
`volumeName` | string

## Example

```typescript
import type { V1ScaleIOVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "fsType": null,
  "gateway": null,
  "protectionDomain": null,
  "readOnly": null,
  "secretRef": null,
  "sslEnabled": null,
  "storageMode": null,
  "storagePool": null,
  "system": null,
  "volumeName": null,
} satisfies V1ScaleIOVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ScaleIOVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


