
# V1ObjectMeta


## Properties

Name | Type
------------ | -------------
`annotations` | { [key: string]: string; }
`creationTimestamp` | string
`deletionGracePeriodSeconds` | number
`deletionTimestamp` | string
`finalizers` | Array&lt;string&gt;
`generateName` | string
`generation` | number
`labels` | { [key: string]: string; }
`managedFields` | [Array&lt;V1ManagedFieldsEntry&gt;](V1ManagedFieldsEntry.md)
`name` | string
`namespace` | string
`ownerReferences` | [Array&lt;V1OwnerReference&gt;](V1OwnerReference.md)
`resourceVersion` | string
`selfLink` | string
`uid` | string

## Example

```typescript
import type { V1ObjectMeta } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "annotations": null,
  "creationTimestamp": null,
  "deletionGracePeriodSeconds": null,
  "deletionTimestamp": null,
  "finalizers": null,
  "generateName": null,
  "generation": null,
  "labels": null,
  "managedFields": null,
  "name": null,
  "namespace": null,
  "ownerReferences": null,
  "resourceVersion": null,
  "selfLink": null,
  "uid": null,
} satisfies V1ObjectMeta

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1ObjectMeta
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


