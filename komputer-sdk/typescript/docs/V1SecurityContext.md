
# V1SecurityContext


## Properties

Name | Type
------------ | -------------
`allowPrivilegeEscalation` | boolean
`appArmorProfile` | [V1AppArmorProfile](V1AppArmorProfile.md)
`capabilities` | [V1Capabilities](V1Capabilities.md)
`privileged` | boolean
`procMount` | [V1ProcMountType](V1ProcMountType.md)
`readOnlyRootFilesystem` | boolean
`runAsGroup` | number
`runAsNonRoot` | boolean
`runAsUser` | number
`seLinuxOptions` | [V1SELinuxOptions](V1SELinuxOptions.md)
`seccompProfile` | [V1SeccompProfile](V1SeccompProfile.md)
`windowsOptions` | [V1WindowsSecurityContextOptions](V1WindowsSecurityContextOptions.md)

## Example

```typescript
import type { V1SecurityContext } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "allowPrivilegeEscalation": null,
  "appArmorProfile": null,
  "capabilities": null,
  "privileged": null,
  "procMount": null,
  "readOnlyRootFilesystem": null,
  "runAsGroup": null,
  "runAsNonRoot": null,
  "runAsUser": null,
  "seLinuxOptions": null,
  "seccompProfile": null,
  "windowsOptions": null,
} satisfies V1SecurityContext

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1SecurityContext
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


