
# V1ServiceAccountTokenProjection


## Properties

Name | Type
------------ | -------------
`audience` | string
`expirationSeconds` | number
`path` | string

## Example

```typescript
import type { V1ServiceAccountTokenProjection } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "audience": null,
  "expirationSeconds": null,
  "path": null,
} satisfies V1ServiceAccountTokenProjection

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ServiceAccountTokenProjection
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


