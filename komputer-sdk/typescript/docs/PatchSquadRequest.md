
# PatchSquadRequest


## Properties

Name | Type
------------ | -------------
`members` | [Array&lt;V1alpha1KomputerSquadMember&gt;](V1alpha1KomputerSquadMember.md)
`orphanTTL` | string

## Example

```typescript
import type { PatchSquadRequest } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "members": null,
  "orphanTTL": null,
} satisfies PatchSquadRequest

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as PatchSquadRequest
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


