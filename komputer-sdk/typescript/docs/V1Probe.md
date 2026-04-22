
# V1Probe


## Properties

Name | Type
------------ | -------------
`exec` | [V1ExecAction](V1ExecAction.md)
`failureThreshold` | number
`grpc` | [V1GRPCAction](V1GRPCAction.md)
`httpGet` | [V1HTTPGetAction](V1HTTPGetAction.md)
`initialDelaySeconds` | number
`periodSeconds` | number
`successThreshold` | number
`tcpSocket` | [V1TCPSocketAction](V1TCPSocketAction.md)
`terminationGracePeriodSeconds` | number
`timeoutSeconds` | number

## Example

```typescript
import type { V1Probe } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "exec": null,
  "failureThreshold": null,
  "grpc": null,
  "httpGet": null,
  "initialDelaySeconds": null,
  "periodSeconds": null,
  "successThreshold": null,
  "tcpSocket": null,
  "terminationGracePeriodSeconds": null,
  "timeoutSeconds": null,
} satisfies V1Probe

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1Probe
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


