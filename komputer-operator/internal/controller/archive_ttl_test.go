package controller

import (
	"context"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func TestEvaluateArchiveTTL(t *testing.T) {
	tests := []struct {
		name        string
		agent       *komputerv1alpha1.KomputerAgent
		archiveTTL  *metav1.Duration
		now         time.Time
		wantArchive bool
		wantExpiry  bool
		// wantRequeue is checked only when non-zero.
		wantRequeue time.Duration
	}{
		{
			name:  "unset never archives",
			agent: ttlAgent(withLastActivity(0)),
			now:   baseTime.Add(999 * time.Hour),
		},
		{
			name:       "never-ran agent has no idle clock",
			agent:      ttlAgent(),
			archiveTTL: dur(time.Minute),
			now:        baseTime.Add(999 * time.Hour),
		},
		{
			name:        "not yet elapsed requeues for the remainder",
			agent:       ttlAgent(withLastActivity(0)),
			archiveTTL:  dur(2 * time.Hour),
			now:         baseTime.Add(30 * time.Minute),
			wantExpiry:  true,
			wantRequeue: 90 * time.Minute,
		},
		{
			name:        "elapsed archives a running agent",
			agent:       ttlAgent(withLastActivity(0)),
			archiveTTL:  dur(2 * time.Hour),
			now:         baseTime.Add(2 * time.Hour),
			wantArchive: true,
			wantExpiry:  true,
		},
		{
			name:        "elapsed archives a sleeping agent too",
			agent:       ttlAgent(withLastActivity(0), withPhase(komputerv1alpha1.AgentPhaseSleeping)),
			archiveTTL:  dur(2 * time.Hour),
			now:         baseTime.Add(3 * time.Hour),
			wantArchive: true,
			wantExpiry:  true,
		},
		{
			name:       "never fires mid-task",
			agent:      ttlAgent(withLastActivity(0), withTaskStatus(komputerv1alpha1.AgentTaskInProgress)),
			archiveTTL: dur(time.Minute),
			now:        baseTime.Add(999 * time.Hour),
		},
		{
			name:       "never fires while compacting",
			agent:      ttlAgent(withLastActivity(0), withTaskStatus(komputerv1alpha1.AgentTaskCompacting)),
			archiveTTL: dur(time.Minute),
			now:        baseTime.Add(999 * time.Hour),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := evaluateArchiveTTL(tc.agent, tc.archiveTTL, tc.now)
			if d.Archive != tc.wantArchive {
				t.Errorf("Archive = %v, want %v", d.Archive, tc.wantArchive)
			}
			if (d.ExpiresAt != nil) != tc.wantExpiry {
				t.Errorf("ExpiresAt = %v, want set=%v", d.ExpiresAt, tc.wantExpiry)
			}
			if tc.wantRequeue != 0 && d.RequeueAfter != tc.wantRequeue {
				t.Errorf("RequeueAfter = %v, want %v", d.RequeueAfter, tc.wantRequeue)
			}
			if tc.wantArchive && d.RequeueAfter != 0 {
				t.Errorf("RequeueAfter = %v, want 0 once elapsed", d.RequeueAfter)
			}
		})
	}
}

// archiveTTLAgent is an idle agent with a pod and PVC, past a 30m archiveTTL.
func archiveTTLAgent(mutate ...func(*komputerv1alpha1.KomputerAgent)) (*komputerv1alpha1.KomputerAgent, *corev1.Pod, *corev1.PersistentVolumeClaim, *corev1.Service) {
	pod, pvc, svc := archiveTestObjects()
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Spec.Lifecycle = ""
	last := metav1.NewTime(time.Now().Add(-time.Hour))
	agent.Status.LastActivityAt = &last
	agent.CreationTimestamp = metav1.NewTime(time.Now().Add(-2 * time.Hour))
	for _, m := range mutate {
		m(agent)
	}
	return agent, pod, pvc, svc
}

