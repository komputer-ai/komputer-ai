
# V1ContainerResizePolicy


## Properties

Name | Type
------------ | -------------
`resourceName` | [V1ResourceName](V1ResourceName.md)
`restartPolicy` | [V1ResourceResizeRestartPolicy](V1ResourceResizeRestartPolicy.md)

## Example

```typescript
import type { V1ContainerResizePolicy } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "resourceName": null,
  "restartPolicy": null,
} satisfies V1ContainerResizePolicy

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ContainerResizePolicy
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


