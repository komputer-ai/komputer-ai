
# V1PodSpec


## Properties

Name | Type
------------ | -------------
`activeDeadlineSeconds` | number
`affinity` | [V1Affinity](V1Affinity.md)
`automountServiceAccountToken` | boolean
`containers` | [Array&lt;V1Container&gt;](V1Container.md)
`dnsConfig` | [V1PodDNSConfig](V1PodDNSConfig.md)
`dnsPolicy` | [V1DNSPolicy](V1DNSPolicy.md)
`enableServiceLinks` | boolean
`ephemeralContainers` | [Array&lt;V1EphemeralContainer&gt;](V1EphemeralContainer.md)
`hostAliases` | [Array&lt;V1HostAlias&gt;](V1HostAlias.md)
`hostIPC` | boolean
`hostNetwork` | boolean
`hostPID` | boolean
`hostUsers` | boolean
`hostname` | string
`hostnameOverride` | string
`imagePullSecrets` | [Array&lt;V1LocalObjectReference&gt;](V1LocalObjectReference.md)
`initContainers` | [Array&lt;V1Container&gt;](V1Container.md)
`nodeName` | string
`nodeSelector` | { [key: string]: string; }
`os` | [V1PodOS](V1PodOS.md)
`overhead` | [{ [key: string]: ResourceQuantity; }](ResourceQuantity.md)
`preemptionPolicy` | [V1PreemptionPolicy](V1PreemptionPolicy.md)
`priority` | number
`priorityClassName` | string
`readinessGates` | [Array&lt;V1PodReadinessGate&gt;](V1PodReadinessGate.md)
`resourceClaims` | [Array&lt;V1PodResourceClaim&gt;](V1PodResourceClaim.md)
`resources` | [V1ResourceRequirements](V1ResourceRequirements.md)
`restartPolicy` | [V1RestartPolicy](V1RestartPolicy.md)
`runtimeClassName` | string
`schedulerName` | string
`schedulingGates` | [Array&lt;V1PodSchedulingGate&gt;](V1PodSchedulingGate.md)
`securityContext` | [V1PodSecurityContext](V1PodSecurityContext.md)
`serviceAccount` | string
`serviceAccountName` | string
`setHostnameAsFQDN` | boolean
`shareProcessNamespace` | boolean
`subdomain` | string
`terminationGracePeriodSeconds` | number
`tolerations` | [Array&lt;V1Toleration&gt;](V1Toleration.md)
`topologySpreadConstraints` | [Array&lt;V1TopologySpreadConstraint&gt;](V1TopologySpreadConstraint.md)
`volumes` | [Array&lt;V1Volume&gt;](V1Volume.md)
`workloadRef` | [V1WorkloadReference](V1WorkloadReference.md)

## Example

```typescript
import type { V1PodSpec } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "activeDeadlineSeconds": null,
  "affinity": null,
  "automountServiceAccountToken": null,
  "containers": null,
  "dnsConfig": null,
  "dnsPolicy": null,
  "enableServiceLinks": null,
  "ephemeralContainers": null,
  "hostAliases": null,
  "hostIPC": null,
  "hostNetwork": null,
  "hostPID": null,
  "hostUsers": null,
  "hostname": null,
  "hostnameOverride": null,
  "imagePullSecrets": null,
  "initContainers": null,
  "nodeName": null,
  "nodeSelector": null,
  "os": null,
  "overhead": null,
  "preemptionPolicy": null,
  "priority": null,
  "priorityClassName": null,
  "readinessGates": null,
  "resourceClaims": null,
  "resources": null,
  "restartPolicy": null,
  "runtimeClassName": null,
  "schedulerName": null,
  "schedulingGates": null,
  "securityContext": null,
  "serviceAccount": null,
  "serviceAccountName": null,
  "setHostnameAsFQDN": null,
  "shareProcessNamespace": null,
  "subdomain": null,
  "terminationGracePeriodSeconds": null,
  "tolerations": null,
  "topologySpreadConstraints": null,
  "volumes": null,
  "workloadRef": null,
} satisfies V1PodSpec

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PodSpec
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


