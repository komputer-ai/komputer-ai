package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// Deletes wait for the session. While the S3 session store is configured every
// agent carries the komputer.ai/session-store finalizer, so any delete — the
// API, deleteTTL, AutoDelete, an office or namespace delete — first saves the
// latest task's session to S3 (sessions outlive their agents). DELETE
// /agents/{name}?force=true drops the finalizer to skip it.

const (
	sessionSaverContainer     = "saver"
	sessionSaverDeadline      = int64(300)
	sessionFinalizerRetry     = 30 * time.Second
	sessionSaverSlotRetry     = 15 * time.Second
	sessionFinalizerShortWait = 5 * time.Second
	deletingWaitingForSlot    = "Deleting — waiting for a session saver slot"
	deletingWaitingToSave     = "Deleting — waiting to save the session to S3: "
	errPVCAlreadyDeleting     = "workspace volume is already being deleted"
)

// ensureSessionFinalizer adds the finalizer to a live agent when S3 is on.
// Returns whether it changed the object.
func (r *KomputerAgentReconciler) ensureSessionFinalizer(ctx context.Context, agent *komputerv1alpha1.KomputerAgent) (bool, error) {
	if !agent.DeletionTimestamp.IsZero() || controllerutil.ContainsFinalizer(agent, komputerv1alpha1.FinalizerSessionStore) {
		return false, nil
	}
	if !s3SpecFromCluster(ctx, r.Client).IsEnabled() {
		return false, nil
	}
	original := agent.DeepCopy()
	controllerutil.AddFinalizer(agent, komputerv1alpha1.FinalizerSessionStore)
	return true, r.Patch(ctx, agent, client.MergeFrom(original))
}

// handleSessionFinalizer runs while a deleting agent still carries the
// finalizer: it saves the latest session (from the agent's pod, or through a
// short-lived saver pod that mounts its PVC), then releases the delete.
func (r *KomputerAgentReconciler) handleSessionFinalizer(ctx context.Context, agent *komputerv1alpha1.KomputerAgent) (ctrl.Result, error) {
	s3 := s3SpecFromCluster(ctx, r.Client)
	if !s3.IsEnabled() {
		return ctrl.Result{}, r.releaseSessionFinalizer(ctx, agent)
	}
	pod := r.liveAgentPod(ctx, agent)
	if taskInProgress(agent.Status.TaskStatus) && pod != nil {
		// The agent saves as part of ending the task; cancel it and wait. (With
		// no live pod the "in progress" task is a leftover from a pod that died
		// mid-task — there's nothing to cancel, so go on and save from the PVC.)
		if err := cancelAgentTaskViaAPI(ctx, r.Client, agent.Namespace, agent.Name); err != nil {
			logf.FromContext(ctx).Info("couldn't cancel the task of a deleting agent; will retry", "agent", agent.Name, "error", err.Error())
		}
		return ctrl.Result{RequeueAfter: sessionFinalizerShortWait}, nil
	}
	if komputerv1alpha1.LatestTaskSaved(agent) || agent.Status.Phase == komputerv1alpha1.AgentPhaseArchived {
		// Saved, or nothing local left to save (an archive only happens after a save).
		return ctrl.Result{}, r.releaseSessionFinalizer(ctx, agent)
	}

	if due, wait := r.copyOutDue(agent); !due {
		return ctrl.Result{RequeueAfter: wait}, nil
	}

	if pod != nil {
		c, ok := agentContainerName(pod, agent)
		if !ok {
			return r.waitToSave(ctx, agent, fmt.Errorf("no agent container in pod %s", pod.Name))
		}
		if err := r.copyOutSession(ctx, agent, s3, pod.Name, c.Name); err != nil {
			return r.waitToSave(ctx, agent, err)
		}
		return ctrl.Result{}, r.releaseSessionFinalizer(ctx, agent)
	}

	pvcName := agent.Status.PvcName
	if pvcName == "" {
		pvcName = agent.Name + "-pvc"
	}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: agent.Namespace}, pvc); err != nil {
		if apierrors.IsNotFound(err) {
			// No pod and no volume: there's no local copy left to save.
			return ctrl.Result{}, r.releaseSessionFinalizer(ctx, agent)
		}
		return ctrl.Result{}, err
	}
	if !pvc.DeletionTimestamp.IsZero() {
		// A volume being deleted can't be mounted by a new pod.
		return r.waitToSave(ctx, agent, errors.New(errPVCAlreadyDeleting))
	}

	saver, slot, err := r.ensureSaverPod(ctx, agent, pvcName)
	if err != nil {
		return r.waitToSave(ctx, agent, err)
	}
	if !slot {
		if err := r.setDeletingMessage(ctx, agent, deletingWaitingForSlot); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: sessionSaverSlotRetry}, nil
	}
	switch saver.Status.Phase {
	case corev1.PodRunning:
	case corev1.PodFailed, corev1.PodSucceeded:
		// Ran out its deadline without being used: start over.
		_ = r.Delete(ctx, saver)
		return ctrl.Result{RequeueAfter: sessionFinalizerShortWait}, nil
	default:
		return ctrl.Result{RequeueAfter: sessionFinalizerShortWait}, nil
	}
	if err := r.copyOutSession(ctx, agent, s3, saver.Name, sessionSaverContainer); err != nil {
		return r.waitToSave(ctx, agent, err)
	}
	if err := r.Delete(ctx, saver); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, r.releaseSessionFinalizer(ctx, agent)
}

