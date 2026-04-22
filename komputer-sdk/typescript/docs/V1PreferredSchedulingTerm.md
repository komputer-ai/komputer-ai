
# V1PreferredSchedulingTerm


## Properties

Name | Type
------------ | -------------
`preference` | [V1NodeSelectorTerm](V1NodeSelectorTerm.md)
`weight` | number

## Example

```typescript
import type { V1PreferredSchedulingTerm } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "preference": null,
  "weight": null,
} satisfies V1PreferredSchedulingTerm

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PreferredSchedulingTerm
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


