
# V1Container


## Properties

Name | Type
------------ | -------------
`args` | Array&lt;string&gt;
`command` | Array&lt;string&gt;
`env` | [Array&lt;V1EnvVar&gt;](V1EnvVar.md)
`envFrom` | [Array&lt;V1EnvFromSource&gt;](V1EnvFromSource.md)
`image` | string
`imagePullPolicy` | [V1PullPolicy](V1PullPolicy.md)
`lifecycle` | [V1Lifecycle](V1Lifecycle.md)
`livenessProbe` | [V1Probe](V1Probe.md)
`name` | string
`ports` | [Array&lt;V1ContainerPort&gt;](V1ContainerPort.md)
`readinessProbe` | [V1Probe](V1Probe.md)
`resizePolicy` | [Array&lt;V1ContainerResizePolicy&gt;](V1ContainerResizePolicy.md)
`resources` | [V1ResourceRequirements](V1ResourceRequirements.md)
`restartPolicy` | [V1ContainerRestartPolicy](V1ContainerRestartPolicy.md)
`restartPolicyRules` | [Array&lt;V1ContainerRestartRule&gt;](V1ContainerRestartRule.md)
`securityContext` | [V1SecurityContext](V1SecurityContext.md)
`startupProbe` | [V1Probe](V1Probe.md)
`stdin` | boolean
`stdinOnce` | boolean
`terminationMessagePath` | string
`terminationMessagePolicy` | [V1TerminationMessagePolicy](V1TerminationMessagePolicy.md)
`tty` | boolean
`volumeDevices` | [Array&lt;V1VolumeDevice&gt;](V1VolumeDevice.md)
`volumeMounts` | [Array&lt;V1VolumeMount&gt;](V1VolumeMount.md)
`workingDir` | string

## Example

```typescript
import type { V1Container } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "args": null,
  "command": null,
  "env": null,
  "envFrom": null,
  "image": null,
  "imagePullPolicy": null,
  "lifecycle": null,
  "livenessProbe": null,
  "name": null,
  "ports": null,
  "readinessProbe": null,
  "resizePolicy": null,
  "resources": null,
  "restartPolicy": null,
  "restartPolicyRules": null,
  "securityContext": null,
  "startupProbe": null,
  "stdin": null,
  "stdinOnce": null,
  "terminationMessagePath": null,
  "terminationMessagePolicy": null,
  "tty": null,
  "volumeDevices": null,
  "volumeMounts": null,
  "workingDir": null,
} satisfies V1Container

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1Container
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


