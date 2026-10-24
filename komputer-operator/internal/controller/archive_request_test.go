package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// Archive now (komputer.ai/archive-requested): archive the agent once no task
// is running and — with the S3 session store on — its latest task is saved,
// whatever its lifecycle, then clear the request.

func archiveRequestedAgent(taskStatus komputerv1alpha1.AgentTaskStatus) *komputerv1alpha1.KomputerAgent {
	a := archiveAgent(taskStatus)
	a.Spec.Lifecycle = "" // archive now doesn't depend on the lifecycle
	a.Annotations = map[string]string{komputerv1alpha1.AnnotationArchiveRequested: "true"}
	startedTask(a)
	return a
}

func s3OffKomputerConfig() *komputerv1alpha1.KomputerConfig {
	cfg := s3KomputerConfig()
	cfg.Spec.S3 = nil
	return cfg
}

func TestArchiveRequested_IdleAndSaved_Archives(t *testing.T) {
	pod, pvc, svc := archiveTestObjects()
	agent := archiveRequestedAgent(komputerv1alpha1.AgentTaskComplete)
	markSaved(agent)
	r, c := saveFirstReconciler(t, nil, nil, pod, pvc, svc, agent, s3KomputerConfig())

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatal(err)
	}
	got := getAgentT(t, c, "my-agent")
	if got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Fatalf("phase = %q, want Archived", got.Status.Phase)
	}
	if _, ok := got.Annotations[komputerv1alpha1.AnnotationArchiveRequested]; ok {
		t.Error("the request must be cleared once archived, or a later wake would archive again")
	}
	if got.Spec.Lifecycle != "" {
		t.Errorf("lifecycle = %q, archive now must not change it", got.Spec.Lifecycle)
	}
	assertGone(t, c, &corev1.Pod{}, "my-agent-pod")
	assertGone(t, c, &corev1.PersistentVolumeClaim{}, "my-agent-pvc")
}

func TestArchiveRequested_TaskInProgress_Waits(t *testing.T) {
	pod, pvc, svc := archiveTestObjects()
	agent := archiveRequestedAgent(komputerv1alpha1.AgentTaskInProgress)
	markSaved(agent)
	r, c := saveFirstReconciler(t, nil, nil, pod, pvc, svc, agent, s3KomputerConfig())

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatal(err)
	}
	got := getAgentT(t, c, "my-agent")
	if got.Status.Phase == komputerv1alpha1.AgentPhaseArchived {
		t.Fatal("a running task must finish first")
	}
	if got.Annotations[komputerv1alpha1.AnnotationArchiveRequested] != "true" {
		t.Error("the request must stay until it can be honored")
	}
}

func TestArchiveRequested_S3OnUnsaved_WaitsForTheSave(t *testing.T) {
	pod, pvc, svc := archiveTestObjects()
	agent := archiveRequestedAgent(komputerv1alpha1.AgentTaskComplete)
	r, c := saveFirstReconciler(t, nil, nil, pod, pvc, svc, agent, s3KomputerConfig())

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if got := getAgentT(t, c, "my-agent"); got.Status.Phase == komputerv1alpha1.AgentPhaseArchived {
		t.Fatal("must not archive (and delete the workspace) before the session is saved")
	}
}

func TestArchiveRequested_SleepingAndSaved_Archives(t *testing.T) {
	_, pvc, svc := archiveTestObjects()
	agent := archiveRequestedAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Status.Phase = komputerv1alpha1.AgentPhaseSleeping
	agent.Status.PodName = ""
	markSaved(agent)
	r, c := saveFirstReconciler(t, nil, nil, pvc, svc, agent, s3KomputerConfig())

	if err := r.reconcileStatus(context.Background(), agent, nil, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if got := getAgentT(t, c, "my-agent"); got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Fatalf("phase = %q, want Archived", got.Status.Phase)
	}
	assertGone(t, c, &corev1.PersistentVolumeClaim{}, "my-agent-pvc")
}

func TestArchiveRequested_S3Off_ArchivesWithoutASave(t *testing.T) {
	pod, pvc, svc := archiveTestObjects()
	agent := archiveRequestedAgent(komputerv1alpha1.AgentTaskComplete)
	r, c := saveFirstReconciler(t, nil, nil, pod, pvc, svc, agent, s3OffKomputerConfig())

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if got := getAgentT(t, c, "my-agent"); got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Fatalf("phase = %q, want Archived", got.Status.Phase)
	}
}

// Without a request, nothing changes for an idle agent with no Archive lifecycle.
func TestArchiveRequested_NoRequest_NoArchive(t *testing.T) {
	pod, pvc, svc := archiveTestObjects()
	agent := archiveRequestedAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Annotations = nil
	markSaved(agent)
	r, c := saveFirstReconciler(t, nil, nil, pod, pvc, svc, agent, s3KomputerConfig())

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if got := getAgentT(t, c, "my-agent"); got.Status.Phase == komputerv1alpha1.AgentPhaseArchived {
		t.Fatal("archived without a request")
	}
}
