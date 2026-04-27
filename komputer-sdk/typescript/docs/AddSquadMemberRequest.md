
# AddSquadMemberRequest


## Properties

Name | Type
------------ | -------------
`ref` | [V1alpha1KomputerSquadMemberRef](V1alpha1KomputerSquadMemberRef.md)
`spec` | [V1alpha1KomputerAgentSpec](V1alpha1KomputerAgentSpec.md)

## Example

```typescript
import type { AddSquadMemberRequest } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "ref": null,
  "spec": null,
} satisfies AddSquadMemberRequest

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as AddSquadMemberRequest
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


