package controller

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// fakeExecer writes fixed bytes to stdout (or fails) and records the call.
type fakeExecer struct {
	out   []byte
	err   error
	calls []string
	delay time.Duration // how long the stream takes
	mu    sync.Mutex
}

func (f *fakeExecer) ExecStream(ctx context.Context, ns, pod, container string, cmd []string, stdout io.Writer) error {
	f.mu.Lock()
	f.calls = append(f.calls, ns+"/"+pod+"/"+container)
	f.mu.Unlock()
	time.Sleep(f.delay)
	if f.err != nil {
		return f.err
	}
	_, err := stdout.Write(f.out)
	return err
}

// fakeUploader records uploads.
type fakeUploader struct {
	key     string
	body    []byte
	n       int
	err     error
	copyErr error
	calls   []string // "copy src->dst" / "upload key", in order
}

func (f *fakeUploader) Copy(ctx context.Context, src, dst string) error {
	f.calls = append(f.calls, "copy "+src+"->"+dst)
	return f.copyErr
}

func (f *fakeUploader) Upload(ctx context.Context, key string, body io.Reader) error {
	f.n++
	f.key = key
	f.calls = append(f.calls, "upload "+key)
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	f.body = b
	return f.err
}

var copyOutS3 = &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "b", Prefix: "komputer"}

func copyOutAgent() *komputerv1alpha1.KomputerAgent {
	started := metav1.NewTime(time.Now().Add(-time.Minute))
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default", UID: types.UID("u-1")},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase: komputerv1alpha1.AgentPhaseRunning, PodName: "a1-pod",
			TaskStatus: komputerv1alpha1.AgentTaskComplete, TaskStartedAt: &started,
		},
	}
}

