
# V1LifecycleHandler


## Properties

Name | Type
------------ | -------------
`exec` | [V1ExecAction](V1ExecAction.md)
`httpGet` | [V1HTTPGetAction](V1HTTPGetAction.md)
`sleep` | [V1SleepAction](V1SleepAction.md)
`tcpSocket` | [V1TCPSocketAction](V1TCPSocketAction.md)

## Example

```typescript
import type { V1LifecycleHandler } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "exec": null,
  "httpGet": null,
  "sleep": null,
  "tcpSocket": null,
} satisfies V1LifecycleHandler

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1LifecycleHandler
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


