package controller

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func TestIgnoreAgentProgressUpdates(t *testing.T) {
	t0 := metav1.NewTime(time.Now())
	t1 := metav1.NewTime(t0.Add(time.Second))
	base := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "ns", Generation: 3, Labels: map[string]string{"x": "1"}},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:           komputerv1alpha1.AgentPhaseRunning,
			TaskStatus:      komputerv1alpha1.AgentTaskInProgress,
			LastTaskMessage: "Calling Bash",
			LastActivityAt:  &t0,
		},
	}

	cases := []struct {
		name   string
		mutate func(a *komputerv1alpha1.KomputerAgent)
		want   bool // true = event passes the predicate
	}{
		{"message and activity only", func(a *komputerv1alpha1.KomputerAgent) {
			a.Status.LastTaskMessage = "Got result from Bash"
			a.Status.LastActivityAt = &t1
		}, false},
		{"task status change", func(a *komputerv1alpha1.KomputerAgent) {
			a.Status.TaskStatus = komputerv1alpha1.AgentTaskComplete
		}, true},
		{"status-only wake to Pending", func(a *komputerv1alpha1.KomputerAgent) {
			a.Status.Phase = komputerv1alpha1.AgentPhasePending
		}, true},
		{"spec change", func(a *komputerv1alpha1.KomputerAgent) { a.Generation++ }, true},
		{"label change", func(a *komputerv1alpha1.KomputerAgent) { a.Labels["x"] = "2" }, true},
		{"annotation change", func(a *komputerv1alpha1.KomputerAgent) {
			a.Annotations = map[string]string{komputerv1alpha1.AnnotationSleepRequested: "true"}
		}, true},
		{"deletion", func(a *komputerv1alpha1.KomputerAgent) { a.DeletionTimestamp = &t1 }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			newA := base.DeepCopy()
			tc.mutate(newA)
			got := ignoreAgentProgressUpdates.Update(event.UpdateEvent{ObjectOld: base, ObjectNew: newA})
			if got != tc.want {
				t.Errorf("predicate = %v, want %v", got, tc.want)
			}
		})
	}
}

// countStatusPatches builds a fake client that counts status-subresource writes,
// so the tests can assert a no-op reconcile sends nothing.
func countStatusPatches(t *testing.T, writes *int, objs ...client.Object) client.Client {
	t.Helper()
	return fake.NewClientBuilder().WithScheme(newTestScheme(t)).
		WithObjects(objs...).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}, &komputerv1alpha1.KomputerSchedule{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourcePatch: func(ctx context.Context, c client.Client, sub string, obj client.Object, p client.Patch, opts ...client.SubResourcePatchOption) error {
				*writes++
				return c.SubResource(sub).Patch(ctx, obj, p, opts...)
			},
			SubResourceUpdate: func(ctx context.Context, c client.Client, sub string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				*writes++
				return c.SubResource(sub).Update(ctx, obj, opts...)
			},
		}).Build()
}

func TestUpdateStatus_SkipsUnchanged(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "ns"},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseRunning, PodName: "a-pod", Port: 8000},
	}
	var writes int
	r := &KomputerAgentReconciler{Client: countStatusPatches(t, &writes, agent)}
	ctx := context.Background()

	got := &komputerv1alpha1.KomputerAgent{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(agent), got); err != nil {
		t.Fatal(err)
	}
	// What reconcileStatus sets on every pass for a running solo agent.
	if err := r.updateStatus(ctx, got, func(s *komputerv1alpha1.KomputerAgentStatus) {
		s.Phase = komputerv1alpha1.AgentPhaseRunning
		s.PodName = "a-pod"
		s.Port = 8000
	}); err != nil {
		t.Fatal(err)
	}
	if writes != 0 {
		t.Errorf("writes = %d, want 0 when nothing changed", writes)
	}

	if err := r.updateStatus(ctx, got, func(s *komputerv1alpha1.KomputerAgentStatus) {
		s.Phase = komputerv1alpha1.AgentPhaseSucceeded
	}); err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Errorf("writes = %d, want 1 after a real change", writes)
	}
}

func TestPatchScheduleStatus_SkipsUnchangedAndKeepsOptimisticLock(t *testing.T) {
	sched := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "ns"},
		Status:     komputerv1alpha1.KomputerScheduleStatus{Phase: komputerv1alpha1.SchedulePhaseActive, LastRunStatus: "InProgress", RunCount: 4},
	}
	var writes int
	r := &KomputerScheduleReconciler{Client: countStatusPatches(t, &writes, sched)}
	ctx := context.Background()

	got := &komputerv1alpha1.KomputerSchedule{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(sched), got); err != nil {
		t.Fatal(err)
	}
	base := got.DeepCopy()
	staleBase := got.DeepCopy() // a second reader's view, about to go stale

	// The 15s completion poll recomputes the same status: no write.
	if err := r.patchScheduleStatus(ctx, base, got); err != nil {
		t.Fatal(err)
	}
	if writes != 0 {
		t.Fatalf("writes = %d, want 0 for an unchanged status", writes)
	}

	got.Status.RunCount++
	if err := r.patchScheduleStatus(ctx, base, got); err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Fatalf("writes = %d, want 1 after a change", writes)
	}

	// A write computed from a stale copy must conflict, as Status().Update did,
	// rather than silently overwrite RunCount.
	stale := staleBase.DeepCopy()
	stale.Status.RunCount = 99
	if err := r.patchScheduleStatus(ctx, staleBase, stale); err == nil {
		t.Error("expected a conflict for a patch built on a stale resourceVersion")
	}
}
