
# V1PersistentVolumeClaimSpec


## Properties

Name | Type
------------ | -------------
`accessModes` | [Array&lt;V1PersistentVolumeAccessMode&gt;](V1PersistentVolumeAccessMode.md)
`dataSource` | [V1TypedLocalObjectReference](V1TypedLocalObjectReference.md)
`dataSourceRef` | [V1TypedObjectReference](V1TypedObjectReference.md)
`resources` | [V1VolumeResourceRequirements](V1VolumeResourceRequirements.md)
`selector` | [V1LabelSelector](V1LabelSelector.md)
`storageClassName` | string
`volumeAttributesClassName` | string
`volumeMode` | [V1PersistentVolumeMode](V1PersistentVolumeMode.md)
`volumeName` | string

## Example

```typescript
import type { V1PersistentVolumeClaimSpec } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "accessModes": null,
  "dataSource": null,
  "dataSourceRef": null,
  "resources": null,
  "selector": null,
  "storageClassName": null,
  "volumeAttributesClassName": null,
  "volumeMode": null,
  "volumeName": null,
} satisfies V1PersistentVolumeClaimSpec

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PersistentVolumeClaimSpec
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


