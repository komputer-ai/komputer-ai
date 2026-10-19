package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func archiveTestObjects() (*corev1.Pod, *corev1.PersistentVolumeClaim, *corev1.Service) {
	return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "my-agent-pod", Namespace: "default"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "my-agent-pvc", Namespace: "default"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: agentServiceName("my-agent"), Namespace: "default"}}
}

func archiveAgent(taskStatus komputerv1alpha1.AgentTaskStatus) *komputerv1alpha1.KomputerAgent {
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Lifecycle: komputerv1alpha1.AgentLifecycleArchive},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:        komputerv1alpha1.AgentPhaseRunning,
			PodName:      "my-agent-pod",
			PvcName:      "my-agent-pvc",
			TaskStatus:   taskStatus,
			TotalCostUSD: "1.2345",
			TotalTokens:  4200,
		},
	}
}

func assertGone(t *testing.T, c client.Client, obj client.Object, name string) {
	t.Helper()
	err := c.Get(context.Background(), types.NamespacedName{Name: name, Namespace: "default"}, obj)
	if !apierrors.IsNotFound(err) {
		t.Errorf("%T %s should have been deleted, got err=%v", obj, name, err)
	}
}

// A completed task on an Archive agent keeps the CR — with its cost and tokens —
// and removes the pod, the PVC and the Service.
func TestReconcileStatus_ArchiveLifecycle_ArchivesOnComplete(t *testing.T) {
	scheme := newTestScheme(t)
	pod, pvc, svc := archiveTestObjects()
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(pod, pvc, svc, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatalf("reconcileStatus returned error: %v", err)
	}

	var got komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("agent CR must be kept: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Errorf("Phase = %q, want Archived", got.Status.Phase)
	}
	if got.Status.PodName != "" || got.Status.PvcName != "" {
		t.Errorf("PodName/PvcName = %q/%q, want both empty", got.Status.PodName, got.Status.PvcName)
	}
	if got.Status.TotalCostUSD != "1.2345" || got.Status.TotalTokens != 4200 {
		t.Errorf("cost/tokens not preserved: %q / %d", got.Status.TotalCostUSD, got.Status.TotalTokens)
	}
	if got.Status.CompletionTime == nil {
		t.Error("CompletionTime should be stamped")
	}
	assertGone(t, c, &corev1.Pod{}, "my-agent-pod")
	assertGone(t, c, &corev1.PersistentVolumeClaim{}, "my-agent-pvc")
	assertGone(t, c, &corev1.Service{}, agentServiceName("my-agent"))
}

// Like AutoDelete, an errored task leaves the agent up for inspection.
func TestReconcileStatus_ArchiveLifecycle_ErrorDoesNotArchive(t *testing.T) {
	scheme := newTestScheme(t)
	pod, pvc, svc := archiveTestObjects()
	agent := archiveAgent(komputerv1alpha1.AgentTaskError)
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(pod, pvc, svc, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatalf("reconcileStatus returned error: %v", err)
	}
	var got komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.Phase == komputerv1alpha1.AgentPhaseArchived {
		t.Error("an errored task must not archive the agent")
	}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pvc", Namespace: "default"}, &corev1.PersistentVolumeClaim{}); err != nil {
		t.Errorf("PVC must be kept on error: %v", err)
	}
}

// A deleteTTL does not delay archiving (it is the record's retention window, not
// a delay like it is for AutoDelete).
func TestReconcileStatus_ArchiveLifecycle_DeleteTTLDoesNotDelay(t *testing.T) {
	scheme := newTestScheme(t)
	pod, pvc, svc := archiveTestObjects()
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(pod, pvc, svc, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	deleteTTL := &metav1.Duration{Duration: 720 * time.Hour}
	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, deleteTTL, false); err != nil {
		t.Fatalf("reconcileStatus returned error: %v", err)
	}
	var got komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Errorf("Phase = %q, want Archived", got.Status.Phase)
	}
}

// The full Reconcile on an archived agent finishes any cleanup a failed archive
// left behind, and never recreates the pod or PVC.
func TestReconcile_ArchivedAgent_CleansUpAndNeverRecreates(t *testing.T) {
	scheme := newTestScheme(t)
	pod, pvc, svc := archiveTestObjects()
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Status.Phase = komputerv1alpha1.AgentPhaseArchived
	agent.Status.PodName = ""
	agent.Status.PvcName = ""
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(pod, pvc, svc, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-agent", Namespace: "default"}}
	for i := 0; i < 2; i++ {
		if _, err := r.Reconcile(context.Background(), req); err != nil {
			t.Fatalf("Reconcile #%d returned error: %v", i+1, err)
		}
	}
	assertGone(t, c, &corev1.Pod{}, "my-agent-pod")
	assertGone(t, c, &corev1.PersistentVolumeClaim{}, "my-agent-pvc")
	assertGone(t, c, &corev1.Service{}, agentServiceName("my-agent"))

	var got komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), req.NamespacedName, &got); err != nil {
		t.Fatalf("archived agent must survive reconcile: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Errorf("Phase = %q, want Archived", got.Status.Phase)
	}
}

