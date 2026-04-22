
# V1NodeAffinity


## Properties

Name | Type
------------ | -------------
`preferredDuringSchedulingIgnoredDuringExecution` | [Array&lt;V1PreferredSchedulingTerm&gt;](V1PreferredSchedulingTerm.md)
`requiredDuringSchedulingIgnoredDuringExecution` | [V1NodeSelector](V1NodeSelector.md)

## Example

```typescript
import type { V1NodeAffinity } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "preferredDuringSchedulingIgnoredDuringExecution": null,
  "requiredDuringSchedulingIgnoredDuringExecution": null,
} satisfies V1NodeAffinity

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1NodeAffinity
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


