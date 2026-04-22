
# V1ConfigMapVolumeSource


## Properties

Name | Type
------------ | -------------
`defaultMode` | number
`items` | [Array&lt;V1KeyToPath&gt;](V1KeyToPath.md)
`name` | string
`optional` | boolean

## Example

```typescript
import type { V1ConfigMapVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "defaultMode": null,
  "items": null,
  "name": null,
  "optional": null,
} satisfies V1ConfigMapVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ConfigMapVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


