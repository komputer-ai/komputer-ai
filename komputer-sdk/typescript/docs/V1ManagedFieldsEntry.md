
# V1ManagedFieldsEntry


## Properties

Name | Type
------------ | -------------
`apiVersion` | string
`fieldsType` | string
`fieldsV1` | object
`manager` | string
`operation` | [V1ManagedFieldsOperationType](V1ManagedFieldsOperationType.md)
`subresource` | string
`time` | string

## Example

```typescript
import type { V1ManagedFieldsEntry } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "apiVersion": null,
  "fieldsType": null,
  "fieldsV1": null,
  "manager": null,
  "operation": null,
  "subresource": null,
  "time": null,
} satisfies V1ManagedFieldsEntry

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ManagedFieldsEntry
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


