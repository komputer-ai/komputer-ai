
# V1ISCSIVolumeSource


## Properties

Name | Type
------------ | -------------
`chapAuthDiscovery` | boolean
`chapAuthSession` | boolean
`fsType` | string
`initiatorName` | string
`iqn` | string
`iscsiInterface` | string
`lun` | number
`portals` | Array&lt;string&gt;
`readOnly` | boolean
`secretRef` | [V1LocalObjectReference](V1LocalObjectReference.md)
`targetPortal` | string

## Example

```typescript
import type { V1ISCSIVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "chapAuthDiscovery": null,
  "chapAuthSession": null,
  "fsType": null,
  "initiatorName": null,
  "iqn": null,
  "iscsiInterface": null,
  "lun": null,
  "portals": null,
  "readOnly": null,
  "secretRef": null,
  "targetPortal": null,
} satisfies V1ISCSIVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ISCSIVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


