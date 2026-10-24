package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// Save first: with the S3 session store on, nothing that destroys an agent's
// last local copy of its session runs until the latest task is saved.

func s3KomputerConfig() *komputerv1alpha1.KomputerConfig {
	return &komputerv1alpha1.KomputerConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec: komputerv1alpha1.KomputerConfigSpec{
			Redis: komputerv1alpha1.RedisSpec{Address: "redis:6379"},
			S3:    &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "b", Prefix: "komputer"},
		},
	}
}

func startedTask(a *komputerv1alpha1.KomputerAgent) {
	started := metav1.NewTime(time.Now().Add(-time.Minute))
	a.Status.TaskStartedAt = &started
	a.UID = "u-1"
}

func markSaved(a *komputerv1alpha1.KomputerAgent) {
	saved := metav1.NewTime(time.Now())
	a.Status.SessionCopiedOutAt = &saved
}

func saveFirstReconciler(t *testing.T, ex PodExecer, up SessionUploader, objs ...client.Object) (*KomputerAgentReconciler, client.Client) {
	t.Helper()
	scheme := newTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}, &komputerv1alpha1.KomputerSchedule{}).Build()
	return &KomputerAgentReconciler{
		Client: c, Scheme: scheme, Exec: ex,
		NewUploader: func(ctx context.Context, s3 *komputerv1alpha1.S3Spec) (SessionUploader, error) { return up, nil },
	}, c
}

func getAgentT(t *testing.T, c client.Client, name string) *komputerv1alpha1.KomputerAgent {
	t.Helper()
	var a komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: name, Namespace: "default"}, &a); err != nil {
		t.Fatal(err)
	}
	return &a
}

func TestArchive_S3OnUnsaved_DoesNotArchive(t *testing.T) {
	pod, pvc, svc := archiveTestObjects()
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	startedTask(agent)
	r, c := saveFirstReconciler(t, nil, nil, pod, pvc, svc, agent, s3KomputerConfig())

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if got := getAgentT(t, c, "my-agent").Status.Phase; got == komputerv1alpha1.AgentPhaseArchived {
		t.Fatal("an unsaved agent must not be archived")
	}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pvc", Namespace: "default"}, &corev1.PersistentVolumeClaim{}); err != nil {
		t.Errorf("PVC must be kept: %v", err)
	}
}

func TestArchive_S3OnSaved_Archives(t *testing.T) {
	pod, pvc, svc := archiveTestObjects()
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	startedTask(agent)
	markSaved(agent)
	r, c := saveFirstReconciler(t, nil, nil, pod, pvc, svc, agent, s3KomputerConfig())

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if got := getAgentT(t, c, "my-agent").Status.Phase; got != komputerv1alpha1.AgentPhaseArchived {
		t.Fatalf("phase = %q, want Archived", got)
	}
}

