
# V1EnvVar


## Properties

Name | Type
------------ | -------------
`name` | string
`value` | string
`valueFrom` | [V1EnvVarSource](V1EnvVarSource.md)

## Example

```typescript
import type { V1EnvVar } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "name": null,
  "value": null,
  "valueFrom": null,
} satisfies V1EnvVar

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1EnvVar
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


