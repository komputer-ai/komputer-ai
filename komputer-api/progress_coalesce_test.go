package main

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// newCountingK8s returns a K8sClient over a fake client that counts status patches.
func newCountingK8s(t *testing.T, patches *int, objs ...client.Object) *K8sClient {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := komputerv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourcePatch: func(ctx context.Context, c client.Client, sub string, obj client.Object, p client.Patch, opts ...client.SubResourcePatchOption) error {
				*patches++
				return c.SubResource(sub).Patch(ctx, obj, p, opts...)
			},
		}).
		Build()
	return &K8sClient{client: c, defaultNamespace: "default"}
}

// TestPatchAgentTaskEvent_CoalescesProgress replays one task's event stream and
// checks that only the events that move the task state reach the API server.
func TestPatchAgentTaskEvent_CoalescesProgress(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"}}
	var patches int
	k8s := newCountingK8s(t, &patches, agent)
	ctx := context.Background()

	send := func(eventType string) {
		t.Helper()
		status, msg := mapEventToTaskStatus(AgentEvent{Type: eventType, Payload: map[string]interface{}{}})
		if err := k8s.PatchAgentTaskEvent(ctx, eventType, "default", "a1", status, msg, "", 0, 0, 0); err != nil {
			t.Fatalf("%s: %v", eventType, err)
		}
	}

	send("task_started") // write: "" -> InProgress
	for i := 0; i < 20; i++ {
		send("text")
		send("tool_call")
		send("tool_result")
		send("thinking")
	} // all coalesced: same status, inside the window
	send("compaction") // write: InProgress -> Compacting
	send("text")       // write: Compacting -> InProgress must not be dropped
	send("text")       // coalesced
	send("task_completed")

	if patches != 4 {
		t.Errorf("status patches = %d, want 4 (started, compaction, back to InProgress, completed)", patches)
	}
	got, err := k8s.GetAgent(ctx, "default", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status.TaskStatus != komputerv1alpha1.AgentTaskComplete {
		t.Errorf("TaskStatus = %q, want Complete", got.Status.TaskStatus)
	}
}

func TestProgressCoalescer_Window(t *testing.T) {
	var c progressCoalescer
	t0 := time.Now()
	c.record("ns", "a", "InProgress", t0)

	if !c.skip("ns", "a", "text", "InProgress", t0.Add(activityCoalesceWindow-time.Second)) {
		t.Error("progress event inside the window should be skipped")
	}
	if c.skip("ns", "a", "text", "InProgress", t0.Add(activityCoalesceWindow)) {
		t.Error("progress event at the window edge should be written, so LastActivityAt keeps moving")
	}
	if c.skip("ns", "a", "task_completed", "Complete", t0.Add(time.Second)) {
		t.Error("non-progress events are never skipped")
	}
	if c.skip("ns", "b", "text", "InProgress", t0.Add(time.Second)) {
		t.Error("an agent with no remembered write is never skipped")
	}
	c.forget("ns", "a")
	if c.skip("ns", "a", "text", "InProgress", t0.Add(time.Second)) {
		t.Error("forget must clear the remembered write")
	}
}

// TestPatchAgentTaskStatus_SkipsActivityOnlyWrite covers the second guard, which works
// across API replicas: an event that would only move LastActivityAt by less than the
// window is not written even when this replica's coalescer has no memory of the agent.
func TestPatchAgentTaskStatus_SkipsActivityOnlyWrite(t *testing.T) {
	recent := metav1.NewTime(time.Now().Add(-5 * time.Second))
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			TaskStatus:      komputerv1alpha1.AgentTaskInProgress,
			LastTaskMessage: "Thinking...",
			LastActivityAt:  &recent,
		},
	}
	var patches int
	k8s := newCountingK8s(t, &patches, agent)

	if err := k8s.PatchAgentTaskStatus(context.Background(), "default", "a1", "InProgress", "Thinking...", "", 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if patches != 0 {
		t.Errorf("patches = %d, want 0 for an activity-only change inside the window", patches)
	}

	if err := k8s.PatchAgentTaskStatus(context.Background(), "default", "a1", "InProgress", "Calling Bash", "", 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if patches != 1 {
		t.Errorf("patches = %d, want 1 once another field changes", patches)
	}
}
