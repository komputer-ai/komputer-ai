
# V1FCVolumeSource


## Properties

Name | Type
------------ | -------------
`fsType` | string
`lun` | number
`readOnly` | boolean
`targetWWNs` | Array&lt;string&gt;
`wwids` | Array&lt;string&gt;

## Example

```typescript
import type { V1FCVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "fsType": null,
  "lun": null,
  "readOnly": null,
  "targetWWNs": null,
  "wwids": null,
} satisfies V1FCVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1FCVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


