package main

import (
	"context"
	"fmt"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// forwardWithLifecycleChange forwards a task to a running agent, applying a
// lifecycle change carried on the same request.
//
// Every lifecycle transition (Sleep, AutoDelete, Archive) fires on a finished
// task's Complete/Error status while the pod is up. Patching the lifecycle after
// the forward let the operator act on the PREVIOUS task's status before the new
// one reported in — killing the new task, and for Archive deleting its
// workspace. So for a new task the old status is cleared first, then the
// lifecycle is patched, then the task is forwarded. A steer (task in progress)
// is never at risk, since InProgress triggers nothing.
//
// If the forward fails, the lifecycle is reverted before the status is put back
// (in that order, so the stale status never meets the new lifecycle), and the
// status is only restored if nothing wrote a newer one in the meantime — the
// agent may have taken the task even though the forward reported an error.
func forwardWithLifecycleChange(ctx context.Context, k8s *K8sClient, existing *komputerv1alpha1.KomputerAgent, lifecycle string, forward func() (int64, error)) (int64, error) {
	ns, name := existing.Namespace, existing.Name
	if lifecycle == "" || lifecycle == string(existing.Spec.Lifecycle) {
		return forward()
	}

	prevStatus := existing.Status.TaskStatus
	cleared := false
	if !agentTaskInProgress(prevStatus) && prevStatus != "" {
		var err error
		if cleared, err = k8s.SwapAgentTaskStatus(ctx, ns, name, prevStatus, ""); err != nil {
			return 0, fmt.Errorf("failed to prepare lifecycle change: %w", err)
		}
	}
	if err := k8s.PatchAgentLifecycle(ctx, ns, name, lifecycle); err != nil {
		Logger.Warnw("failed to patch lifecycle", "agent_name", name, "error", err)
	}

	cw, err := forward()
	if err != nil {
		if revertErr := k8s.PatchAgentLifecycle(ctx, ns, name, string(existing.Spec.Lifecycle)); revertErr != nil {
			Logger.Warnw("failed to revert lifecycle after failed forward", "agent_name", name, "error", revertErr)
		}
		if cleared {
			if _, restoreErr := k8s.SwapAgentTaskStatus(ctx, ns, name, "", prevStatus); restoreErr != nil {
				Logger.Warnw("failed to restore task status after failed forward", "agent_name", name, "error", restoreErr)
			}
		}
		return 0, err
	}
	return cw, nil
}
