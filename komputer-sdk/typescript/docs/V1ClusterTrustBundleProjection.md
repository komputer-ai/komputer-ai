
# V1ClusterTrustBundleProjection


## Properties

Name | Type
------------ | -------------
`labelSelector` | [V1LabelSelector](V1LabelSelector.md)
`name` | string
`optional` | boolean
`path` | string
`signerName` | string

## Example

```typescript
import type { V1ClusterTrustBundleProjection } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "labelSelector": null,
  "name": null,
  "optional": null,
  "path": null,
  "signerName": null,
} satisfies V1ClusterTrustBundleProjection

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ClusterTrustBundleProjection
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


