
# V1RBDVolumeSource


## Properties

Name | Type
------------ | -------------
`fsType` | string
`image` | string
`keyring` | string
`monitors` | Array&lt;string&gt;
`pool` | string
`readOnly` | boolean
`secretRef` | [V1LocalObjectReference](V1LocalObjectReference.md)
`user` | string

## Example

```typescript
import type { V1RBDVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "fsType": null,
  "image": null,
  "keyring": null,
  "monitors": null,
  "pool": null,
  "readOnly": null,
  "secretRef": null,
  "user": null,
} satisfies V1RBDVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1RBDVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


