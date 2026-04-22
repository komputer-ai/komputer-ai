
# V1NFSVolumeSource


## Properties

Name | Type
------------ | -------------
`path` | string
`readOnly` | boolean
`server` | string

## Example

```typescript
import type { V1NFSVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "path": null,
  "readOnly": null,
  "server": null,
} satisfies V1NFSVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1NFSVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