func TestArchive_S3OnOldPod_CopiesOutThenArchives(t *testing.T) {
	pod, pvc, svc := archiveTestObjects()
	pod.Spec.Containers = []corev1.Container{{Name: "agent"}}
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	startedTask(agent)
	up := &fakeUploader{}
	r, c := saveFirstReconciler(t, &fakeExecer{out: []byte("tar")}, up, pod, pvc, svc, agent, s3KomputerConfig())
	cfg := s3KomputerConfig()

	r.maybeCopyOut(context.Background(), agent, pod, cfg)
	if up.n != 1 {
		t.Fatalf("uploads = %d, want 1", up.n)
	}
	saved := getAgentT(t, c, "my-agent")
	if err := r.reconcileStatus(context.Background(), saved, pod, "my-agent-pvc", "my-agent-pod", nil, nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if got := getAgentT(t, c, "my-agent").Status.Phase; got != komputerv1alpha1.AgentPhaseArchived {
		t.Fatalf("phase = %q, want Archived after the copy-out", got)
	}
}

func TestArchive_S3OnCopyOutFails_DoesNotArchive(t *testing.T) {
	pod, pvc, svc := archiveTestObjects()
	pod.Spec.Containers = []corev1.Container{{Name: "agent"}}
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	startedTask(agent)
	r, c := saveFirstReconciler(t, &fakeExecer{err: errors.New("boom")}, &fakeUploader{}, pod, pvc, svc, agent, s3KomputerConfig())

	r.maybeCopyOut(context.Background(), agent, pod, s3KomputerConfig())
	cur := getAgentT(t, c, "my-agent")
	if err := r.reconcileStatus(context.Background(), cur, pod, "my-agent-pvc", "my-agent-pod", nil, nil, nil, false); err != nil {
		t.Fatal(err)
	}
	got := getAgentT(t, c, "my-agent")
	if got.Status.Phase == komputerv1alpha1.AgentPhaseArchived {
		t.Fatal("a failed save must stop the archive")
	}
	cond := meta.FindStatusCondition(got.Status.Conditions, komputerv1alpha1.ConditionSessionSaved)
	if cond == nil || cond.Message != "Archive stopped: couldn't save the session to S3: boom" {
		t.Errorf("condition = %+v", cond)
	}
}

func ephemeralSleepAgent() *komputerv1alpha1.KomputerAgent {
	a := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	a.Spec.Lifecycle = komputerv1alpha1.AgentLifecycleSleep
	startedTask(a)
	return a
}

func TestEphemeralSleep_S3OnSaved_Sleeps(t *testing.T) {
	pod, _, _ := archiveTestObjects()
	agent := ephemeralSleepAgent()
	markSaved(agent)
	r, c := saveFirstReconciler(t, nil, nil, pod, agent, s3KomputerConfig())

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, nil, true); err != nil {
		t.Fatal(err)
	}
	if got := getAgentT(t, c, "my-agent").Status.Phase; got != komputerv1alpha1.AgentPhaseSleeping {
		t.Fatalf("phase = %q, want Sleeping", got)
	}
	assertGone(t, c, &corev1.Pod{}, "my-agent-pod")
}

func TestEphemeralSleep_S3OnUnsaved_StaysAwake(t *testing.T) {
	pod, _, _ := archiveTestObjects()
	agent := ephemeralSleepAgent()
	r, c := saveFirstReconciler(t, nil, nil, pod, agent, s3KomputerConfig())

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, nil, true); err != nil {
		t.Fatal(err)
	}
	if got := getAgentT(t, c, "my-agent").Status.Phase; got == komputerv1alpha1.AgentPhaseSleeping {
		t.Fatal("an unsaved ephemeral agent must stay awake")
	}
}

func TestSleepUnsafe(t *testing.T) {
	saved := ephemeralSleepAgent()
	markSaved(saved)
	unsaved := ephemeralSleepAgent()
	for _, tc := range []struct {
		name      string
		ephemeral bool
		s3On      bool
		agent     *komputerv1alpha1.KomputerAgent
		want      bool
	}{
		{"PVC workspace", false, false, unsaved, false},
		{"ephemeral, S3 off", true, false, saved, true},
		{"ephemeral, S3 on, saved", true, true, saved, false},
		{"ephemeral, S3 on, unsaved", true, true, unsaved, true},
	} {
		if got := sleepUnsafe(tc.ephemeral, tc.s3On, tc.agent); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestScheduleReconcile_S3OnArchivedAgent_Wakes(t *testing.T) {
	scheme := newTestScheme(t)
	past := metav1.NewTime(time.Now().Add(-time.Minute))
	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerScheduleSpec{Schedule: "0 0 * * *", Instructions: "scheduled prompt", AgentName: "my-agent"},
		Status:     komputerv1alpha1.KomputerScheduleStatus{Phase: komputerv1alpha1.SchedulePhaseActive, NextRunTime: &past},
	}
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Status.Phase = komputerv1alpha1.AgentPhaseArchived
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(schedule, agent, s3KomputerConfig()).
		WithStatusSubresource(&komputerv1alpha1.KomputerSchedule{}, &komputerv1alpha1.KomputerAgent{}).Build()
	r := &KomputerScheduleReconciler{Client: c, Scheme: scheme}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "sched1", Namespace: "default"}}); err != nil {
		t.Fatal(err)
	}
	got := getAgentT(t, c, "my-agent")
	if got.Status.Phase != komputerv1alpha1.AgentPhasePending || got.Spec.Instructions != "scheduled prompt" {
		t.Errorf("phase = %q, instructions = %q; want Pending and the scheduled prompt", got.Status.Phase, got.Spec.Instructions)
	}
}

