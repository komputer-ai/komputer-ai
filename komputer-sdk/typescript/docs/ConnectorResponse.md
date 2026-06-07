
# ConnectorResponse


## Properties

Name | Type
------------ | -------------
`agentNames` | Array&lt;string&gt;
`attachedAgents` | number
`authSecretKey` | string
`authSecretName` | string
`authType` | string
`createdAt` | string
`displayName` | string
`headerName` | string
`name` | string
`namespace` | string
`oauthStatus` | string
`service` | string
`type` | string
`url` | string

## Example

```typescript
import type { ConnectorResponse } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "agentNames": null,
  "attachedAgents": null,
  "authSecretKey": null,
  "authSecretName": null,
  "authType": null,
  "createdAt": null,
  "displayName": null,
  "headerName": null,
  "name": null,
  "namespace": null,
  "oauthStatus": null,
  "service": null,
  "type": null,
  "url": null,
} satisfies ConnectorResponse

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ConnectorResponse
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


