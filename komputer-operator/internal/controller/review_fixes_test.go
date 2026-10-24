package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// The copy-out records when the snapshot was taken (the exec start), not when
// the upload finished: a task that starts during a long copy-out must not
// count as saved by it.
func TestCopyOutSession_RecordsTheSnapshotTime(t *testing.T) {
	agent := copyOutAgent()
	r, c := copyOutReconciler(t, &fakeExecer{out: []byte("tarball-bytes"), delay: 1200 * time.Millisecond}, &fakeUploader{}, agent)

	start := time.Now().Truncate(time.Second)
	if err := r.copyOutSession(context.Background(), agent, copyOutS3, "a1-pod", "agent"); err != nil {
		t.Fatal(err)
	}
	got := getCopyOutAgent(t, c)
	if got.Status.SessionCopiedOutAt == nil || got.Status.SessionCopiedOutAt.After(start) {
		t.Errorf("sessionCopiedOutAt = %v, want the start of the copy-out (%v)", got.Status.SessionCopiedOutAt, start)
	}
}

// A failed idle copy-out must be retried after its backoff: the reconciler
// needs to know when, or nothing wakes it up again for hours.
func TestMaybeCopyOut_FailureAsksForARetry(t *testing.T) {
	agent := copyOutAgent()
	pod := copyOutPod(false)
	r, _ := copyOutReconciler(t, &fakeExecer{err: errors.New("connection reset")}, &fakeUploader{}, agent, pod)
	cfg := &komputerv1alpha1.KomputerConfig{Spec: komputerv1alpha1.KomputerConfigSpec{S3: copyOutS3}}

	if wait := r.maybeCopyOut(context.Background(), agent, pod, cfg); wait <= 0 || wait > copyOutRetryInterval {
		t.Errorf("after a failure: retry in %v, want (0, %v]", wait, copyOutRetryInterval)
	}
	if wait := r.maybeCopyOut(context.Background(), agent, pod, cfg); wait <= 0 {
		t.Errorf("during the backoff: retry in %v, want > 0", wait)
	}
}

// A member added to a running squad runs as an ephemeral container.
func TestAgentContainerName_FindsAnEphemeralSquadContainer(t *testing.T) {
	agent := copyOutAgent()
	agent.Status.Squad = true
	pod := &corev1.Pod{Spec: corev1.PodSpec{
		Containers: []corev1.Container{{Name: "other-member"}},
		EphemeralContainers: []corev1.EphemeralContainer{{EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name: agent.Name, Env: []corev1.EnvVar{{Name: s3KeyEnv, Value: "k"}},
		}}},
	}}
	c, ok := agentContainerName(pod, agent)
	if !ok || c.Name != agent.Name {
		t.Fatalf("got %q, %v; want the ephemeral container %q", c.Name, ok, agent.Name)
	}
}

// If clearing the request failed after the agent was archived, a later wake
// must not archive it again: the archived reconcile clears it.
func TestReconcileArchived_ClearsALeftoverArchiveRequest(t *testing.T) {
	agent := archiveAgent(komputerv1alpha1.AgentTaskComplete)
	agent.Status.Phase = komputerv1alpha1.AgentPhaseArchived
	agent.Annotations = map[string]string{komputerv1alpha1.AnnotationArchiveRequested: "true"}
	r, c := saveFirstReconciler(t, nil, nil, agent, s3KomputerConfig(), agentTemplate())

	if _, err := r.reconcileArchived(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	if _, ok := getAgentT(t, c, "my-agent").Annotations[komputerv1alpha1.AnnotationArchiveRequested]; ok {
		t.Error("the leftover archive request must be cleared")
	}
}

func getCopyOutAgent(t *testing.T, c client.Client) *komputerv1alpha1.KomputerAgent {
	t.Helper()
	var a komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &a); err != nil {
		t.Fatal(err)
	}
	return &a
}
