
# V1AzureDiskVolumeSource


## Properties

Name | Type
------------ | -------------
`cachingMode` | [V1AzureDataDiskCachingMode](V1AzureDataDiskCachingMode.md)
`diskName` | string
`diskURI` | string
`fsType` | string
`kind` | [V1AzureDataDiskKind](V1AzureDataDiskKind.md)
`readOnly` | boolean

## Example

```typescript
import type { V1AzureDiskVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "cachingMode": null,
  "diskName": null,
  "diskURI": null,
  "fsType": null,
  "kind": null,
  "readOnly": null,
} satisfies V1AzureDiskVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1AzureDiskVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


