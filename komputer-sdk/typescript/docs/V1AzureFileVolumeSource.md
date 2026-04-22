
# V1AzureFileVolumeSource


## Properties

Name | Type
------------ | -------------
`readOnly` | boolean
`secretName` | string
`shareName` | string

## Example

```typescript
import type { V1AzureFileVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "readOnly": null,
  "secretName": null,
  "shareName": null,
} satisfies V1AzureFileVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1AzureFileVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


