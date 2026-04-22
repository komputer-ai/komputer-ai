
# V1AppArmorProfile


## Properties

Name | Type
------------ | -------------
`localhostProfile` | string
`type` | [V1AppArmorProfileType](V1AppArmorProfileType.md)

## Example

```typescript
import type { V1AppArmorProfile } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "localhostProfile": null,
  "type": null,
} satisfies V1AppArmorProfile

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1AppArmorProfile
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


