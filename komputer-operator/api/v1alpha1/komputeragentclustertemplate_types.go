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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:metadata:annotations="argocd.argoproj.io/sync-options=ServerSideApply=true"
// +kubebuilder:metadata:annotations="argocd.argoproj.io/sync-wave=-1"
// +kubebuilder:resource:scope=Cluster

// KomputerAgentClusterTemplate is a cluster-scoped agent template shared across all namespaces.
// Agents reference it via templateRef. If a namespaced KomputerAgentTemplate with the same
// name exists, it takes precedence over the cluster template.
type KomputerAgentClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KomputerAgentTemplateSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// KomputerAgentClusterTemplateList contains a list of KomputerAgentClusterTemplate.
type KomputerAgentClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KomputerAgentClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KomputerAgentClusterTemplate{}, &KomputerAgentClusterTemplateList{})
}
