package controller

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// copyOutCommand tars the agent's session out of a container on stdout: the
// .claude dir (minus skills, regenerated from env on every start) and the resume
// pointer, exactly what the agent's own save uploads. It only needs sh and tar,
// so it works on any agent image, including ones that predate the session store.
var copyOutCommand = []string{"sh", "-c", "cd /workspace && tar -czf - --exclude=.claude/skills $(ls -d .claude .komputer-session 2>/dev/null)"}

// defaultMaxConcurrentSavers caps saver pods when the env var isn't set.
const defaultMaxConcurrentSavers = 4

// MaxConcurrentSaversFromEnv reads KOMPUTER_MAX_CONCURRENT_SESSION_SAVERS,
// falling back to the default for a missing or invalid value.
func MaxConcurrentSaversFromEnv() int {
	if n, err := strconv.Atoi(os.Getenv("KOMPUTER_MAX_CONCURRENT_SESSION_SAVERS")); err == nil && n > 0 {
		return n
	}
	return defaultMaxConcurrentSavers
}

// agentContainerName is the container that runs the agent: the first container
// of a solo pod, or the container named after the agent in a squad pod.
func agentContainerName(pod *corev1.Pod, agent *komputerv1alpha1.KomputerAgent) (corev1.Container, bool) {
	for _, c := range pod.Spec.Containers {
		if c.Name == agent.Name {
			return c, true
		}
	}
	if len(pod.Spec.Containers) > 0 && !agent.Status.Squad {
		return pod.Spec.Containers[0], true
	}
	return corev1.Container{}, false
}

// needsCopyOut reports whether the operator has to save an agent's session on
// its behalf: the store is on, the agent's pod is running but predates the S3
// env (so it can't save itself), the agent is idle (a consistent snapshot), and
// its latest task isn't saved yet.
func needsCopyOut(agent *komputerv1alpha1.KomputerAgent, pod *corev1.Pod, s3On bool) bool {
	if !s3On || pod == nil || pod.Status.Phase != corev1.PodRunning || !pod.DeletionTimestamp.IsZero() {
		return false
	}
	if !agent.DeletionTimestamp.IsZero() || agent.Status.Phase == komputerv1alpha1.AgentPhaseArchived || agent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		return false
	}
	c, ok := agentContainerName(pod, agent)
	if !ok || podHasS3Env(c) {
		return false
	}
	return !taskInProgress(agent.Status.TaskStatus) && !komputerv1alpha1.LatestTaskSaved(agent)
}

// copyOutSession streams the session tarball out of a container into the store
// under the agent's key. On success it records status.sessionCopiedOutAt and
// SessionSaved=True; on failure SessionSaved=False with the error.
func (r *KomputerAgentReconciler) copyOutSession(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, s3 *komputerv1alpha1.S3Spec, podName, container string) error {
	err := r.streamSessionToStore(ctx, agent, s3, podName, container)
	now := metav1.Now()
	if statusErr := r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
		cond := metav1.Condition{Type: komputerv1alpha1.ConditionSessionSaved, Status: metav1.ConditionTrue, Reason: "CopiedOut", Message: "Session saved to S3 by the operator"}
		if err != nil {
			cond.Status, cond.Reason, cond.Message = metav1.ConditionFalse, "SaveFailed", err.Error()
		} else {
			s.SessionCopiedOutAt = &now
		}
		meta.SetStatusCondition(&s.Conditions, cond)
	}); statusErr != nil && err == nil {
		return statusErr
	}
	return err
}

func (r *KomputerAgentReconciler) streamSessionToStore(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, s3 *komputerv1alpha1.S3Spec, podName, container string) error {
	if r.Exec == nil || r.NewUploader == nil {
		return fmt.Errorf("session copy-out isn't configured on this operator")
	}
	up, err := r.NewUploader(ctx, s3)
	if err != nil {
		return err
	}
	pr, pw := io.Pipe()
	go func() {
		pw.CloseWithError(r.Exec.ExecStream(ctx, agent.Namespace, podName, container, copyOutCommand, pw))
	}()
	if err := up.Upload(ctx, komputerv1alpha1.SessionStoreKey(s3, agent), pr); err != nil {
		_ = pr.CloseWithError(err)
		return err
	}
	return nil
}

// maybeCopyOut saves the session of an idle agent whose pod can't save itself.
// Failures are recorded on the agent and logged; they never fail the reconcile.
func (r *KomputerAgentReconciler) maybeCopyOut(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, pod *corev1.Pod, config *komputerv1alpha1.KomputerConfig) {
	if !needsCopyOut(agent, pod, config.Spec.S3.IsEnabled()) {
		return
	}
	c, _ := agentContainerName(pod, agent)
	if err := r.copyOutSession(ctx, agent, config.Spec.S3, pod.Name, c.Name); err != nil {
		logf.FromContext(ctx).Info("couldn't copy the session out to S3", "agent", agent.Name, "error", err.Error())
	}
}
