
# V1PersistentVolumeClaimTemplate


## Properties

Name | Type
------------ | -------------
`metadata` | [V1ObjectMeta](V1ObjectMeta.md)
`spec` | [V1PersistentVolumeClaimSpec](V1PersistentVolumeClaimSpec.md)

## Example

```typescript
import type { V1PersistentVolumeClaimTemplate } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "metadata": null,
  "spec": null,
} satisfies V1PersistentVolumeClaimTemplate

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PersistentVolumeClaimTemplate
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


