
# V1PodAffinity


## Properties

Name | Type
------------ | -------------
`preferredDuringSchedulingIgnoredDuringExecution` | [Array&lt;V1WeightedPodAffinityTerm&gt;](V1WeightedPodAffinityTerm.md)
`requiredDuringSchedulingIgnoredDuringExecution` | [Array&lt;V1PodAffinityTerm&gt;](V1PodAffinityTerm.md)

## Example

```typescript
import type { V1PodAffinity } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "preferredDuringSchedulingIgnoredDuringExecution": null,
  "requiredDuringSchedulingIgnoredDuringExecution": null,
} satisfies V1PodAffinity

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PodAffinity
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


