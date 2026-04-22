
# V1Lifecycle


## Properties

Name | Type
------------ | -------------
`postStart` | [V1LifecycleHandler](V1LifecycleHandler.md)
`preStop` | [V1LifecycleHandler](V1LifecycleHandler.md)
`stopSignal` | [V1Signal](V1Signal.md)

## Example

```typescript
import type { V1Lifecycle } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "postStart": null,
  "preStop": null,
  "stopSignal": null,
} satisfies V1Lifecycle

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1Lifecycle
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


