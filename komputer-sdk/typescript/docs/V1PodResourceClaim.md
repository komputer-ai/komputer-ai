
# V1PodResourceClaim


## Properties

Name | Type
------------ | -------------
`name` | string
`resourceClaimName` | string
`resourceClaimTemplateName` | string

## Example

```typescript
import type { V1PodResourceClaim } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "name": null,
  "resourceClaimName": null,
  "resourceClaimTemplateName": null,
} satisfies V1PodResourceClaim

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PodResourceClaim
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


