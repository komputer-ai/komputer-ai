# V1VolumeProjection


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cluster_trust_bundle** | [**V1ClusterTrustBundleProjection**](V1ClusterTrustBundleProjection.md) | ClusterTrustBundle allows a pod to access the &#x60;.spec.trustBundle&#x60; field of ClusterTrustBundle objects in an auto-updating file.  Alpha, gated by the ClusterTrustBundleProjection feature gate.  ClusterTrustBundle objects can either be selected by name, or by the combination of signer name and a label selector.  Kubelet performs aggressive normalization of the PEM contents written into the pod filesystem.  Esoteric PEM features such as inter-block comments and block headers are stripped.  Certificates are deduplicated. The ordering of certificates within the file is arbitrary, and Kubelet may change the order over time.  +featureGate&#x3D;ClusterTrustBundleProjection +optional | [optional] 
**config_map** | [**V1ConfigMapProjection**](V1ConfigMapProjection.md) | configMap information about the configMap data to project +optional | [optional] 
**downward_api** | [**V1DownwardAPIProjection**](V1DownwardAPIProjection.md) | downwardAPI information about the downwardAPI data to project +optional | [optional] 
**pod_certificate** | [**V1PodCertificateProjection**](V1PodCertificateProjection.md) | Projects an auto-rotating credential bundle (private key and certificate chain) that the pod can use either as a TLS client or server.  Kubelet generates a private key and uses it to send a PodCertificateRequest to the named signer.  Once the signer approves the request and issues a certificate chain, Kubelet writes the key and certificate chain to the pod filesystem.  The pod does not start until certificates have been issued for each podCertificate projected volume source in its spec.  Kubelet will begin trying to rotate the certificate at the time indicated by the signer using the PodCertificateRequest.Status.BeginRefreshAt timestamp.  Kubelet can write a single file, indicated by the credentialBundlePath field, or separate files, indicated by the keyPath and certificateChainPath fields.  The credential bundle is a single file in PEM format.  The first PEM entry is the private key (in PKCS#8 format), and the remaining PEM entries are the certificate chain issued by the signer (typically, signers will return their certificate chain in leaf-to-root order).  Prefer using the credential bundle format, since your application code can read it atomically.  If you use keyPath and certificateChainPath, your application must make two separate file reads. If these coincide with a certificate rotation, it is possible that the private key and leaf certificate you read may not correspond to each other.  Your application will need to check for this condition, and re-read until they are consistent.  The named signer controls chooses the format of the certificate it issues; consult the signer implementation&#39;s documentation to learn how to use the certificates it issues.  +featureGate&#x3D;PodCertificateProjection +optional | [optional] 
**secret** | [**V1SecretProjection**](V1SecretProjection.md) | secret information about the secret data to project +optional | [optional] 
**service_account_token** | [**V1ServiceAccountTokenProjection**](V1ServiceAccountTokenProjection.md) | serviceAccountToken is information about the serviceAccountToken data to project +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_volume_projection import V1VolumeProjection

# TODO update the JSON string below
json = "{}"
# create an instance of V1VolumeProjection from a JSON string
v1_volume_projection_instance = V1VolumeProjection.from_json(json)
# print the JSON string representation of the object
print(V1VolumeProjection.to_json())

# convert the object into a dict
v1_volume_projection_dict = v1_volume_projection_instance.to_dict()
# create an instance of V1VolumeProjection from a dict
v1_volume_projection_from_dict = V1VolumeProjection.from_dict(v1_volume_projection_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


