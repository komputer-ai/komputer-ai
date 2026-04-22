
# V1Volume


## Properties

Name | Type
------------ | -------------
`awsElasticBlockStore` | [V1AWSElasticBlockStoreVolumeSource](V1AWSElasticBlockStoreVolumeSource.md)
`azureDisk` | [V1AzureDiskVolumeSource](V1AzureDiskVolumeSource.md)
`azureFile` | [V1AzureFileVolumeSource](V1AzureFileVolumeSource.md)
`cephfs` | [V1CephFSVolumeSource](V1CephFSVolumeSource.md)
`cinder` | [V1CinderVolumeSource](V1CinderVolumeSource.md)
`configMap` | [V1ConfigMapVolumeSource](V1ConfigMapVolumeSource.md)
`csi` | [V1CSIVolumeSource](V1CSIVolumeSource.md)
`downwardAPI` | [V1DownwardAPIVolumeSource](V1DownwardAPIVolumeSource.md)
`emptyDir` | [V1EmptyDirVolumeSource](V1EmptyDirVolumeSource.md)
`ephemeral` | [V1EphemeralVolumeSource](V1EphemeralVolumeSource.md)
`fc` | [V1FCVolumeSource](V1FCVolumeSource.md)
`flexVolume` | [V1FlexVolumeSource](V1FlexVolumeSource.md)
`flocker` | [V1FlockerVolumeSource](V1FlockerVolumeSource.md)
`gcePersistentDisk` | [V1GCEPersistentDiskVolumeSource](V1GCEPersistentDiskVolumeSource.md)
`gitRepo` | [V1GitRepoVolumeSource](V1GitRepoVolumeSource.md)
`glusterfs` | [V1GlusterfsVolumeSource](V1GlusterfsVolumeSource.md)
`hostPath` | [V1HostPathVolumeSource](V1HostPathVolumeSource.md)
`image` | [V1ImageVolumeSource](V1ImageVolumeSource.md)
`iscsi` | [V1ISCSIVolumeSource](V1ISCSIVolumeSource.md)
`name` | string
`nfs` | [V1NFSVolumeSource](V1NFSVolumeSource.md)
`persistentVolumeClaim` | [V1PersistentVolumeClaimVolumeSource](V1PersistentVolumeClaimVolumeSource.md)
`photonPersistentDisk` | [V1PhotonPersistentDiskVolumeSource](V1PhotonPersistentDiskVolumeSource.md)
`portworxVolume` | [V1PortworxVolumeSource](V1PortworxVolumeSource.md)
`projected` | [V1ProjectedVolumeSource](V1ProjectedVolumeSource.md)
`quobyte` | [V1QuobyteVolumeSource](V1QuobyteVolumeSource.md)
`rbd` | [V1RBDVolumeSource](V1RBDVolumeSource.md)
`scaleIO` | [V1ScaleIOVolumeSource](V1ScaleIOVolumeSource.md)
`secret` | [V1SecretVolumeSource](V1SecretVolumeSource.md)
`storageos` | [V1StorageOSVolumeSource](V1StorageOSVolumeSource.md)
`vsphereVolume` | [V1VsphereVirtualDiskVolumeSource](V1VsphereVirtualDiskVolumeSource.md)

## Example

```typescript
import type { V1Volume } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "awsElasticBlockStore": null,
  "azureDisk": null,
  "azureFile": null,
  "cephfs": null,
  "cinder": null,
  "configMap": null,
  "csi": null,
  "downwardAPI": null,
  "emptyDir": null,
  "ephemeral": null,
  "fc": null,
  "flexVolume": null,
  "flocker": null,
  "gcePersistentDisk": null,
  "gitRepo": null,
  "glusterfs": null,
  "hostPath": null,
  "image": null,
  "iscsi": null,
  "name": null,
  "nfs": null,
  "persistentVolumeClaim": null,
  "photonPersistentDisk": null,
  "portworxVolume": null,
  "projected": null,
  "quobyte": null,
  "rbd": null,
  "scaleIO": null,
  "secret": null,
  "storageos": null,
  "vsphereVolume": null,
} satisfies V1Volume

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1Volume
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


