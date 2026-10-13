
# AgentListResponse


## Properties

Name | Type
------------ | -------------
`agents` | [Array&lt;AgentResponse&gt;](AgentResponse.md)
`facets` | [AgentListFacets](AgentListFacets.md)
`nextOffset` | number
`total` | number

## Example

```typescript
import type { AgentListResponse } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "agents": null,
  "facets": null,
  "nextOffset": null,
  "total": null,
} satisfies AgentListResponse

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as AgentListResponse
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


