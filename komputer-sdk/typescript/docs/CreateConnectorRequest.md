
# CreateConnectorRequest


## Properties

Name | Type
------------ | -------------
`authSecretKey` | string
`authSecretName` | string
`authType` | string
`displayName` | string
`headerName` | string
`name` | string
`namespace` | string
`oauthClientId` | string
`oauthClientSecret` | string
`service` | string
`type` | string
`url` | string

## Example

```typescript
import type { CreateConnectorRequest } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "authSecretKey": null,
  "authSecretName": null,
  "authType": null,
  "displayName": null,
  "headerName": null,
  "name": null,
  "namespace": null,
  "oauthClientId": null,
  "oauthClientSecret": null,
  "service": null,
  "type": null,
  "url": null,
} satisfies CreateConnectorRequest

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as CreateConnectorRequest
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


