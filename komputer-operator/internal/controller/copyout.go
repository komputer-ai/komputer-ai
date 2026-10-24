package controller

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

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
// With no session files it prints nothing (GNU tar refuses an empty archive),
// and tar's exit 1 ("file changed as we read it" — the archive is still usable)
// counts as success.
var copyOutCommand = []string{"sh", "-c", `cd /workspace || exit 1
f=$(ls -d .claude .komputer-session 2>/dev/null)
[ -z "$f" ] && exit 0
tar -czf - --exclude=.claude/skills $f
rc=$?
[ $rc -le 1 ] && exit 0
exit $rc`}

// copyOutRetryInterval is how long a failed copy-out waits before the next
// attempt. Each failure writes a fresh status message, which would otherwise
// trigger the next reconcile — and attempt — immediately.
const copyOutRetryInterval = time.Minute

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
// its behalf: the store is on, the agent's pod is running, the agent is idle (a
// consistent snapshot), and its latest task isn't saved. That covers pods that
// can't save themselves (they predate the S3 env, or run an older image) and
// pods that can but didn't (a cancel that lost the race, a crash, a failed save).
func needsCopyOut(agent *komputerv1alpha1.KomputerAgent, pod *corev1.Pod, s3On bool) bool {
	if !s3On || pod == nil || pod.Status.Phase != corev1.PodRunning || !pod.DeletionTimestamp.IsZero() {
		return false
	}
	if !agent.DeletionTimestamp.IsZero() || agent.Status.Phase == komputerv1alpha1.AgentPhaseArchived || agent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		return false
	}
	if _, ok := agentContainerName(pod, agent); !ok {
		return false
	}
	return !taskInProgress(agent.Status.TaskStatus) && !komputerv1alpha1.LatestTaskSaved(agent)
}

// copyOutSession streams the session tarball out of a container into the store
// under the agent's key. On success it records status.sessionCopiedOutAt and
// SessionSaved=True; on failure SessionSaved=False with the error.
func (r *KomputerAgentReconciler) copyOutSession(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, s3 *komputerv1alpha1.S3Spec, podName, container string) error {
	// Bounded: the agent controller reconciles one agent at a time, so a hung
	// exec or a stalled store must not hold every other agent up.
	copyCtx, cancel := context.WithTimeout(ctx, copyOutTimeout)
	err := r.streamSessionToStore(copyCtx, agent, s3, podName, container)
	cancel()
	if err != nil {
		r.copyOutRetryAt.Store(agent.UID, time.Now().Add(copyOutRetryInterval))
	} else {
		r.copyOutRetryAt.Delete(agent.UID)
	}
	now := metav1.Now()
	if statusErr := r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
		cond := metav1.Condition{Type: komputerv1alpha1.ConditionSessionSaved, Status: metav1.ConditionTrue, Reason: "CopiedOut", Message: "Session saved to S3 by the operator"}
		if err != nil {
			cond.Status, cond.Reason, cond.Message = metav1.ConditionFalse, "SaveFailed", copyOutFailureMessage(agent, err)
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
	body := bufio.NewReader(pr)
	if _, err := body.Peek(1); err != nil {
		_ = pr.CloseWithError(err)
		if errors.Is(err, io.EOF) {
			// No session files in the container: nothing to save — and never
			// overwrite a session already in the store with an empty object.
			return nil
		}
		return err
	}
	// Keep the current latest as previous before overwriting it. Best effort:
	// a failed copy never stops the save.
	latest := komputerv1alpha1.SessionStoreKey(s3, agent)
	if err := up.Copy(ctx, latest, komputerv1alpha1.SessionStorePreviousKey(s3, agent)); err != nil {
		logf.FromContext(ctx).Info("couldn't keep the previous session in S3", "agent", agent.Name, "error", err.Error())
	}
	if err := up.Upload(ctx, latest, body); err != nil {
		_ = pr.CloseWithError(err)
		return err
	}
	return nil
}

// copyOutDue reports whether a copy-out may be attempted now, or how long until
// the backoff after the last failure ends.
func (r *KomputerAgentReconciler) copyOutDue(agent *komputerv1alpha1.KomputerAgent) (bool, time.Duration) {
	v, ok := r.copyOutRetryAt.Load(agent.UID)
	if !ok {
		return true, 0
	}
	if wait := time.Until(v.(time.Time)); wait > 0 {
		return false, wait
	}
	return true, 0
}

// maybeCopyOut saves the session of an idle agent whose pod can't save itself.
// Failures are recorded on the agent and logged; they never fail the reconcile.
func (r *KomputerAgentReconciler) maybeCopyOut(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, pod *corev1.Pod, config *komputerv1alpha1.KomputerConfig) {
	if !needsCopyOut(agent, pod, config.Spec.S3.IsEnabled()) {
		return
	}
	if due, _ := r.copyOutDue(agent); !due {
		return
	}
	c, _ := agentContainerName(pod, agent)
	if err := r.copyOutSession(ctx, agent, config.Spec.S3, pod.Name, c.Name); err != nil {
		logf.FromContext(ctx).Info("couldn't copy the session out to S3", "agent", agent.Name, "error", err.Error())
	}
}

// copyOutFailureMessage says what a failed save is holding back: for a live
// Archive agent, the archive (which would delete its workspace).
func copyOutFailureMessage(agent *komputerv1alpha1.KomputerAgent, err error) string {
	if agent.Spec.Lifecycle == komputerv1alpha1.AgentLifecycleArchive && agent.DeletionTimestamp.IsZero() {
		return "Archive stopped: couldn't save the session to S3: " + err.Error()
	}
	return "Couldn't save the session to S3: " + err.Error()
}
