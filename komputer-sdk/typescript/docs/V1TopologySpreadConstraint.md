
# V1TopologySpreadConstraint


## Properties

Name | Type
------------ | -------------
`labelSelector` | [V1LabelSelector](V1LabelSelector.md)
`matchLabelKeys` | Array&lt;string&gt;
`maxSkew` | number
`minDomains` | number
`nodeAffinityPolicy` | [V1NodeInclusionPolicy](V1NodeInclusionPolicy.md)
`nodeTaintsPolicy` | [V1NodeInclusionPolicy](V1NodeInclusionPolicy.md)
`topologyKey` | string
`whenUnsatisfiable` | [V1UnsatisfiableConstraintAction](V1UnsatisfiableConstraintAction.md)

## Example

```typescript
import type { V1TopologySpreadConstraint } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "labelSelector": null,
  "matchLabelKeys": null,
  "maxSkew": null,
  "minDomains": null,
  "nodeAffinityPolicy": null,
  "nodeTaintsPolicy": null,
  "topologyKey": null,
  "whenUnsatisfiable": null,
} satisfies V1TopologySpreadConstraint

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1TopologySpreadConstraint
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


