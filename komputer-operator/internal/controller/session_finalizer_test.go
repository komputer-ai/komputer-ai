package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// Deletes wait for the session: the komputer.ai/session-store finalizer holds an
// agent's deletion until its latest task's session is saved to S3.

func deletingAgent(mutate ...func(*komputerv1alpha1.KomputerAgent)) *komputerv1alpha1.KomputerAgent {
	now := metav1.Now()
	started := metav1.NewTime(time.Now().Add(-time.Minute))
	a := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "a1", Namespace: "default", UID: "u-1",
			DeletionTimestamp: &now,
			Finalizers:        []string{komputerv1alpha1.FinalizerSessionStore},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase: komputerv1alpha1.AgentPhaseRunning, PodName: "a1-pod", PvcName: "a1-pvc",
			TaskStatus: komputerv1alpha1.AgentTaskComplete, TaskStartedAt: &started,
		},
	}
	for _, m := range mutate {
		m(a)
	}
	return a
}

func runningAgentPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "a1-pod", Namespace: "default"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "agent", Image: "img"}}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

func agentPVC(terminating bool) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "a1-pvc", Namespace: "default"}}
	if terminating {
		now := metav1.Now()
		pvc.DeletionTimestamp = &now
		pvc.Finalizers = []string{"kubernetes.io/pvc-protection"}
	}
	return pvc
}

func agentTemplate() *komputerv1alpha1.KomputerAgentTemplate {
	return &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentTemplateSpec{PodSpec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "agent", Image: "agent-img:1"}},
		}},
	}
}

func finalizerGone(t *testing.T, c client.Client) bool {
	t.Helper()
	var a komputerv1alpha1.KomputerAgent
	err := c.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &a)
	if err != nil {
		return true // the fake client deletes the object once its last finalizer is gone
	}
	for _, f := range a.Finalizers {
		if f == komputerv1alpha1.FinalizerSessionStore {
			return false
		}
	}
	return true
}

func agentMessage(t *testing.T, c client.Client) string {
	t.Helper()
	var a komputerv1alpha1.KomputerAgent
	_ = c.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &a)
	return a.Status.Message
}

func saverPods(t *testing.T, c client.Client) []corev1.Pod {
	t.Helper()
	var pods corev1.PodList
	if err := c.List(context.Background(), &pods, client.MatchingLabels{komputerv1alpha1.LabelSessionSaver: "true"}); err != nil {
		t.Fatal(err)
	}
	return pods.Items
}

func TestSessionFinalizer_AddedOnlyWhenS3On(t *testing.T) {
	for _, on := range []bool{true, false} {
		agent := &komputerv1alpha1.KomputerAgent{ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"}}
		objs := []client.Object{agent}
		if on {
			objs = append(objs, s3KomputerConfig())
		}
		r, c := saveFirstReconciler(t, nil, nil, objs...)
		if _, err := r.ensureSessionFinalizer(context.Background(), agent); err != nil {
			t.Fatal(err)
		}
		var got komputerv1alpha1.KomputerAgent
		_ = c.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got)
		has := len(got.Finalizers) == 1 && got.Finalizers[0] == komputerv1alpha1.FinalizerSessionStore
		if has != on {
			t.Errorf("S3 on=%v: finalizer present=%v", on, has)
		}
	}
}

