package controller

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// s3SpecFromCluster reads the S3 session store config from the singleton
// KomputerConfig. Nil when there's none or it can't be read: the store is off.
func s3SpecFromCluster(ctx context.Context, c client.Client) *komputerv1alpha1.S3Spec {
	list := &komputerv1alpha1.KomputerConfigList{}
	if err := c.List(ctx, list); err != nil || len(list.Items) == 0 {
		return nil
	}
	return list.Items[0].Spec.S3
}

// sleepUnsafe reports whether sleeping the agent would lose its session.
// Sleeping deletes the pod, which wipes an ephemeral (emptyDir) workspace — fine
// only when the S3 session store holds the latest task's session, since the wake
// restores it. A PVC-backed workspace is always safe to sleep.
func sleepUnsafe(ephemeralWorkspace, s3On bool, agent *komputerv1alpha1.KomputerAgent) bool {
	return ephemeralWorkspace && !(s3On && komputerv1alpha1.LatestTaskSaved(agent))
}

// savedBeforeDestroy reports whether a transition that destroys the agent's last
// local copy of its session (Archive deleting the PVC) may run: always without
// the S3 session store, and with it only once the latest task is saved. An
// agent that can't save itself (an old pod) gets saved by maybeCopyOut, whose
// status write re-triggers the reconcile that then proceeds.
func savedBeforeDestroy(s3On bool, agent *komputerv1alpha1.KomputerAgent) bool {
	return !s3On || komputerv1alpha1.LatestTaskSaved(agent)
}
