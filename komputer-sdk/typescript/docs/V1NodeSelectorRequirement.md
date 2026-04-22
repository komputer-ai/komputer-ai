
# V1NodeSelectorRequirement


## Properties

Name | Type
------------ | -------------
`key` | string
`operator` | [V1NodeSelectorOperator](V1NodeSelectorOperator.md)
`values` | Array&lt;string&gt;

## Example

```typescript
import type { V1NodeSelectorRequirement } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "key": null,
  "operator": null,
  "values": null,
} satisfies V1NodeSelectorRequirement

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1NodeSelectorRequirement
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


