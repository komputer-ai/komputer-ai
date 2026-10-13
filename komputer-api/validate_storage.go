package main

import (
	"fmt"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// validateEphemeralStorage rejects configuring an effectively-ephemeral agent with
// Lifecycle=Sleep or a sleepTTL. The operator never sleeps an ephemeral (emptyDir)
// workspace — sleeping deletes the pod, which would wipe it — so letting such a
// request through would mean the user's Sleep/sleepTTL setting is silently ignored.
// Rejecting it here surfaces that mismatch immediately instead of at reconcile time.
func validateEphemeralStorage(ephemeral bool, lifecycle string, sleepTTL *metav1.Duration) error {
	if !ephemeral {
		return nil
	}
	if lifecycle == string(komputerv1alpha1.AgentLifecycleSleep) {
		return fmt.Errorf("storage.ephemeral cannot be combined with lifecycle: Sleep — sleeping deletes the pod, which would wipe the ephemeral workspace")
	}
	if sleepTTL != nil && sleepTTL.Duration > 0 {
		return fmt.Errorf("storage.ephemeral cannot be combined with sleepTTL — sleeping deletes the pod, which would wipe the ephemeral workspace")
	}
	return nil
}

// validateEphemeralImmutable rejects changing storage.ephemeral on an existing agent,
// in either direction. Flipping it after creation would either strand an existing PVC
// (persistent → ephemeral) or silently keep using one the user thought they'd dropped
// (ephemeral → persistent would require creating a brand new, empty PVC) — so it's
// immutable after creation. A patch that doesn't touch storage.ephemeral (nil, or no
// storage in the patch at all) is always allowed.
func validateEphemeralImmutable(before bool, newStorage *komputerv1alpha1.StorageSpec) error {
	if newStorage == nil || newStorage.Ephemeral == nil {
		return nil
	}
	if *newStorage.Ephemeral != before {
		return fmt.Errorf("storage.ephemeral is immutable after agent creation (currently %v)", before)
	}
	return nil
}
