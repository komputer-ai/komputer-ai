
# V1PodAffinityTerm


## Properties

Name | Type
------------ | -------------
`labelSelector` | [V1LabelSelector](V1LabelSelector.md)
`matchLabelKeys` | Array&lt;string&gt;
`mismatchLabelKeys` | Array&lt;string&gt;
`namespaceSelector` | [V1LabelSelector](V1LabelSelector.md)
`namespaces` | Array&lt;string&gt;
`topologyKey` | string

## Example

```typescript
import type { V1PodAffinityTerm } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "labelSelector": null,
  "matchLabelKeys": null,
  "mismatchLabelKeys": null,
  "namespaceSelector": null,
  "namespaces": null,
  "topologyKey": null,
} satisfies V1PodAffinityTerm

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PodAffinityTerm
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


