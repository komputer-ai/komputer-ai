
# V1VolumeResourceRequirements


## Properties

Name | Type
------------ | -------------
`limits` | [{ [key: string]: ResourceQuantity; }](ResourceQuantity.md)
`requests` | [{ [key: string]: ResourceQuantity; }](ResourceQuantity.md)

## Example

```typescript
import type { V1VolumeResourceRequirements } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "limits": null,
  "requests": null,
} satisfies V1VolumeResourceRequirements

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1VolumeResourceRequirements
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


