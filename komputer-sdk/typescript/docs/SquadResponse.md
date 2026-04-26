
# SquadResponse


## Properties

Name | Type
------------ | -------------
`createdAt` | string
`members` | [Array&lt;SquadMemberResponse&gt;](SquadMemberResponse.md)
`message` | string
`name` | string
`namespace` | string
`orphanTTL` | string
`orphanedSince` | string
`phase` | string
`podName` | string

## Example

```typescript
import type { SquadResponse } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "createdAt": null,
  "members": null,
  "message": null,
  "name": null,
  "namespace": null,
  "orphanTTL": null,
  "orphanedSince": null,
  "phase": null,
  "podName": null,
} satisfies SquadResponse

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as SquadResponse
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