func TestApplyTTL_ArchiveTTLArchivesIdleAgent(t *testing.T) {
	ctx := context.Background()
	agent, pod, pvc, svc := archiveTTLAgent()
	stale := metav1.NewTime(time.Now())
	agent.Status.ArchiveExpiresAt = &stale
	r := newTTLReconciler(t, agent, pod, pvc, svc)

	_, handled, err := r.applyTTL(ctx, agent, pod, "my-agent-pvc", nil, nil, dur(30*time.Minute), nil, false)
	if err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true (archive fired)")
	}
	got := getAgentT(t, r.Client, "my-agent")
	if got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Errorf("Phase = %q, want Archived", got.Status.Phase)
	}
	if !strings.Contains(got.Status.Message, "archiveTTL") {
		t.Errorf("Message = %q, want it to name archiveTTL", got.Status.Message)
	}
	if got.Status.ArchiveExpiresAt != nil {
		t.Error("ArchiveExpiresAt should be cleared once archived")
	}
	assertGone(t, r.Client, &corev1.Pod{}, "my-agent-pod")
	assertGone(t, r.Client, &corev1.PersistentVolumeClaim{}, "my-agent-pvc")
}

func TestApplyTTL_ArchiveTTLArchivesSleepingAgent(t *testing.T) {
	ctx := context.Background()
	agent, _, pvc, _ := archiveTTLAgent(withPhase(komputerv1alpha1.AgentPhaseSleeping))
	agent.Status.PodName = ""
	r := newTTLReconciler(t, agent, pvc)

	_, handled, err := r.applyTTL(ctx, agent, nil, "my-agent-pvc", dur(10*time.Minute), nil, dur(30*time.Minute), nil, false)
	if err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true (archive fired)")
	}
	if got := getAgentT(t, r.Client, "my-agent"); got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Errorf("Phase = %q, want Archived", got.Status.Phase)
	}
	assertGone(t, r.Client, &corev1.PersistentVolumeClaim{}, "my-agent-pvc")
}

// Both clocks elapsed: archiving subsumes sleeping.
func TestApplyTTL_ArchiveWinsOverSleep(t *testing.T) {
	ctx := context.Background()
	agent, pod, pvc, svc := archiveTTLAgent()
	r := newTTLReconciler(t, agent, pod, pvc, svc)

	if _, _, err := r.applyTTL(ctx, agent, pod, "my-agent-pvc", dur(10*time.Minute), nil, dur(30*time.Minute), nil, false); err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	if got := getAgentT(t, r.Client, "my-agent"); got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Errorf("Phase = %q, want Archived", got.Status.Phase)
	}
}

func TestApplyTTL_DeleteWinsOverArchive(t *testing.T) {
	ctx := context.Background()
	agent, pod, pvc, svc := archiveTTLAgent()
	r := newTTLReconciler(t, agent, pod, pvc, svc)

	if _, _, err := r.applyTTL(ctx, agent, pod, "my-agent-pvc", nil, dur(time.Hour), dur(30*time.Minute), nil, false); err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	assertGone(t, r.Client, &komputerv1alpha1.KomputerAgent{}, "my-agent")
}

