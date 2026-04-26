// komputer-operator/api/v1alpha1/komputersquad_types.go
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KomputerSquadMemberRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"` // defaults to squad's namespace
}

// +kubebuilder:validation:XValidation:rule="(has(self.ref) ? 1 : 0) + (has(self.spec) ? 1 : 0) == 1",message="exactly one of ref or spec must be set"
type KomputerSquadMember struct {
	// Exactly one of Ref or Spec must be set.
	Ref  *KomputerSquadMemberRef `json:"ref,omitempty"`
	Spec *KomputerAgentSpec      `json:"spec,omitempty"`
}

type KomputerSquadSpec struct {
	// Members of the squad. The operator co-locates them in a single Pod.
	// +kubebuilder:validation:MinItems=1
	Members []KomputerSquadMember `json:"members"`

	// OrphanTTL is how long to keep an empty squad before deleting it.
	// Defaults to "10m".
	// +optional
	OrphanTTL *metav1.Duration `json:"orphanTTL,omitempty"`
}

type KomputerSquadMemberStatus struct {
	Name  string `json:"name"`
	Ready bool   `json:"ready"`
	// TaskStatus is owned by the API worker (mirrors the agent's TaskStatus).
	// The squad controller only writes Name and Ready; TaskStatus is populated
	// by the API worker as it reconciles the underlying agent's status.
	TaskStatus string `json:"taskStatus,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Running;Orphaned;Failed
type KomputerSquadPhase string

const (
	SquadPhasePending  KomputerSquadPhase = "Pending"
	SquadPhaseRunning  KomputerSquadPhase = "Running"
	SquadPhaseOrphaned KomputerSquadPhase = "Orphaned"
	SquadPhaseFailed   KomputerSquadPhase = "Failed"
)

type KomputerSquadStatus struct {
	Phase         KomputerSquadPhase          `json:"phase,omitempty"`
	PodName       string                      `json:"podName,omitempty"`
	Members       []KomputerSquadMemberStatus `json:"members,omitempty"`
	OrphanedSince *metav1.Time                `json:"orphanedSince,omitempty"`
	Message       string                      `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ks
type KomputerSquad struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KomputerSquadSpec   `json:"spec,omitempty"`
	Status KomputerSquadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KomputerSquadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KomputerSquad `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KomputerSquad{}, &KomputerSquadList{})
}
