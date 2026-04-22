
# V1VolumeMount


## Properties

Name | Type
------------ | -------------
`mountPath` | string
`mountPropagation` | [V1MountPropagationMode](V1MountPropagationMode.md)
`name` | string
`readOnly` | boolean
`recursiveReadOnly` | [V1RecursiveReadOnlyMode](V1RecursiveReadOnlyMode.md)
`subPath` | string
`subPathExpr` | string

## Example

```typescript
import type { V1VolumeMount } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "mountPath": null,
  "mountPropagation": null,
  "name": null,
  "readOnly": null,
  "recursiveReadOnly": null,
  "subPath": null,
  "subPathExpr": null,
} satisfies V1VolumeMount

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1VolumeMount
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


