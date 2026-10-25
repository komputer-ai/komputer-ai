package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// Right after a sleep, the deleted pod lingers while it terminates and reports a
// terminal phase. It must not be handed back for the sleeping agent: reconcileStatus
// would copy its Failed/Succeeded phase onto the agent, the next reconcile would see
// an idle non-sleeping agent past sleepTTL and sleep it again — a loop that lasted
// until the pod object was gone.
func TestEnsurePod_SleepingAgentIgnoresTerminatingPod(t *testing.T) {
	ctx := context.Background()
	agent := ttlAgent(withPhase(komputerv1alpha1.AgentPhaseSleeping), withLastActivityAgo(time.Hour))
	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-agent-pod", Namespace: "default",
			DeletionTimestamp: &now, Finalizers: []string{"test/hold"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodFailed},
	}
	r := newTTLReconciler(t, agent, pod)

	got, err := r.ensurePod(ctx, agent, &komputerv1alpha1.KomputerAgentTemplate{}, "test-agent-pvc", "test-agent-pod", &komputerv1alpha1.KomputerConfig{}, false)
	if err != nil {
		t.Fatalf("ensurePod: %v", err)
	}
	if got != nil {
		t.Fatalf("ensurePod returned the terminating pod (phase %s) for a sleeping agent, want nil", got.Status.Phase)
	}

	if err := r.reconcileStatus(ctx, agent, got, "test-agent-pvc", "test-agent-pod", dur(30*time.Minute), nil, nil, false); err != nil {
		t.Fatalf("reconcileStatus: %v", err)
	}
	if phase := getAgentT(t, r.Client, "test-agent").Status.Phase; phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Phase = %q, want it to stay Sleeping while the old pod terminates", phase)
	}
}
