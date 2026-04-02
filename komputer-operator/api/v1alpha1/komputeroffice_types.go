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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// OfficeMemberStatus tracks the status of a single agent in the office.
type OfficeMemberStatus struct {
	Name            string `json:"name"`
	Role            string `json:"role"`
	TaskStatus      string `json:"taskStatus,omitempty"`
	LastTaskCostUSD string `json:"lastTaskCostUSD,omitempty"`
}

type KomputerOfficePhase string

const (
	OfficePhaseInProgress KomputerOfficePhase = "InProgress"
	OfficePhaseComplete   KomputerOfficePhase = "Complete"
	OfficePhaseError      KomputerOfficePhase = "Error"
)

type KomputerOfficeSpec struct {
	// Manager is the name of the manager agent that leads this office.
	Manager string `json:"manager"`
}

type KomputerOfficeStatus struct {
	Phase           KomputerOfficePhase  `json:"phase,omitempty"`
	Manager         OfficeMemberStatus   `json:"manager,omitempty"`
	Members         []OfficeMemberStatus `json:"members,omitempty"`
	TotalAgents     int                  `json:"totalAgents,omitempty"`
	ActiveAgents    int                  `json:"activeAgents,omitempty"`
	CompletedAgents int                  `json:"completedAgents,omitempty"`
	TotalCostUSD    string               `json:"totalCostUSD,omitempty"`
	// TotalTokens is the cumulative number of tokens consumed by all agents in this office.
	// +optional
	TotalTokens  int64        `json:"totalTokens,omitempty"`
	CreatedAt    *metav1.Time `json:"createdAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Manager",type=string,JSONPath=`.spec.manager`
// +kubebuilder:printcolumn:name="Agents",type=integer,JSONPath=`.status.totalAgents`
// +kubebuilder:printcolumn:name="Active",type=integer,JSONPath=`.status.activeAgents`
// +kubebuilder:printcolumn:name="Cost",type=string,JSONPath=`.status.totalCostUSD`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type KomputerOffice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KomputerOfficeSpec   `json:"spec,omitempty"`
	Status            KomputerOfficeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KomputerOfficeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KomputerOffice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KomputerOffice{}, &KomputerOfficeList{})
}
