/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageSpec defines PVC settings for agent workspaces.
type StorageSpec struct {
	// Size is the PVC storage size (e.g. "5Gi").
	// +kubebuilder:default="5Gi"
	Size string `json:"size,omitempty"`
	// StorageClassName is the optional storage class name.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// KomputerAgentTemplateSpec defines the desired state of KomputerAgentTemplate.
type KomputerAgentTemplateSpec struct {
	// PodSpec is a full corev1.PodSpec passthrough for the agent pod.
	PodSpec corev1.PodSpec `json:"podSpec"`
	// Storage defines the PVC settings for agent workspaces.
	// +optional
	Storage StorageSpec `json:"storage,omitempty"`
	// MaxConcurrentAgents caps how many KomputerAgents using this template can be
	// in Phase=Running at once (per namespace). Excess agents enter Phase=Queued
	// and are admitted by Priority (higher first; ties by creationTimestamp).
	// 0 (default) disables the cap.
	// +kubebuilder:default=0
	// +optional
	MaxConcurrentAgents int32 `json:"maxConcurrentAgents,omitempty"`
	// AnthropicKeySecretRef is the absolute reference to the Anthropic API
	// key secret. The operator mirrors this secret into every agent namespace
	// and injects the ANTHROPIC_API_KEY env var into the pod automatically —
	// users MUST NOT add ANTHROPIC_API_KEY to PodSpec env themselves; if they
	// do, the pod-builder strips it before injecting its own copy.
	//
	// Required. The default cluster template shipped by helm always sets this
	// from chart values.
	AnthropicKeySecretRef SecretKeyRef `json:"anthropicKeySecretRef"`
}

// KomputerAgentTemplateStatus defines the observed state of KomputerAgentTemplate.
type KomputerAgentTemplateStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KomputerAgentTemplate is the Schema for the komputeragenttemplates API.
type KomputerAgentTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KomputerAgentTemplateSpec   `json:"spec,omitempty"`
	Status KomputerAgentTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KomputerAgentTemplateList contains a list of KomputerAgentTemplate.
type KomputerAgentTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KomputerAgentTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KomputerAgentTemplate{}, &KomputerAgentTemplateList{})
}
