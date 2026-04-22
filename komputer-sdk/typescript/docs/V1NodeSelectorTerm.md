
# V1NodeSelectorTerm


## Properties

Name | Type
------------ | -------------
`matchExpressions` | [Array&lt;V1NodeSelectorRequirement&gt;](V1NodeSelectorRequirement.md)
`matchFields` | [Array&lt;V1NodeSelectorRequirement&gt;](V1NodeSelectorRequirement.md)

## Example

```typescript
import type { V1NodeSelectorTerm } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "matchExpressions": null,
  "matchFields": null,
} satisfies V1NodeSelectorTerm

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1NodeSelectorTerm
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


