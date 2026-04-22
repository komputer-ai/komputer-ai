
# V1Affinity


## Properties

Name | Type
------------ | -------------
`nodeAffinity` | [V1NodeAffinity](V1NodeAffinity.md)
`podAffinity` | [V1PodAffinity](V1PodAffinity.md)
`podAntiAffinity` | [V1PodAntiAffinity](V1PodAntiAffinity.md)

## Example

```typescript
import type { V1Affinity } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "nodeAffinity": null,
  "podAffinity": null,
  "podAntiAffinity": null,
} satisfies V1Affinity

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1Affinity
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


