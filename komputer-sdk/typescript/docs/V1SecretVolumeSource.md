
# V1SecretVolumeSource


## Properties

Name | Type
------------ | -------------
`defaultMode` | number
`items` | [Array&lt;V1KeyToPath&gt;](V1KeyToPath.md)
`optional` | boolean
`secretName` | string

## Example

```typescript
import type { V1SecretVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "defaultMode": null,
  "items": null,
  "optional": null,
  "secretName": null,
} satisfies V1SecretVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1SecretVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


