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

package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestResolveSquadMemberVolume_Ephemeral verifies that a member with no existing PVC,
// configured ephemeral directly on its own spec, gets an emptyDir volume.
func TestResolveSquadMemberVolume_Ephemeral(t *testing.T) {
	ctx := context.Background()
	agent := squadMember("ephemeral-member", time.Minute, func(a *komputerv1alpha1.KomputerAgent) {
		a.Spec.Storage = &komputerv1alpha1.StorageSpec{Ephemeral: boolPtr(true)}
	})
	r := newSquadTTLReconciler(t, agent)

	vol := r.resolveSquadMemberVolume(ctx, agent)

	if vol.Name != "ephemeral-member-workspace" {
		t.Errorf("volume name = %q, want ephemeral-member-workspace", vol.Name)
	}
	if vol.EmptyDir == nil {
		t.Fatal("expected an EmptyDir volume source for an ephemeral member with no PVC")
	}
	if vol.PersistentVolumeClaim != nil {
		t.Error("expected no PersistentVolumeClaim source for an ephemeral member")
	}
}

// TestResolveSquadMemberVolume_NonEphemeral verifies that a member without ephemeral
// set gets the usual PVC-backed volume (today's behavior, unchanged).
func TestResolveSquadMemberVolume_NonEphemeral(t *testing.T) {
	ctx := context.Background()
	agent := squadMember("persistent-member", time.Minute)
	r := newSquadTTLReconciler(t, agent)

	vol := r.resolveSquadMemberVolume(ctx, agent)

	if vol.PersistentVolumeClaim == nil {
		t.Fatal("expected a PersistentVolumeClaim volume source for a non-ephemeral member")
	}
	if vol.PersistentVolumeClaim.ClaimName != "persistent-member-pvc" {
		t.Errorf("claimName = %q, want persistent-member-pvc", vol.PersistentVolumeClaim.ClaimName)
	}
	if vol.EmptyDir != nil {
		t.Error("expected no EmptyDir source for a non-ephemeral member")
	}
}

// TestResolveSquadMemberVolume_ExistingPVCKept verifies the data-safety override: a
// member marked ephemeral that already has a PVC (status.pvcName set) keeps mounting
// it rather than switching to an emptyDir and orphaning its data.
func TestResolveSquadMemberVolume_ExistingPVCKept(t *testing.T) {
	ctx := context.Background()
	agent := squadMember("flipped-member", time.Minute, func(a *komputerv1alpha1.KomputerAgent) {
		a.Spec.Storage = &komputerv1alpha1.StorageSpec{Ephemeral: boolPtr(true)}
		a.Status.PvcName = "flipped-member-pvc"
	})
	r := newSquadTTLReconciler(t, agent)

	vol := r.resolveSquadMemberVolume(ctx, agent)

	if vol.PersistentVolumeClaim == nil {
		t.Fatal("expected the existing PVC to still be mounted despite ephemeral=true")
	}
	if vol.PersistentVolumeClaim.ClaimName != "flipped-member-pvc" {
		t.Errorf("claimName = %q, want flipped-member-pvc", vol.PersistentVolumeClaim.ClaimName)
	}
}

// TestResolveSquadMemberVolume_MixedSquad verifies that a squad with both ephemeral
// and persistent members resolves each member's volume independently.
func TestResolveSquadMemberVolume_MixedSquad(t *testing.T) {
	ctx := context.Background()
	ephemeralMember := squadMember("mix-ephemeral", time.Minute, func(a *komputerv1alpha1.KomputerAgent) {
		a.Spec.Storage = &komputerv1alpha1.StorageSpec{Ephemeral: boolPtr(true)}
	})
	persistentMember := squadMember("mix-persistent", time.Minute)
	r := newSquadTTLReconciler(t, ephemeralMember, persistentMember)

	ev := r.resolveSquadMemberVolume(ctx, ephemeralMember)
	pv := r.resolveSquadMemberVolume(ctx, persistentMember)

	if ev.EmptyDir == nil {
		t.Error("expected the ephemeral member to get an EmptyDir volume")
	}
	if pv.PersistentVolumeClaim == nil {
		t.Error("expected the persistent member to keep its PVC volume")
	}
}

