
# V1PodCertificateProjection


## Properties

Name | Type
------------ | -------------
`certificateChainPath` | string
`credentialBundlePath` | string
`keyPath` | string
`keyType` | string
`maxExpirationSeconds` | number
`signerName` | string
`userAnnotations` | { [key: string]: string; }

## Example

```typescript
import type { V1PodCertificateProjection } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "certificateChainPath": null,
  "credentialBundlePath": null,
  "keyPath": null,
  "keyType": null,
  "maxExpirationSeconds": null,
  "signerName": null,
  "userAnnotations": null,
} satisfies V1PodCertificateProjection

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PodCertificateProjection
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


