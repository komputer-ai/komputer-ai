package main

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// The save time is the worker's clock, not the agent pod's: TaskStartedAt is
// stamped by the worker when it processes task_started, so a lagging worker
// (or a skewed pod clock) must not make a fresh save look older than its task.
func TestPatchAgentTaskEvent_SaveTimeUsesWorkerClock(t *testing.T) {
	agent := storeAgent("")
	k8s := newFakeK8sWithStatusSubresource(t, agent)
	skewed := agent.Status.TaskStartedAt.Add(-time.Hour) // the pod's clock is an hour behind
	store := &SessionStoreResult{OK: true, At: skewed, Bytes: 10}
	if err := k8s.PatchAgentTaskEvent(context.Background(), "task_completed", "default", "a1", "Complete", "Task completed", "s1", 0, 0, 0, store); err != nil {
		t.Fatal(err)
	}
	if !komputerv1alpha1.LatestTaskSaved(getStoredAgent(t, k8s)) {
		t.Error("a save recorded just now must count for the task that just ended")
	}
}

// A failed copy-out from before the agent saved itself successfully must not
// keep reporting "save failed" forever.
func TestAgentResponse_StaleCopyOutFailureDoesNotOverrideNewerSave(t *testing.T) {
	failedAt := metav1.NewTime(time.Now().Add(-time.Hour))
	savedAt := metav1.NewTime(time.Now())
	agent := storeAgent("")
	agent.Status.SessionStore = &komputerv1alpha1.SessionStoreStatus{OK: true, LastSavedAt: &savedAt, LastAttemptAt: &savedAt}
	agent.Status.Conditions = []metav1.Condition{{
		Type: komputerv1alpha1.ConditionSessionSaved, Status: metav1.ConditionFalse, Reason: "SaveFailed",
		Message: "Couldn't save the session to S3: boom", LastTransitionTime: failedAt,
	}}
	var resp AgentResponse
	fillAgentTTL(&resp, agent)
	if resp.SessionStore == nil || !resp.SessionStore.OK {
		t.Fatalf("sessionStore = %+v, want ok (the newer save wins)", resp.SessionStore)
	}
}
