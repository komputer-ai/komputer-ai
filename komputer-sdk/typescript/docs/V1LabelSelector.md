
# V1LabelSelector


## Properties

Name | Type
------------ | -------------
`matchExpressions` | [Array&lt;V1LabelSelectorRequirement&gt;](V1LabelSelectorRequirement.md)
`matchLabels` | { [key: string]: string; }

## Example

```typescript
import type { V1LabelSelector } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "matchExpressions": null,
  "matchLabels": null,
} satisfies V1LabelSelector

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1LabelSelector
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