// With the S3 session store on, archiving waits for the latest task's save, and
// the sleep clock is held off meanwhile — sleeping would remove the pod that
// still has to save it.
func TestApplyTTL_ArchiveTTLWaitsForS3Save(t *testing.T) {
	ctx := context.Background()
	agent, pod, pvc, svc := archiveTTLAgent(startedTask)
	r, c := saveFirstReconciler(t, nil, nil, agent, pod, pvc, svc, s3KomputerConfig())

	_, handled, err := r.applyTTL(ctx, agent, pod, "my-agent-pvc", dur(10*time.Minute), nil, dur(30*time.Minute), nil, false)
	if err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	if handled {
		t.Fatal("handled = true, want false while the session is unsaved")
	}
	got := getAgentT(t, c, "my-agent")
	if got.Status.Phase != komputerv1alpha1.AgentPhaseRunning {
		t.Errorf("Phase = %q, want Running until the save lands", got.Status.Phase)
	}
	if err := c.Get(ctx, types.NamespacedName{Name: "my-agent-pod", Namespace: "default"}, &corev1.Pod{}); err != nil {
		t.Errorf("pod must be kept until the save lands: %v", err)
	}

	markSaved(got)
	if _, handled, err = r.applyTTL(ctx, got, pod, "my-agent-pvc", dur(10*time.Minute), nil, dur(30*time.Minute), nil, false); err != nil {
		t.Fatalf("applyTTL after save: %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true once saved")
	}
	if got := getAgentT(t, c, "my-agent"); got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Errorf("Phase = %q, want Archived", got.Status.Phase)
	}
}

func TestApplyTTL_PublishesAndClearsArchiveExpiry(t *testing.T) {
	ctx := context.Background()
	agent, pod, pvc, svc := archiveTTLAgent()
	r := newTTLReconciler(t, agent, pod, pvc, svc)

	res, handled, err := r.applyTTL(ctx, agent, pod, "my-agent-pvc", nil, nil, dur(3*time.Hour), nil, false)
	if err != nil || handled {
		t.Fatalf("applyTTL = handled %v, err %v; want a pending deadline only", handled, err)
	}
	if res.RequeueAfter <= time.Hour || res.RequeueAfter > 2*time.Hour {
		t.Errorf("RequeueAfter = %v, want ~2h", res.RequeueAfter)
	}
	got := getAgentT(t, r.Client, "my-agent")
	if got.Status.ArchiveExpiresAt == nil {
		t.Fatal("ArchiveExpiresAt = nil, want the projected deadline")
	}

	if _, _, err := r.applyTTL(ctx, got, pod, "my-agent-pvc", nil, nil, nil, nil, false); err != nil {
		t.Fatalf("applyTTL with no TTLs: %v", err)
	}
	if got := getAgentT(t, r.Client, "my-agent"); got.Status.ArchiveExpiresAt != nil {
		t.Error("ArchiveExpiresAt should be cleared once archiveTTL is removed")
	}
}

// archiveTTL turns lifecycle Archive into "archive after idle", the way sleepTTL
// delays lifecycle Sleep.
func TestReconcileStatus_ArchiveLifecycle_ArchiveTTLDelays(t *testing.T) {
	ctx := context.Background()
	pod, pvc, svc := archiveTestObjects()
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	r := newTTLReconciler(t, agent, pod, pvc, svc)

	if err := r.reconcileStatus(ctx, agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, dur(time.Hour), false); err != nil {
		t.Fatalf("reconcileStatus: %v", err)
	}
	if got := getAgentT(t, r.Client, "my-agent"); got.Status.Phase == komputerv1alpha1.AgentPhaseArchived {
		t.Error("lifecycle Archive with an archiveTTL must wait for the TTL")
	}
}

func TestApplyAgentOverrides_ArchiveTTL(t *testing.T) {
	tpl := &komputerv1alpha1.KomputerAgentTemplate{}
	tpl.Spec.ArchiveTTL = dur(48 * time.Hour)

	if out := applyAgentOverrides(tpl, &komputerv1alpha1.KomputerAgent{}); out.Spec.ArchiveTTL.Duration != 48*time.Hour {
		t.Errorf("archiveTTL = %v, want the template's 48h", out.Spec.ArchiveTTL.Duration)
	}
	agent := &komputerv1alpha1.KomputerAgent{}
	agent.Spec.ArchiveTTL = dur(time.Hour)
	if out := applyAgentOverrides(tpl, agent); out.Spec.ArchiveTTL.Duration != time.Hour {
		t.Errorf("archiveTTL = %v, want the agent's 1h", out.Spec.ArchiveTTL.Duration)
	}
}
