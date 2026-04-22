
# V1EnvFromSource


## Properties

Name | Type
------------ | -------------
`configMapRef` | [V1ConfigMapEnvSource](V1ConfigMapEnvSource.md)
`prefix` | string
`secretRef` | [V1SecretEnvSource](V1SecretEnvSource.md)

## Example

```typescript
import type { V1EnvFromSource } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "configMapRef": null,
  "prefix": null,
  "secretRef": null,
} satisfies V1EnvFromSource

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1EnvFromSource
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


