
# V1ResourceRequirements


## Properties

Name | Type
------------ | -------------
`claims` | [Array&lt;K8sIoApiCoreV1ResourceClaim&gt;](K8sIoApiCoreV1ResourceClaim.md)
`limits` | [{ [key: string]: ResourceQuantity; }](ResourceQuantity.md)
`requests` | [{ [key: string]: ResourceQuantity; }](ResourceQuantity.md)

## Example

```typescript
import type { V1ResourceRequirements } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "claims": null,
  "limits": null,
  "requests": null,
} satisfies V1ResourceRequirements

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ResourceRequirements
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


