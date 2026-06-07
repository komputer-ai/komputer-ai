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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisSpec defines the Redis connection settings.
type RedisSpec struct {
	// Address is the Redis host:port.
	Address string `json:"address"`
	// DB is the Redis database number.
	// +kubebuilder:default=0
	DB int `json:"db,omitempty"`
	// StreamPrefix is the Redis stream key prefix for agent events.
	// +kubebuilder:default="komputer-events"
	StreamPrefix string `json:"streamPrefix,omitempty"`
	// PasswordSecret references a Kubernetes Secret containing the Redis password.
	// +optional
	PasswordSecret *SecretKeyRef `json:"passwordSecret,omitempty"`
}

// KomputerConfigSpec defines the desired state of KomputerConfig.
type KomputerConfigSpec struct {
	// Redis defines the Redis connection settings for agent event streaming.
	Redis RedisSpec `json:"redis"`
	// APIURL is the internal URL of the komputer-api service.
	// Used by manager agents to create/manage sub-agents.
	// +kubebuilder:default="http://komputer-api.default.svc.cluster.local:8080"
	APIURL string `json:"apiURL,omitempty"`
}

// KomputerConfigStatus defines the observed state of KomputerConfig.
type KomputerConfigStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:annotations="argocd.argoproj.io/sync-options=ServerSideApply=true"
// +kubebuilder:metadata:annotations="argocd.argoproj.io/sync-wave=-1"
// +kubebuilder:resource:scope=Cluster

// KomputerConfig is the cluster-scoped singleton configuration for the komputer.ai platform.
type KomputerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KomputerConfigSpec   `json:"spec,omitempty"`
	Status KomputerConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KomputerConfigList contains a list of KomputerConfig.
type KomputerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KomputerConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KomputerConfig{}, &KomputerConfigList{})
}