// liveAgentPod returns the agent's running, non-terminating pod (solo pod, or
// the squad pod it's a member of), or nil.
func (r *KomputerAgentReconciler) liveAgentPod(ctx context.Context, agent *komputerv1alpha1.KomputerAgent) *corev1.Pod {
	name := agent.Status.PodName
	if name == "" {
		return nil
	}
	pod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: agent.Namespace}, pod); err != nil {
		return nil
	}
	if pod.Status.Phase != corev1.PodRunning || !pod.DeletionTimestamp.IsZero() {
		return nil
	}
	return pod
}

// ensureSaverPod returns this agent's saver pod, creating it if a slot is free.
// slot is false when MaxConcurrentSavers saver pods already exist cluster-wide;
// they're counted by label so the cap survives operator restarts.
func (r *KomputerAgentReconciler) ensureSaverPod(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, pvcName string) (*corev1.Pod, bool, error) {
	name := agent.Name + "-session-saver"
	existing := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: agent.Namespace}, existing); err == nil {
		return existing, true, nil
	} else if !apierrors.IsNotFound(err) {
		return nil, false, err
	}

	var savers corev1.PodList
	if err := r.List(ctx, &savers, client.MatchingLabels{komputerv1alpha1.LabelSessionSaver: "true"}); err != nil {
		return nil, false, err
	}
	limit := r.MaxConcurrentSavers
	if limit <= 0 {
		limit = defaultMaxConcurrentSavers
	}
	if len(savers.Items) >= limit {
		return nil, false, nil
	}

	templateRef := agent.Spec.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}
	tpl, err := readTemplateSpec(ctx, r.Client, templateRef, agent.Namespace)
	if err != nil {
		return nil, false, fmt.Errorf("resolve template %q for the saver pod: %w", templateRef, err)
	}
	if len(tpl.PodSpec.Containers) == 0 {
		return nil, false, fmt.Errorf("template %q has no containers", templateRef)
	}
	deadline := sessionSaverDeadline
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: agent.Namespace,
			Labels: map[string]string{komputerv1alpha1.LabelSessionSaver: "true", "komputer.ai/agent-name": agent.Name},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:         corev1.RestartPolicyNever,
			ActiveDeadlineSeconds: &deadline,
			SecurityContext:       tpl.PodSpec.SecurityContext.DeepCopy(),
			ImagePullSecrets:      tpl.PodSpec.ImagePullSecrets,
			NodeSelector:          tpl.PodSpec.NodeSelector,
			Tolerations:           tpl.PodSpec.Tolerations,
			Containers: []corev1.Container{{
				Name:    sessionSaverContainer,
				Image:   tpl.PodSpec.Containers[0].Image,
				Command: []string{"sleep", fmt.Sprint(sessionSaverDeadline)},
				Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("10m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				}},
				VolumeMounts: []corev1.VolumeMount{{Name: "workspace", MountPath: "/workspace", ReadOnly: true}},
			}},
			Volumes: []corev1.Volume{{Name: "workspace", VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName, ReadOnly: true},
			}}},
		},
	}
	if err := ctrl.SetControllerReference(agent, pod, r.Scheme); err != nil {
		return nil, false, err
	}
	if err := r.Create(ctx, pod); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, false, err
	}
	return pod, true, nil
}

func (r *KomputerAgentReconciler) waitToSave(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, cause error) (ctrl.Result, error) {
	if err := r.setDeletingMessage(ctx, agent, deletingWaitingToSave+cause.Error()); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: sessionFinalizerRetry}, nil
}

func (r *KomputerAgentReconciler) setDeletingMessage(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, msg string) error {
	return r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) { s.Message = msg })
}

func (r *KomputerAgentReconciler) releaseSessionFinalizer(ctx context.Context, agent *komputerv1alpha1.KomputerAgent) error {
	if !controllerutil.ContainsFinalizer(agent, komputerv1alpha1.FinalizerSessionStore) {
		return nil
	}
	original := agent.DeepCopy()
	controllerutil.RemoveFinalizer(agent, komputerv1alpha1.FinalizerSessionStore)
	if err := r.Patch(ctx, agent, client.MergeFrom(original)); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