func TestMarkMembersAsSquad_S3OnArchivedMember_AdoptedAsSleeping(t *testing.T) {
	live := squadMember("live", time.Minute)
	archived := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	archived.Name = "gone"
	archived.Status.Phase = komputerv1alpha1.AgentPhaseArchived
	squad := testSquad("live", "gone")
	r := newSquadTTLReconciler(t, live, archived, squad, s3KomputerConfig())

	agents, err := r.markMembersAsSquad(context.Background(), squad)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 2 {
		t.Fatalf("agents = %d, want both members kept", len(agents))
	}
	var got komputerv1alpha1.KomputerAgent
	_ = r.Get(context.Background(), types.NamespacedName{Name: "gone", Namespace: "default"}, &got)
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("phase = %q, want Sleeping", got.Status.Phase)
	}
}

func TestReconcile_ArchivedWokenToPending_BuildsFreshPVCAndPod(t *testing.T) {
	template := &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentTemplateSpec{PodSpec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "agent", Image: "img"}},
		}},
	}
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Status = komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhasePending}
	r, c := saveFirstReconciler(t, nil, nil, template, agent, s3KomputerConfig())

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-agent", Namespace: "default"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pvc", Namespace: "default"}, &corev1.PersistentVolumeClaim{}); err != nil {
		t.Fatalf("a fresh PVC should be created: %v", err)
	}
	var pod corev1.Pod
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pod", Namespace: "default"}, &pod); err != nil {
		t.Fatalf("a pod should be created: %v", err)
	}
	if !podHasS3Env(pod.Spec.Containers[0]) {
		t.Error("the new pod must carry the S3 env so it restores the session")
	}
}

// An S3-backed ephemeral agent can sleep; its status must not then name a PVC,
// or the next reconcile takes that for an existing workspace and creates one.
func ephemeralReconcileObjects(lifecycle komputerv1alpha1.AgentLifecycle, sleepTTL *metav1.Duration) (*komputerv1alpha1.KomputerAgentTemplate, *corev1.Pod, *komputerv1alpha1.KomputerAgent) {
	template := &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentTemplateSpec{PodSpec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "agent", Image: "img"}},
		}},
	}
	pod, _, _ := archiveTestObjects()
	ephemeral := true
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Spec.Lifecycle = lifecycle
	agent.Spec.SleepTTL = sleepTTL
	agent.Spec.Storage = &komputerv1alpha1.StorageSpec{Ephemeral: &ephemeral}
	agent.Status.PvcName = ""
	startedTask(agent)
	markSaved(agent)
	idle := metav1.NewTime(time.Now().Add(-time.Hour))
	agent.Status.LastActivityAt = &idle
	return template, pod, agent
}

func assertSleptWithoutPVC(t *testing.T, r *KomputerAgentReconciler, c client.Client) {
	t.Helper()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-agent", Namespace: "default"}}
	for i := 0; i < 3; i++ {
		if _, err := r.Reconcile(context.Background(), req); err != nil {
			t.Fatal(err)
		}
	}
	got := getAgentT(t, c, "my-agent")
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Fatalf("phase = %q, want Sleeping", got.Status.Phase)
	}
	if got.Status.PvcName != "" {
		t.Errorf("status.pvcName = %q, want empty for an emptyDir workspace", got.Status.PvcName)
	}
	assertGone(t, c, &corev1.PersistentVolumeClaim{}, "my-agent-pvc")
}

func TestReconcile_EphemeralSleepLifecycle_RecordsNoPVC(t *testing.T) {
	template, pod, agent := ephemeralReconcileObjects(komputerv1alpha1.AgentLifecycleSleep, nil)
	r, c := saveFirstReconciler(t, nil, nil, template, pod, agent, s3KomputerConfig())
	assertSleptWithoutPVC(t, r, c)
}

func TestReconcile_EphemeralSleepTTL_RecordsNoPVC(t *testing.T) {
	template, pod, agent := ephemeralReconcileObjects("", &metav1.Duration{Duration: time.Minute})
	r, c := saveFirstReconciler(t, nil, nil, template, pod, agent, s3KomputerConfig())
	assertSleptWithoutPVC(t, r, c)
}
