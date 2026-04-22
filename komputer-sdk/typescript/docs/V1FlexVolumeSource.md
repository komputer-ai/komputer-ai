
# V1FlexVolumeSource


## Properties

Name | Type
------------ | -------------
`driver` | string
`fsType` | string
`options` | { [key: string]: string; }
`readOnly` | boolean
`secretRef` | [V1LocalObjectReference](V1LocalObjectReference.md)

## Example

```typescript
import type { V1FlexVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "driver": null,
  "fsType": null,
  "options": null,
  "readOnly": null,
  "secretRef": null,
} satisfies V1FlexVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1FlexVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


