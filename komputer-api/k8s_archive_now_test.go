package main

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func archiveNowAgent(phase komputerv1alpha1.KomputerAgentPhase, task komputerv1alpha1.AgentTaskStatus) *komputerv1alpha1.KomputerAgent {
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: phase, TaskStatus: task, PvcName: "a1-pvc"},
	}
}

// unsaved gives the agent a task newer than any save.
func unsaved(a *komputerv1alpha1.KomputerAgent) *komputerv1alpha1.KomputerAgent {
	started := metav1.NewTime(time.Now().Add(-time.Minute))
	a.Status.TaskStartedAt = &started
	return a
}

func TestArchiveAgentNow_RunningIdle_SetsRequest(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, archiveNowAgent(komputerv1alpha1.AgentPhaseRunning, komputerv1alpha1.AgentTaskComplete), s3Config(true))

	got, err := k8s.ArchiveAgentNow(context.Background(), "default", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotations[komputerv1alpha1.AnnotationArchiveRequested] != "true" {
		t.Errorf("annotations = %v, want the archive request", got.Annotations)
	}
	if got.Spec.Lifecycle != "" || got.Status.Phase != komputerv1alpha1.AgentPhaseRunning {
		t.Errorf("lifecycle %q / phase %q must be left to the operator", got.Spec.Lifecycle, got.Status.Phase)
	}
}

func TestArchiveAgentNow_AlreadyArchived_NoOp(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, archiveNowAgent(komputerv1alpha1.AgentPhaseArchived, komputerv1alpha1.AgentTaskComplete), s3Config(true))

	got, err := k8s.ArchiveAgentNow(context.Background(), "default", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Annotations[komputerv1alpha1.AnnotationArchiveRequested]; ok {
		t.Error("an archived agent needs no request")
	}
}

func TestArchiveAgentNow_TaskInProgress_Busy(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, archiveNowAgent(komputerv1alpha1.AgentPhaseRunning, komputerv1alpha1.AgentTaskInProgress), s3Config(true))

	if _, err := k8s.ArchiveAgentNow(context.Background(), "default", "a1"); err != ErrAgentBusy {
		t.Fatalf("err = %v, want ErrAgentBusy", err)
	}
}

func TestArchiveAgentNow_SquadMember_Refused(t *testing.T) {
	a := archiveNowAgent(komputerv1alpha1.AgentPhaseRunning, komputerv1alpha1.AgentTaskComplete)
	a.Status.Squad = true
	k8s := newFakeK8sWithStatusSubresource(t, a, s3Config(true))

	if _, err := k8s.ArchiveAgentNow(context.Background(), "default", "a1"); err != ErrAgentInSquad {
		t.Fatalf("err = %v, want ErrAgentInSquad", err)
	}
}

// A sleeping agent whose latest task isn't in S3 is woken idle, so the operator
// can copy its session out before archiving deletes the workspace.
func TestArchiveAgentNow_SleepingUnsaved_WakesIdleToSave(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, unsaved(archiveNowAgent(komputerv1alpha1.AgentPhaseSleeping, komputerv1alpha1.AgentTaskComplete)), s3Config(true))

	got, err := k8s.ArchiveAgentNow(context.Background(), "default", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Annotations[komputerv1alpha1.AnnotationArchiveRequested] != "true" || got.Annotations[komputerv1alpha1.AnnotationWakeIdle] != "true" {
		t.Errorf("annotations = %v, want archive request + wake-idle", got.Annotations)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhasePending {
		t.Errorf("phase = %q, want Pending (woken idle)", got.Status.Phase)
	}
}

func TestArchiveAgentNow_SleepingSaved_StaysAsleep(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, archiveNowAgent(komputerv1alpha1.AgentPhaseSleeping, komputerv1alpha1.AgentTaskComplete), s3Config(true))

	got, err := k8s.ArchiveAgentNow(context.Background(), "default", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("phase = %q, a saved sleeping agent needs no wake", got.Status.Phase)
	}
	if got.Annotations[komputerv1alpha1.AnnotationArchiveRequested] != "true" {
		t.Error("want the archive request")
	}
}

func TestArchiveAgentNow_SleepingS3Off_NoWake(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, unsaved(archiveNowAgent(komputerv1alpha1.AgentPhaseSleeping, komputerv1alpha1.AgentTaskComplete)), s3Config(false))

	got, err := k8s.ArchiveAgentNow(context.Background(), "default", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("phase = %q, without S3 there's nothing to save first", got.Status.Phase)
	}
}
