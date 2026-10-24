
# SessionStoreResponse


## Properties

Name | Type
------------ | -------------
`error` | string
`lastSavedAt` | string
`ok` | boolean

## Example

```typescript
import type { SessionStoreResponse } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "error": null,
  "lastSavedAt": null,
  "ok": null,
} satisfies SessionStoreResponse

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as SessionStoreResponse
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


