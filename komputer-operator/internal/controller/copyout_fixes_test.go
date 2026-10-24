package controller

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// A pod with the S3 env normally saves itself, but not on every path (a cancel
// that loses the race, a crash) and its save can fail: when its ended task is
// still unsaved, the operator saves it from outside too.
func TestNeedsCopyOut_SelfSavingPodLeftUnsaved(t *testing.T) {
	if !needsCopyOut(copyOutAgent(), copyOutPod(true), true) {
		t.Error("an idle pod whose ended task is unsaved should be copied out, S3 env or not")
	}
}

// A failing copy-out must not retry on every reconcile (each failure writes a
// new status message, which triggers the next reconcile immediately).
func TestMaybeCopyOut_FailureBacksOff(t *testing.T) {
	ex := &fakeExecer{err: errors.New("boom")}
	agent := copyOutAgent()
	pod := copyOutPod(false)
	r, c := copyOutReconciler(t, ex, &fakeUploader{}, agent, pod)
	config := &komputerv1alpha1.KomputerConfig{Spec: komputerv1alpha1.KomputerConfigSpec{S3: copyOutS3}}

	for i := 0; i < 3; i++ {
		var cur komputerv1alpha1.KomputerAgent
		_ = c.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &cur)
		r.maybeCopyOut(context.Background(), &cur, pod, config)
	}
	if len(ex.calls) != 1 {
		t.Errorf("exec calls = %d, want 1 (later attempts wait out the backoff)", len(ex.calls))
	}
}

// No session files at all (nothing ever ran there) is nothing to save, not an error.
func TestCopyOutSession_EmptyOutput_NothingToSave(t *testing.T) {
	up := &fakeUploader{}
	agent := copyOutAgent()
	r, c := copyOutReconciler(t, &fakeExecer{out: nil}, up, agent)
	if err := r.copyOutSession(context.Background(), agent, copyOutS3, "a1-pod", "agent"); err != nil {
		t.Fatalf("empty output should be a no-op, got %v", err)
	}
	if up.n != 0 {
		t.Errorf("uploads = %d, want none (never overwrite a saved session with nothing)", up.n)
	}
	var got komputerv1alpha1.KomputerAgent
	_ = c.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got)
	if !komputerv1alpha1.LatestTaskSaved(&got) {
		t.Error("nothing to save counts as saved")
	}
}

// blockingExecer never returns until its context is done.
type blockingExecer struct{}

func (blockingExecer) ExecStream(ctx context.Context, ns, pod, container string, cmd []string, stdout io.Writer) error {
	<-ctx.Done()
	return ctx.Err()
}

// A hung exec or a stalled S3 endpoint must not block the (single) reconcile
// worker forever.
func TestCopyOutSession_TimesOut(t *testing.T) {
	orig := copyOutTimeout
	copyOutTimeout = 50 * time.Millisecond
	t.Cleanup(func() { copyOutTimeout = orig })

	agent := copyOutAgent()
	r, _ := copyOutReconciler(t, blockingExecer{}, &fakeUploader{}, agent)
	done := make(chan error, 1)
	go func() { done <- r.copyOutSession(context.Background(), agent, copyOutS3, "a1-pod", "agent") }()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a timed-out copy-out must report an error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("copy-out did not time out")
	}
}

// A task stuck "InProgress" on an agent whose pod is gone (OOM, eviction) can't
// be cancelled; the delete must go on to save from the PVC instead of waiting forever.
func TestSessionFinalizer_StuckInProgressNoPod_SavesFromPVC(t *testing.T) {
	agent := deletingAgent(func(a *komputerv1alpha1.KomputerAgent) {
		a.Status.TaskStatus = komputerv1alpha1.AgentTaskInProgress
		a.Status.PodName = ""
	})
	r, c := saveFirstReconciler(t, &fakeExecer{}, &fakeUploader{}, agent, agentPVC(false), agentTemplate(), s3KomputerConfig())
	r.MaxConcurrentSavers = 4
	if _, err := r.handleSessionFinalizer(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	if len(saverPods(t, c)) != 1 {
		t.Error("with no pod to cancel, the finalizer should start a saver pod for the PVC")
	}
}

var _ = corev1.PodRunning
