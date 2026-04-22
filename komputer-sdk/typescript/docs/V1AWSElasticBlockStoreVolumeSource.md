
# V1AWSElasticBlockStoreVolumeSource


## Properties

Name | Type
------------ | -------------
`fsType` | string
`partition` | number
`readOnly` | boolean
`volumeID` | string

## Example

```typescript
import type { V1AWSElasticBlockStoreVolumeSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "fsType": null,
  "partition": null,
  "readOnly": null,
  "volumeID": null,
} satisfies V1AWSElasticBlockStoreVolumeSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1AWSElasticBlockStoreVolumeSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


