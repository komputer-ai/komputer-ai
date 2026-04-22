
# V1HTTPGetAction


## Properties

Name | Type
------------ | -------------
`host` | string
`httpHeaders` | [Array&lt;V1HTTPHeader&gt;](V1HTTPHeader.md)
`path` | string
`port` | [IntstrIntOrString](IntstrIntOrString.md)
`scheme` | [V1URIScheme](V1URIScheme.md)

## Example

```typescript
import type { V1HTTPGetAction } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "host": null,
  "httpHeaders": null,
  "path": null,
  "port": null,
  "scheme": null,
} satisfies V1HTTPGetAction

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1HTTPGetAction
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


