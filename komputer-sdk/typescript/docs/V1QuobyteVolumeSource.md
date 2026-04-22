
# V1QuobyteVolumeSource


## Properties

Name | Type
------------ | -------------
`group` | string
`readOnly` | boolean
`registry` | string
`tenant` | string
`user` | string
`volume` | string

## Example

```typescript
import type { V1QuobyteVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "group": null,
  "readOnly": null,
  "registry": null,
  "tenant": null,
  "user": null,
  "volume": null,
} satisfies V1QuobyteVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1QuobyteVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