// TestResolveSquadMemberVolume_TemplateEphemeral verifies a member inherits
// ephemeral=true from its template when its own spec doesn't set it.
func TestResolveSquadMemberVolume_TemplateEphemeral(t *testing.T) {
	ctx := context.Background()
	tpl := &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
			Storage: komputerv1alpha1.StorageSpec{Ephemeral: boolPtr(true), Size: "2Gi"},
		},
	}
	agent := squadMember("tpl-ephemeral", time.Minute)
	r := newSquadTTLReconciler(t, tpl, agent)

	vol := r.resolveSquadMemberVolume(ctx, agent)

	if vol.EmptyDir == nil {
		t.Fatal("expected the member to inherit ephemeral=true from its template")
	}
	if vol.EmptyDir.SizeLimit == nil || vol.EmptyDir.SizeLimit.String() != "2Gi" {
		t.Errorf("sizeLimit = %v, want 2Gi from the template's storage.size", vol.EmptyDir.SizeLimit)
	}
}

// TestEnsureMemberPVCs_SkipsEphemeralMember verifies ensureMemberPVCs creates no PVC
// for an ephemeral member with none already, while still creating one for a
// non-ephemeral sibling in the same squad.
func TestEnsureMemberPVCs_SkipsEphemeralMember(t *testing.T) {
	ctx := context.Background()
	ephemeralMember := squadMember("skip-pvc-member", time.Minute, func(a *komputerv1alpha1.KomputerAgent) {
		a.Spec.Storage = &komputerv1alpha1.StorageSpec{Ephemeral: boolPtr(true)}
	})
	persistentMember := squadMember("keep-pvc-member", time.Minute)
	squad := testSquad("skip-pvc-member", "keep-pvc-member")
	r := newSquadTTLReconciler(t, squad, ephemeralMember, persistentMember)

	if err := r.ensureMemberPVCs(ctx, squad, []*komputerv1alpha1.KomputerAgent{ephemeralMember, persistentMember}); err != nil {
		t.Fatalf("ensureMemberPVCs: %v", err)
	}

	var pvc corev1.PersistentVolumeClaim
	if err := r.Get(ctx, types.NamespacedName{Name: "skip-pvc-member-pvc", Namespace: "default"}, &pvc); err == nil {
		t.Fatal("expected no PVC to be created for the ephemeral member")
	}
	if err := r.Get(ctx, types.NamespacedName{Name: "keep-pvc-member-pvc", Namespace: "default"}, &pvc); err != nil {
		t.Fatalf("expected a PVC for the non-ephemeral member: %v", err)
	}

	var gotEphemeral komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "skip-pvc-member", Namespace: "default"}, &gotEphemeral); err != nil {
		t.Fatalf("get ephemeral member: %v", err)
	}
	if gotEphemeral.Status.PvcName != "" {
		t.Errorf("PvcName = %q, want empty for an ephemeral member with no PVC", gotEphemeral.Status.PvcName)
	}
}

// TestEnsureMemberPVCs_KeepsExistingPVCForEphemeralMember verifies the data-safety
// override at the ensureMemberPVCs level: an ephemeral member whose PVC already
// exists is left alone (not deleted), and its status.pvcName stays populated.
func TestEnsureMemberPVCs_KeepsExistingPVCForEphemeralMember(t *testing.T) {
	ctx := context.Background()
	agent := squadMember("already-has-pvc", time.Minute, func(a *komputerv1alpha1.KomputerAgent) {
		a.Spec.Storage = &komputerv1alpha1.StorageSpec{Ephemeral: boolPtr(true)}
		a.Status.PvcName = "already-has-pvc-pvc"
	})
	existingPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "already-has-pvc-pvc", Namespace: "default"},
	}
	squad := testSquad("already-has-pvc")
	r := newSquadTTLReconciler(t, squad, agent, existingPVC)

	if err := r.ensureMemberPVCs(ctx, squad, []*komputerv1alpha1.KomputerAgent{agent}); err != nil {
		t.Fatalf("ensureMemberPVCs: %v", err)
	}

	var pvc corev1.PersistentVolumeClaim
	if err := r.Get(ctx, types.NamespacedName{Name: "already-has-pvc-pvc", Namespace: "default"}, &pvc); err != nil {
		t.Fatalf("expected the existing PVC to survive: %v", err)
	}
}