func TestSessionFinalizer_S3Off_Removed(t *testing.T) {
	agent := deletingAgent()
	r, c := saveFirstReconciler(t, nil, nil, agent, runningAgentPod())
	if _, err := r.handleSessionFinalizer(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	if !finalizerGone(t, c) {
		t.Error("finalizer must be removed when S3 is off")
	}
}

func TestSessionFinalizer_Saved_RemovedImmediately(t *testing.T) {
	agent := deletingAgent(markSaved)
	up := &fakeUploader{}
	r, c := saveFirstReconciler(t, &fakeExecer{}, up, agent, runningAgentPod(), s3KomputerConfig())
	if _, err := r.handleSessionFinalizer(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	if !finalizerGone(t, c) || up.n != 0 {
		t.Errorf("finalizer gone=%v uploads=%d; want removed without another upload", finalizerGone(t, c), up.n)
	}
}

func TestSessionFinalizer_RunningPod_CopiesOutThenRemoves(t *testing.T) {
	agent := deletingAgent()
	up := &fakeUploader{}
	ex := &fakeExecer{out: []byte("tar")}
	r, c := saveFirstReconciler(t, ex, up, agent, runningAgentPod(), s3KomputerConfig())
	if _, err := r.handleSessionFinalizer(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	if up.n != 1 || ex.calls[0] != "default/a1-pod/agent" {
		t.Errorf("uploads=%d exec=%v", up.n, ex.calls)
	}
	if !finalizerGone(t, c) {
		t.Error("finalizer must be removed after a successful save")
	}
}

func TestSessionFinalizer_TaskInProgress_CancelsAndWaits(t *testing.T) {
	var cancels atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cancels.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("KOMPUTER_API_URL", srv.URL)

	agent := deletingAgent(func(a *komputerv1alpha1.KomputerAgent) { a.Status.TaskStatus = komputerv1alpha1.AgentTaskInProgress })
	r, c := saveFirstReconciler(t, &fakeExecer{}, &fakeUploader{}, agent, runningAgentPod(), s3KomputerConfig())
	res, err := r.handleSessionFinalizer(context.Background(), agent)
	if err != nil {
		t.Fatal(err)
	}
	if cancels.Load() != 1 || res.RequeueAfter == 0 {
		t.Errorf("cancels=%d requeue=%v; want one cancel and a requeue", cancels.Load(), res.RequeueAfter)
	}
	if finalizerGone(t, c) {
		t.Error("finalizer must stay while the task is still running")
	}
}

func TestSessionFinalizer_PVCOnly_CreatesSaverPod(t *testing.T) {
	agent := deletingAgent(func(a *komputerv1alpha1.KomputerAgent) {
		a.Status.Phase = komputerv1alpha1.AgentPhaseSleeping
		a.Status.PodName = ""
	})
	r, c := saveFirstReconciler(t, &fakeExecer{}, &fakeUploader{}, agent, agentPVC(false), agentTemplate(), s3KomputerConfig())
	r.MaxConcurrentSavers = 4
	if _, err := r.handleSessionFinalizer(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	pods := saverPods(t, c)
	if len(pods) != 1 {
		t.Fatalf("saver pods = %d, want 1", len(pods))
	}
	p := pods[0]
	if p.Spec.ActiveDeadlineSeconds == nil || *p.Spec.ActiveDeadlineSeconds != 300 {
		t.Errorf("activeDeadlineSeconds = %v, want 300", p.Spec.ActiveDeadlineSeconds)
	}
	if p.Spec.Containers[0].Image != "agent-img:1" {
		t.Errorf("image = %q, want the template's agent image", p.Spec.Containers[0].Image)
	}
	vol := p.Spec.Volumes[0]
	if vol.PersistentVolumeClaim == nil || vol.PersistentVolumeClaim.ClaimName != "a1-pvc" || !vol.PersistentVolumeClaim.ReadOnly {
		t.Errorf("volume = %+v, want the agent PVC read-only", vol)
	}
	if finalizerGone(t, c) {
		t.Error("finalizer must stay until the saver pod has copied the session out")
	}
}

func TestSessionFinalizer_SaverPodRunning_CopiesOutAndCleansUp(t *testing.T) {
	agent := deletingAgent(func(a *komputerv1alpha1.KomputerAgent) {
		a.Status.Phase = komputerv1alpha1.AgentPhaseSleeping
		a.Status.PodName = ""
	})
	saver := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "a1-session-saver", Namespace: "default", Labels: map[string]string{komputerv1alpha1.LabelSessionSaver: "true"}},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "saver"}}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	up := &fakeUploader{}
	ex := &fakeExecer{out: []byte("tar")}
	r, c := saveFirstReconciler(t, ex, up, agent, agentPVC(false), saver, agentTemplate(), s3KomputerConfig())
	r.MaxConcurrentSavers = 4
	if _, err := r.handleSessionFinalizer(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	if up.n != 1 || ex.calls[0] != "default/a1-session-saver/saver" {
		t.Errorf("uploads=%d exec=%v", up.n, ex.calls)
	}
	if len(saverPods(t, c)) != 0 {
		t.Error("the saver pod must be deleted after the copy-out")
	}
	if !finalizerGone(t, c) {
		t.Error("finalizer must be removed after the save")
	}
}

func TestSessionFinalizer_SaverCapReached_Waits(t *testing.T) {
	agent := deletingAgent(func(a *komputerv1alpha1.KomputerAgent) {
		a.Status.Phase = komputerv1alpha1.AgentPhaseSleeping
		a.Status.PodName = ""
	})
	objs := []client.Object{agent, agentPVC(false), agentTemplate(), s3KomputerConfig()}
	for i := 0; i < 4; i++ {
		objs = append(objs, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("other-%d-session-saver", i), Namespace: "other",
			Labels: map[string]string{komputerv1alpha1.LabelSessionSaver: "true"},
		}})
	}
	r, c := saveFirstReconciler(t, &fakeExecer{}, &fakeUploader{}, objs...)
	r.MaxConcurrentSavers = 4
	res, err := r.handleSessionFinalizer(context.Background(), agent)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(saverPods(t, c)); n != 4 {
		t.Errorf("saver pods = %d, want still 4 (no 5th)", n)
	}
	if got := agentMessage(t, c); got != "Deleting — waiting for a session saver slot" {
		t.Errorf("message = %q", got)
	}
	if res.RequeueAfter == 0 {
		t.Error("should requeue to retry for a slot")
	}
}