func copyOutPod(withS3Env bool) *corev1.Pod {
	c := corev1.Container{Name: "agent"}
	if withS3Env {
		c.Env = []corev1.EnvVar{{Name: "KOMPUTER_S3_KEY", Value: "k"}}
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "a1-pod", Namespace: "default"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{c}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

func copyOutReconciler(t *testing.T, ex PodExecer, up SessionUploader, objs ...client.Object) (*KomputerAgentReconciler, client.Client) {
	t.Helper()
	scheme := newTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).Build()
	return &KomputerAgentReconciler{
		Client: c, Scheme: scheme, Exec: ex,
		NewUploader: func(ctx context.Context, s3 *komputerv1alpha1.S3Spec) (SessionUploader, error) { return up, nil },
	}, c
}

func TestNeedsCopyOut(t *testing.T) {
	saved := metav1.NewTime(time.Now())
	for _, tc := range []struct {
		name  string
		agent func() *komputerv1alpha1.KomputerAgent
		pod   *corev1.Pod
		s3On  bool
		want  bool
	}{
		{"old pod, idle, unsaved", copyOutAgent, copyOutPod(false), true, true},
		{"pod saved itself", func() *komputerv1alpha1.KomputerAgent {
			a := copyOutAgent()
			a.Status.SessionStore = &komputerv1alpha1.SessionStoreStatus{OK: true, LastSavedAt: &saved}
			return a
		}, copyOutPod(true), true, false},
		{"task in progress", func() *komputerv1alpha1.KomputerAgent {
			a := copyOutAgent()
			a.Status.TaskStatus = komputerv1alpha1.AgentTaskInProgress
			return a
		}, copyOutPod(false), true, false},
		{"already saved", func() *komputerv1alpha1.KomputerAgent {
			a := copyOutAgent()
			a.Status.SessionCopiedOutAt = &saved
			return a
		}, copyOutPod(false), true, false},
		{"S3 off", copyOutAgent, copyOutPod(false), false, false},
		{"no pod", copyOutAgent, nil, true, false},
	} {
		if got := needsCopyOut(tc.agent(), tc.pod, tc.s3On); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCopyOutSession_UploadsExecOutputToAgentKey(t *testing.T) {
	ex := &fakeExecer{out: []byte("tarball-bytes")}
	up := &fakeUploader{}
	agent := copyOutAgent()
	r, c := copyOutReconciler(t, ex, up, agent)

	if err := r.copyOutSession(context.Background(), agent, copyOutS3, "a1-pod", "agent"); err != nil {
		t.Fatal(err)
	}
	if up.key != "komputer/default/a1/u-1/latest.tar.gz" || !bytes.Equal(up.body, []byte("tarball-bytes")) {
		t.Errorf("upload = %q / %q", up.key, up.body)
	}
	if len(ex.calls) != 1 || ex.calls[0] != "default/a1-pod/agent" {
		t.Errorf("exec calls = %v", ex.calls)
	}
	var got komputerv1alpha1.KomputerAgent
	_ = c.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got)
	if got.Status.SessionCopiedOutAt == nil {
		t.Error("sessionCopiedOutAt should be set")
	}
	cond := meta.FindStatusCondition(got.Status.Conditions, komputerv1alpha1.ConditionSessionSaved)
	if cond == nil || cond.Status != metav1.ConditionTrue || cond.Reason != "CopiedOut" {
		t.Errorf("condition = %+v", cond)
	}
	if !komputerv1alpha1.LatestTaskSaved(&got) {
		t.Error("LatestTaskSaved should be true after a copy-out")
	}
}

// Before overwriting latest, the operator keeps it as previous, so one bad save
// can't destroy the last good session.
func TestCopyOutSession_KeepsLatestAsPreviousBeforeUploading(t *testing.T) {
	up := &fakeUploader{}
	agent := copyOutAgent()
	r, _ := copyOutReconciler(t, &fakeExecer{out: []byte("tarball-bytes")}, up, agent)

	if err := r.copyOutSession(context.Background(), agent, copyOutS3, "a1-pod", "agent"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"copy komputer/default/a1/u-1/latest.tar.gz->komputer/default/a1/u-1/previous.tar.gz",
		"upload komputer/default/a1/u-1/latest.tar.gz",
	}
	if !reflect.DeepEqual(up.calls, want) {
		t.Errorf("calls = %v, want %v", up.calls, want)
	}
}

// Keeping the previous copy is best effort: a failed copy never stops the save.
func TestCopyOutSession_CopyFailure_StillSaves(t *testing.T) {
	up := &fakeUploader{copyErr: errors.New("access denied")}
	agent := copyOutAgent()
	r, _ := copyOutReconciler(t, &fakeExecer{out: []byte("tarball-bytes")}, up, agent)

	if err := r.copyOutSession(context.Background(), agent, copyOutS3, "a1-pod", "agent"); err != nil {
		t.Fatalf("a failed copy must not fail the save: %v", err)
	}
	if up.n != 1 {
		t.Errorf("uploads = %d, want 1", up.n)
	}
}

// Nothing to save means nothing to rotate either: previous is left alone.
func TestCopyOutSession_EmptySession_DoesNotRotate(t *testing.T) {
	up := &fakeUploader{}
	agent := copyOutAgent()
	r, _ := copyOutReconciler(t, &fakeExecer{out: nil}, up, agent)

	_ = r.copyOutSession(context.Background(), agent, copyOutS3, "a1-pod", "agent")
	if len(up.calls) != 0 {
		t.Errorf("calls = %v, want none", up.calls)
	}
}

func TestCopyOutSession_ExecFailure_SetsConditionFalse(t *testing.T) {
	agent := copyOutAgent()
	r, c := copyOutReconciler(t, &fakeExecer{err: errors.New("container not found")}, &fakeUploader{}, agent)

	if err := r.copyOutSession(context.Background(), agent, copyOutS3, "a1-pod", "agent"); err == nil {
		t.Fatal("want an error")
	}
	var got komputerv1alpha1.KomputerAgent
	_ = c.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got)
	cond := meta.FindStatusCondition(got.Status.Conditions, komputerv1alpha1.ConditionSessionSaved)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "SaveFailed" {
		t.Errorf("condition = %+v", cond)
	}
	if got.Status.SessionCopiedOutAt != nil {
		t.Error("sessionCopiedOutAt must stay unset on failure")
	}
}

func TestMaybeCopyOut_OldPodIdle_CopiesOutOnce(t *testing.T) {
	ex := &fakeExecer{out: []byte("x")}
	up := &fakeUploader{}
	agent := copyOutAgent()
	pod := copyOutPod(false)
	r, c := copyOutReconciler(t, ex, up, agent, pod)
	config := &komputerv1alpha1.KomputerConfig{Spec: komputerv1alpha1.KomputerConfigSpec{S3: copyOutS3}}

	for i := 0; i < 2; i++ {
		var cur komputerv1alpha1.KomputerAgent
		_ = c.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &cur)
		r.maybeCopyOut(context.Background(), &cur, pod, config)
	}
	if up.n != 1 {
		t.Errorf("uploads = %d, want exactly 1", up.n)
	}
}
