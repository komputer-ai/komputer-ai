
# V1ContainerPort


## Properties

Name | Type
------------ | -------------
`containerPort` | number
`hostIP` | string
`hostPort` | number
`name` | string
`protocol` | [V1Protocol](V1Protocol.md)

## Example

```typescript
import type { V1ContainerPort } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "containerPort": null,
  "hostIP": null,
  "hostPort": null,
  "name": null,
  "protocol": null,
} satisfies V1ContainerPort

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ContainerPort
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


