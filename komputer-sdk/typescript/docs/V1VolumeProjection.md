
# V1VolumeProjection


## Properties

Name | Type
------------ | -------------
`clusterTrustBundle` | [V1ClusterTrustBundleProjection](V1ClusterTrustBundleProjection.md)
`configMap` | [V1ConfigMapProjection](V1ConfigMapProjection.md)
`downwardAPI` | [V1DownwardAPIProjection](V1DownwardAPIProjection.md)
`podCertificate` | [V1PodCertificateProjection](V1PodCertificateProjection.md)
`secret` | [V1SecretProjection](V1SecretProjection.md)
`serviceAccountToken` | [V1ServiceAccountTokenProjection](V1ServiceAccountTokenProjection.md)

## Example

```typescript
import type { V1VolumeProjection } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "clusterTrustBundle": null,
  "configMap": null,
  "downwardAPI": null,
  "podCertificate": null,
  "secret": null,
  "serviceAccountToken": null,
} satisfies V1VolumeProjection

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1VolumeProjection
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


