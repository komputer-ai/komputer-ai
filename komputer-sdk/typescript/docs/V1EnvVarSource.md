
# V1EnvVarSource


## Properties

Name | Type
------------ | -------------
`configMapKeyRef` | [V1ConfigMapKeySelector](V1ConfigMapKeySelector.md)
`fieldRef` | [V1ObjectFieldSelector](V1ObjectFieldSelector.md)
`fileKeyRef` | [V1FileKeySelector](V1FileKeySelector.md)
`resourceFieldRef` | [V1ResourceFieldSelector](V1ResourceFieldSelector.md)
`secretKeyRef` | [V1SecretKeySelector](V1SecretKeySelector.md)

## Example

```typescript
import type { V1EnvVarSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "configMapKeyRef": null,
  "fieldRef": null,
  "fileKeyRef": null,
  "resourceFieldRef": null,
  "secretKeyRef": null,
} satisfies V1EnvVarSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1EnvVarSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