func TestSessionFinalizer_CapFromEnv(t *testing.T) {
	t.Setenv("KOMPUTER_MAX_CONCURRENT_SESSION_SAVERS", "1")
	if got := MaxConcurrentSaversFromEnv(); got != 1 {
		t.Errorf("got %d, want 1", got)
	}
	t.Setenv("KOMPUTER_MAX_CONCURRENT_SESSION_SAVERS", "nope")
	if got := MaxConcurrentSaversFromEnv(); got != 4 {
		t.Errorf("invalid value: got %d, want the default 4", got)
	}
}

func TestSessionFinalizer_TerminatingPVC_NoSaverPod(t *testing.T) {
	agent := deletingAgent(func(a *komputerv1alpha1.KomputerAgent) {
		a.Status.Phase = komputerv1alpha1.AgentPhaseSleeping
		a.Status.PodName = ""
	})
	r, c := saveFirstReconciler(t, &fakeExecer{}, &fakeUploader{}, agent, agentPVC(true), agentTemplate(), s3KomputerConfig())
	r.MaxConcurrentSavers = 4
	if _, err := r.handleSessionFinalizer(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	if len(saverPods(t, c)) != 0 {
		t.Error("no saver pod for a terminating PVC")
	}
	if got := agentMessage(t, c); got != "Deleting — waiting to save the session to S3: workspace volume is already being deleted" {
		t.Errorf("message = %q", got)
	}
	if finalizerGone(t, c) {
		t.Error("finalizer must stay (the user can force the delete)")
	}
}

func TestSessionFinalizer_ArchivedAgent_RemovedImmediately(t *testing.T) {
	agent := deletingAgent(func(a *komputerv1alpha1.KomputerAgent) {
		a.Status.Phase = komputerv1alpha1.AgentPhaseArchived
		a.Status.PodName, a.Status.PvcName = "", ""
	})
	r, c := saveFirstReconciler(t, &fakeExecer{}, &fakeUploader{}, agent, s3KomputerConfig())
	if _, err := r.handleSessionFinalizer(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	if !finalizerGone(t, c) {
		t.Error("an archived agent has no local copy left: nothing to wait for")
	}
}

func TestSessionFinalizer_UploadFails_KeepsFinalizerWithMessage(t *testing.T) {
	agent := deletingAgent()
	r, c := saveFirstReconciler(t, &fakeExecer{out: []byte("tar")}, &fakeUploader{err: errors.New("AccessDenied")}, agent, runningAgentPod(), s3KomputerConfig())
	res, err := r.handleSessionFinalizer(context.Background(), agent)
	if err != nil {
		t.Fatal(err)
	}
	if finalizerGone(t, c) {
		t.Fatal("finalizer must stay when the save fails")
	}
	if got := agentMessage(t, c); got != "Deleting — waiting to save the session to S3: AccessDenied" {
		t.Errorf("message = %q", got)
	}
	if res.RequeueAfter == 0 {
		t.Error("should requeue to retry the save")
	}
}

// The finalizer is handled before the squad early-return in Reconcile, so squad
// members' deletes are released too.
func TestReconcile_DeletingSquadMember_SessionFinalizerHandled(t *testing.T) {
	agent := deletingAgent(markSaved, func(a *komputerv1alpha1.KomputerAgent) {
		a.Labels = map[string]string{"komputer.ai/squad": "sq"}
		a.Status.Squad = true
	})
	r, c := saveFirstReconciler(t, nil, nil, agent, s3KomputerConfig())
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "a1", Namespace: "default"}}); err != nil {
		t.Fatal(err)
	}
	if !finalizerGone(t, c) {
		t.Error("a saved squad member's delete must be released")
	}
}

// A live agent gets the finalizer from Reconcile when S3 is on.
func TestReconcile_AddsSessionFinalizerWhenS3On(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default", Labels: map[string]string{"komputer.ai/squad": "sq"}},
	}
	r, c := saveFirstReconciler(t, nil, nil, agent, s3KomputerConfig())
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "a1", Namespace: "default"}}); err != nil {
		t.Fatal(err)
	}
	if finalizerGone(t, c) {
		t.Error("Reconcile should add the session-store finalizer when S3 is on")
	}
}