// deleteTTL still removes an archived record once it elapses.
func TestReconcile_ArchivedAgent_DeleteTTLRemovesRecord(t *testing.T) {
	scheme := newTestScheme(t)
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Status.Phase = komputerv1alpha1.AgentPhaseArchived
	agent.CreationTimestamp = metav1.NewTime(time.Now().Add(-2 * time.Hour))
	agent.Spec.DeleteTTL = &metav1.Duration{Duration: time.Hour}
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-agent", Namespace: "default"}}
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	assertGone(t, c, &komputerv1alpha1.KomputerAgent{}, "my-agent")
}

// A sleeping agent switched to lifecycle=Archive (e.g. woken with a new task and
// lifecycle) still carries its previous task's Complete status with no pod. It
// must not be archived — that would delete the workspace the wake is about to use.
func TestReconcileStatus_ArchiveLifecycle_NoPodDoesNotArchive(t *testing.T) {
	scheme := newTestScheme(t)
	_, pvc, _ := archiveTestObjects()
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Status.Phase = komputerv1alpha1.AgentPhaseSleeping
	agent.Status.PodName = ""
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(pvc, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileStatus(context.Background(), agent, nil, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatalf("reconcileStatus returned error: %v", err)
	}
	var got komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.Phase == komputerv1alpha1.AgentPhaseArchived {
		t.Error("a sleeping agent with no pod must not be archived")
	}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pvc", Namespace: "default"}, &corev1.PersistentVolumeClaim{}); err != nil {
		t.Errorf("PVC must be kept: %v", err)
	}
}

// Right after a Sleep-lifecycle agent is slept its pod is still terminating. A
// follow-up that switches it to Archive must not archive it off that pod.
func TestReconcileStatus_ArchiveLifecycle_TerminatingPodDoesNotArchive(t *testing.T) {
	scheme := newTestScheme(t)
	pod, pvc, _ := archiveTestObjects()
	now := metav1.Now()
	pod.DeletionTimestamp = &now
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Status.Phase = komputerv1alpha1.AgentPhaseSleeping
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(pvc, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatalf("reconcileStatus returned error: %v", err)
	}
	var got komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.Phase == komputerv1alpha1.AgentPhaseArchived {
		t.Error("an agent whose pod is terminating must not be archived")
	}
}

// A squad never adopts an archived agent: it is pruned from the member list
// instead of being marked squad-managed (which would rebuild its PVC and pod).
func TestMarkMembersAsSquad_PrunesArchivedMember(t *testing.T) {
	live := squadMember("live", time.Minute)
	archived := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	archived.Name = "gone"
	archived.Status.Phase = komputerv1alpha1.AgentPhaseArchived
	squad := testSquad("live", "gone")
	r := newSquadTTLReconciler(t, live, archived, squad)

	agents, err := r.markMembersAsSquad(context.Background(), squad)
	if err != nil {
		t.Fatalf("markMembersAsSquad: %v", err)
	}
	if len(agents) != 1 || agents[0].Name != "live" {
		t.Fatalf("agents = %v, want only the live member", agents)
	}
	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(context.Background(), types.NamespacedName{Name: "gone", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get archived agent: %v", err)
	}
	if got.Status.Squad {
		t.Error("archived agent must not be marked squad-managed")
	}
	var gotSquad komputerv1alpha1.KomputerSquad
	if err := r.Get(context.Background(), types.NamespacedName{Name: "test-squad", Namespace: "default"}, &gotSquad); err != nil {
		t.Fatalf("get squad: %v", err)
	}
	if len(gotSquad.Spec.Members) != 1 {
		t.Errorf("squad members = %d, want 1 (archived member pruned)", len(gotSquad.Spec.Members))
	}
}

// A schedule whose agent has been archived goes to Error instead of retrying
// the run forever.
func TestScheduleReconcile_ArchivedAgent_Errors(t *testing.T) {
	scheme := newTestScheme(t)
	past := metav1.NewTime(time.Now().Add(-time.Minute))
	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerScheduleSpec{
			Schedule:     "0 0 * * *",
			Instructions: "scheduled prompt",
			AgentName:    "my-agent",
		},
		Status: komputerv1alpha1.KomputerScheduleStatus{
			Phase:       komputerv1alpha1.SchedulePhaseActive,
			NextRunTime: &past,
		},
	}
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Status.Phase = komputerv1alpha1.AgentPhaseArchived
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(schedule, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerSchedule{}, &komputerv1alpha1.KomputerAgent{}).Build()
	r := &KomputerScheduleReconciler{Client: c, Scheme: scheme}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "sched1", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	var got komputerv1alpha1.KomputerSchedule
	if err := c.Get(context.Background(), types.NamespacedName{Name: "sched1", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get schedule: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.SchedulePhaseError {
		t.Errorf("Phase = %q, want Error", got.Status.Phase)
	}
}
