
# V1PodSecurityContext


## Properties

Name | Type
------------ | -------------
`appArmorProfile` | [V1AppArmorProfile](V1AppArmorProfile.md)
`fsGroup` | number
`fsGroupChangePolicy` | [V1PodFSGroupChangePolicy](V1PodFSGroupChangePolicy.md)
`runAsGroup` | number
`runAsNonRoot` | boolean
`runAsUser` | number
`seLinuxChangePolicy` | [V1PodSELinuxChangePolicy](V1PodSELinuxChangePolicy.md)
`seLinuxOptions` | [V1SELinuxOptions](V1SELinuxOptions.md)
`seccompProfile` | [V1SeccompProfile](V1SeccompProfile.md)
`supplementalGroups` | Array&lt;number&gt;
`supplementalGroupsPolicy` | [V1SupplementalGroupsPolicy](V1SupplementalGroupsPolicy.md)
`sysctls` | [Array&lt;V1Sysctl&gt;](V1Sysctl.md)
`windowsOptions` | [V1WindowsSecurityContextOptions](V1WindowsSecurityContextOptions.md)

## Example

```typescript
import type { V1PodSecurityContext } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "appArmorProfile": null,
  "fsGroup": null,
  "fsGroupChangePolicy": null,
  "runAsGroup": null,
  "runAsNonRoot": null,
  "runAsUser": null,
  "seLinuxChangePolicy": null,
  "seLinuxOptions": null,
  "seccompProfile": null,
  "supplementalGroups": null,
  "supplementalGroupsPolicy": null,
  "sysctls": null,
  "windowsOptions": null,
} satisfies V1PodSecurityContext

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PodSecurityContext
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


