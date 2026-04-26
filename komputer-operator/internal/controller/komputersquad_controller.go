/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

const defaultOrphanTTL = 10 * time.Minute

// KomputerSquadReconciler reconciles a KomputerSquad object.
type KomputerSquadReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputersquads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputersquads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputersquads/finalizers,verbs=update

// Reconcile moves the cluster state toward the desired state for a KomputerSquad.
func (r *KomputerSquadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("squad", req.NamespacedName)

	// 1. Fetch the KomputerSquad CR
	squad := &komputerv1alpha1.KomputerSquad{}
	if err := r.Get(ctx, req.NamespacedName, squad); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Handle deletion: clear Phase=Squad on all member agents before the squad is gone.
	if !squad.DeletionTimestamp.IsZero() {
		for _, member := range squad.Spec.Members {
			if member.Ref == nil {
				continue
			}
			ns := member.Ref.Namespace
			if ns == "" {
				ns = squad.Namespace
			}
			if err := r.clearSquadPhase(ctx, member.Ref.Name, ns); err != nil {
				log.Error(err, "Failed to clear Squad phase on agent during deletion", "agent", member.Ref.Name)
			}
		}
		return ctrl.Result{}, nil
	}

	// 3. Normalize members: convert embedded spec members → create KomputerAgent + convert to ref.
	if err := r.normalizeMembers(ctx, squad); err != nil {
		log.Error(err, "Failed to normalize squad members")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Re-fetch squad after normalizeMembers to get the latest resourceVersion,
	// avoiding stale-resourceVersion 409s on subsequent status patches.
	if err := r.Get(ctx, req.NamespacedName, squad); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 4. Empty squad handling (after normalization so we count refs only).
	if len(squad.Spec.Members) == 0 {
		return r.handleEmptySquad(ctx, squad)
	}

	// 5. Single-member shrinkage: dissolve the squad, hand the lone agent back to agent controller.
	if len(squad.Spec.Members) == 1 {
		return r.handleSingleMemberShrinkage(ctx, squad)
	}

	// 6. Mark all member agents as squad-managed (Phase=Squad).
	agents, err := r.markMembersAsSquad(ctx, squad)
	if err != nil {
		log.Error(err, "Failed to mark members as squad")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// If no agents resolved yet (all NotFound), requeue.
	if len(agents) == 0 {
		log.Info("No member agents resolved yet, requeueing")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// 7. Get KomputerConfig (needed for env vars in containers).
	komputerConfig, err := r.getSquadConfig(ctx)
	if err != nil {
		log.Error(err, "Failed to get KomputerConfig")
		_ = r.updateSquadStatus(ctx, squad, func(s *komputerv1alpha1.KomputerSquadStatus) {
			s.Phase = komputerv1alpha1.SquadPhasePending
			s.Message = "No KomputerConfig found in the cluster"
		})
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// 8. Ensure all member PVCs exist.
	if err := r.ensureMemberPVCs(ctx, squad, agents); err != nil {
		log.Error(err, "Failed to ensure member PVCs")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// 9. Reconcile the squad Pod.
	if err := r.reconcileSquadPod(ctx, squad, agents, komputerConfig); err != nil {
		log.Error(err, "Failed to reconcile squad pod")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// 10. Update squad status.
	if err := r.updateSquadMemberStatus(ctx, squad, agents); err != nil {
		log.Error(err, "Failed to update squad status")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// normalizeMembers converts any member with an embedded spec into a KomputerAgent CR,
// then mutates the squad to replace spec with a ref. The squad spec patch is applied
// immediately so subsequent reconciles only see refs.
func (r *KomputerSquadReconciler) normalizeMembers(ctx context.Context, squad *komputerv1alpha1.KomputerSquad) error {
	log := logf.FromContext(ctx)
	needsUpdate := false

	for i, member := range squad.Spec.Members {
		if member.Spec == nil {
			continue // already a ref, nothing to do
		}

		// Generate a deterministic agent name: <squad-name>-member-<index>
		agentName := fmt.Sprintf("%s-member-%d", squad.Name, i)

		// Create the KomputerAgent if it does not exist yet
		existing := &komputerv1alpha1.KomputerAgent{}
		err := r.Get(ctx, types.NamespacedName{Name: agentName, Namespace: squad.Namespace}, existing)
		if apierrors.IsNotFound(err) {
			agent := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      agentName,
					Namespace: squad.Namespace,
					Labels: map[string]string{
						"komputer.ai/squad": squad.Name,
					},
				},
				Spec: *member.Spec.DeepCopy(),
			}
			if createErr := r.Create(ctx, agent); createErr != nil && !apierrors.IsAlreadyExists(createErr) {
				return fmt.Errorf("create agent %s for squad %s: %w", agentName, squad.Name, createErr)
			}
			log.Info("Created KomputerAgent for squad member", "agent", agentName, "squad", squad.Name)
		} else if err != nil {
			return fmt.Errorf("get agent %s: %w", agentName, err)
		}

		// Convert spec → ref in the local copy
		squad.Spec.Members[i] = komputerv1alpha1.KomputerSquadMember{
			Ref: &komputerv1alpha1.KomputerSquadMemberRef{
				Name:      agentName,
				Namespace: squad.Namespace,
			},
		}
		needsUpdate = true
	}

	if !needsUpdate {
		return nil
	}

	// Patch the squad spec. We can't use MergeFrom because we changed Spec,
	// not Status. Use Update instead (with conflict tolerance via requeue).
	// Re-fetch to get latest resourceVersion before updating.
	latest := &komputerv1alpha1.KomputerSquad{}
	if err := r.Get(ctx, types.NamespacedName{Name: squad.Name, Namespace: squad.Namespace}, latest); err != nil {
		return fmt.Errorf("re-fetch squad before spec update: %w", err)
	}
	latest.Spec.Members = squad.Spec.Members
	if err := r.Update(ctx, latest); err != nil {
		if apierrors.IsConflict(err) {
			return fmt.Errorf("conflict updating squad spec (will retry): %w", err)
		}
		return fmt.Errorf("update squad spec: %w", err)
	}
	// Reflect the updated resourceVersion back into the local squad so callers
	// don't try to patch stale objects.
	squad.ResourceVersion = latest.ResourceVersion

	return nil
}

// markMembersAsSquad sets Phase=Squad on each referenced agent and returns the
// resolved KomputerAgent objects (skipping any not found yet).
func (r *KomputerSquadReconciler) markMembersAsSquad(ctx context.Context, squad *komputerv1alpha1.KomputerSquad) ([]*komputerv1alpha1.KomputerAgent, error) {
	log := logf.FromContext(ctx)
	agents := make([]*komputerv1alpha1.KomputerAgent, 0, len(squad.Spec.Members))

	for _, member := range squad.Spec.Members {
		if member.Ref == nil {
			continue
		}
		ns := member.Ref.Namespace
		if ns == "" {
			ns = squad.Namespace
		}

		agent := &komputerv1alpha1.KomputerAgent{}
		if err := r.Get(ctx, types.NamespacedName{Name: member.Ref.Name, Namespace: ns}, agent); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("Member agent not found yet, skipping", "agent", member.Ref.Name)
				continue
			}
			return nil, fmt.Errorf("get agent %s: %w", member.Ref.Name, err)
		}

		// Set Phase=Squad if not already set
		if agent.Status.Phase != komputerv1alpha1.KomputerAgentPhaseSquad {
			original := agent.DeepCopy()
			agent.Status.Phase = komputerv1alpha1.KomputerAgentPhaseSquad
			agent.Status.Message = fmt.Sprintf("Managed by squad %s", squad.Name)
			if err := r.Status().Patch(ctx, agent, client.MergeFrom(original)); err != nil {
				// Any error (including conflict) means we have a partial set — return
				// immediately so controller-runtime requeues the entire reconcile.
				return nil, fmt.Errorf("patch agent %s phase to Squad: %w", agent.Name, err)
			}
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// ensureMemberPVCs ensures each member agent has a PVC. Reuses the agent's existing PVC
// (from status.pvcName) if set; otherwise creates <agentName>-pvc.
func (r *KomputerSquadReconciler) ensureMemberPVCs(ctx context.Context, squad *komputerv1alpha1.KomputerSquad, agents []*komputerv1alpha1.KomputerAgent) error {
	for _, agent := range agents {
		pvcName := agent.Status.PvcName
		if pvcName == "" {
			pvcName = agent.Name + "-pvc"
		}

		pvc := &corev1.PersistentVolumeClaim{}
		err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: agent.Namespace}, pvc)
		if err == nil {
			// PVC exists — ensure agent status has pvcName set
			if agent.Status.PvcName != pvcName {
				original := agent.DeepCopy()
				agent.Status.PvcName = pvcName
				if patchErr := r.Status().Patch(ctx, agent, client.MergeFrom(original)); patchErr != nil && !apierrors.IsConflict(patchErr) {
					return fmt.Errorf("patch agent %s pvcName: %w", agent.Name, patchErr)
				}
			}
			continue
		}
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("get pvc %s: %w", pvcName, err)
		}

		// Create PVC with default size (5Gi)
		qty := resource.MustParse("5Gi")
		newPVC := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: agent.Namespace,
				Labels: map[string]string{
					"komputer.ai/agent-name": agent.Name,
					"komputer.ai/squad":      squad.Name,
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: qty,
					},
				},
			},
		}
		// Set the agent as owner so the PVC is GC'd when the agent is deleted
		// (matches the solo agent flow in ensurePVC).
		if err := ctrl.SetControllerReference(agent, newPVC, r.Scheme); err != nil {
			return fmt.Errorf("set owner ref on pvc %s: %w", pvcName, err)
		}
		if createErr := r.Create(ctx, newPVC); createErr != nil && !apierrors.IsAlreadyExists(createErr) {
			return fmt.Errorf("create pvc %s: %w", pvcName, createErr)
		}

		// Update agent status with pvcName
		original := agent.DeepCopy()
		agent.Status.PvcName = pvcName
		if patchErr := r.Status().Patch(ctx, agent, client.MergeFrom(original)); patchErr != nil && !apierrors.IsConflict(patchErr) {
			return fmt.Errorf("patch agent %s pvcName after create: %w", agent.Name, patchErr)
		}
	}
	return nil
}

// reconcileSquadPod ensures the squad Pod exists with the correct set of containers.
// If membership changed since pod creation, the old pod is deleted; a new one will
// be created on the next reconcile loop (after the pod is fully removed).
func (r *KomputerSquadReconciler) reconcileSquadPod(ctx context.Context, squad *komputerv1alpha1.KomputerSquad, agents []*komputerv1alpha1.KomputerAgent, config *komputerv1alpha1.KomputerConfig) error {
	log := logf.FromContext(ctx)
	podName := squad.Name + "-pod"
	ns := squad.Namespace

	existing := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: ns}, existing)

	if err == nil {
		// Pod exists — check if membership changed by comparing container names
		if membershipChanged(existing, agents) {
			log.Info("Squad membership changed, deleting pod for recreation", "pod", podName)
			if delErr := r.Delete(ctx, existing); delErr != nil && !apierrors.IsNotFound(delErr) {
				return fmt.Errorf("delete stale squad pod %s: %w", podName, delErr)
			}
			// Will be created on next reconcile
		}
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get squad pod %s: %w", podName, err)
	}

	// Pod does not exist — build and create it
	desired, buildErr := r.buildSquadPodSpec(ctx, squad, agents, config)
	if buildErr != nil {
		return fmt.Errorf("build squad pod spec: %w", buildErr)
	}

	// Set the squad CR as owner so the pod is GC'd if the squad is deleted.
	if err := controllerutil.SetControllerReference(squad, desired, r.Scheme); err != nil {
		return fmt.Errorf("set owner ref on squad pod %s: %w", podName, err)
	}

	log.Info("Creating squad pod", "pod", podName)
	if createErr := r.Create(ctx, desired); createErr != nil && !apierrors.IsAlreadyExists(createErr) {
		return fmt.Errorf("create squad pod %s: %w", podName, createErr)
	}
	return nil
}

// membershipChanged returns true if the pod's containers don't exactly match the agent list.
func membershipChanged(pod *corev1.Pod, agents []*komputerv1alpha1.KomputerAgent) bool {
	if len(pod.Spec.Containers) != len(agents) {
		return true
	}
	podContainerNames := make(map[string]bool, len(pod.Spec.Containers))
	for _, c := range pod.Spec.Containers {
		podContainerNames[c.Name] = true
	}
	for _, a := range agents {
		if !podContainerNames[a.Name] {
			return true
		}
	}
	return false
}

// buildSquadPodSpec constructs the desired Pod for the squad. Each agent gets
// its own container (named after the agent). Volume mounts:
//   - Own PVC at /workspace
//   - Each sibling's PVC at /agents/<sibling-name>/workspace
//
// Container spec is derived from the agent's resolved template (first container).
//
// Each container gets the same env vars as a solo agent (secrets, connectors,
// MCP servers, OAuth tokens) via the shared buildAgentEnvVars helper.
func (r *KomputerSquadReconciler) buildSquadPodSpec(ctx context.Context, squad *komputerv1alpha1.KomputerSquad, agents []*komputerv1alpha1.KomputerAgent, config *komputerv1alpha1.KomputerConfig) (*corev1.Pod, error) {
	log := logf.FromContext(ctx)
	podName := squad.Name + "-pod"

	// Build the Volumes block: one PVC-backed volume per agent.
	volumes := make([]corev1.Volume, 0, len(agents))
	for _, agent := range agents {
		pvcName := agent.Status.PvcName
		if pvcName == "" {
			pvcName = agent.Name + "-pvc"
		}
		volumes = append(volumes, corev1.Volume{
			Name: agent.Name + "-workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		})
	}

	// Build one container per agent.
	containers := make([]corev1.Container, 0, len(agents))

	for _, agent := range agents {
		// Resolve the agent's template to get the base container spec (image, resources, etc.)
		templateRef := agent.Spec.TemplateRef
		if templateRef == "" {
			templateRef = "default"
		}
		template := &komputerv1alpha1.KomputerAgentTemplate{}
		if err := r.Get(ctx, types.NamespacedName{Name: templateRef, Namespace: agent.Namespace}, template); err != nil {
			// Fall back to cluster-scoped template
			clusterTemplate := &komputerv1alpha1.KomputerAgentClusterTemplate{}
			if clusterErr := r.Get(ctx, types.NamespacedName{Name: templateRef}, clusterTemplate); clusterErr != nil {
				log.Error(err, "Template not found for squad member", "agent", agent.Name, "templateRef", templateRef)
				return nil, fmt.Errorf("template %q not found for agent %s", templateRef, agent.Name)
			}
			template = &komputerv1alpha1.KomputerAgentTemplate{
				Spec: *clusterTemplate.Spec.DeepCopy(),
			}
		}

		if len(template.Spec.PodSpec.Containers) == 0 {
			return nil, fmt.Errorf("template %q for agent %s has no containers defined", templateRef, agent.Name)
		}

		// Deep-copy the first container from the template; rename it to the agent name.
		c := *template.Spec.PodSpec.Containers[0].DeepCopy()
		c.Name = agent.Name

		// Build env vars via the shared helper — picks up secrets, MCP connectors,
		// OAuth tokens, etc. exactly like a solo agent does.
		envVars, err := buildAgentEnvVars(ctx, r.Client, agent, config)
		if err != nil {
			return nil, fmt.Errorf("failed to build env vars for squad member %s: %w", agent.Name, err)
		}
		c.Env = mergeEnvVars(c.Env, envVars)

		// Volume mounts: own PVC at /workspace, siblings at /agents/<sibling>/workspace.
		c.VolumeMounts = append(c.VolumeMounts,
			corev1.VolumeMount{
				Name:      agent.Name + "-workspace",
				MountPath: "/workspace",
			},
		)
		for _, sibling := range agents {
			if sibling.Name == agent.Name {
				continue
			}
			c.VolumeMounts = append(c.VolumeMounts,
				corev1.VolumeMount{
					Name:      sibling.Name + "-workspace",
					MountPath: "/agents/" + sibling.Name + "/workspace",
					ReadOnly:  false,
				},
			)
		}

		containers = append(containers, c)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: squad.Namespace,
			Labels: map[string]string{
				"komputer.ai/squad":      "true",
				"komputer.ai/squad-name": squad.Name,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes:       volumes,
			Containers:    containers,
		},
	}

	return pod, nil
}

// updateSquadMemberStatus refreshes squad.status.phase, podName, and members[].
func (r *KomputerSquadReconciler) updateSquadMemberStatus(ctx context.Context, squad *komputerv1alpha1.KomputerSquad, agents []*komputerv1alpha1.KomputerAgent) error {
	podName := squad.Name + "-pod"

	pod := &corev1.Pod{}
	squadPhase := komputerv1alpha1.SquadPhasePending
	podExists := false
	if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: squad.Namespace}, pod); err == nil {
		podExists = true
		switch pod.Status.Phase {
		case corev1.PodRunning:
			squadPhase = komputerv1alpha1.SquadPhaseRunning
		case corev1.PodFailed:
			squadPhase = komputerv1alpha1.SquadPhaseFailed
		}
	}

	memberStatuses := make([]komputerv1alpha1.KomputerSquadMemberStatus, 0, len(agents))
	for _, agent := range agents {
		ready := false
		if podExists {
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.Name == agent.Name && cs.Ready {
					ready = true
					break
				}
			}
		}
		memberStatuses = append(memberStatuses, komputerv1alpha1.KomputerSquadMemberStatus{
			Name:       agent.Name,
			Ready:      ready,
			TaskStatus: string(agent.Status.TaskStatus),
		})
	}

	// Stable output order
	sort.Slice(memberStatuses, func(i, j int) bool {
		return memberStatuses[i].Name < memberStatuses[j].Name
	})

	return r.updateSquadStatus(ctx, squad, func(s *komputerv1alpha1.KomputerSquadStatus) {
		s.Phase = squadPhase
		s.PodName = podName
		s.Members = memberStatuses
		s.OrphanedSince = nil
		s.Message = ""
	})
}

// handleEmptySquad implements the orphan TTL. Stamps orphanedSince on first call;
// deletes the squad once the TTL elapses.
func (r *KomputerSquadReconciler) handleEmptySquad(ctx context.Context, squad *komputerv1alpha1.KomputerSquad) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	ttl := defaultOrphanTTL
	if squad.Spec.OrphanTTL != nil {
		ttl = squad.Spec.OrphanTTL.Duration
	}

	now := metav1.Now()

	if squad.Status.OrphanedSince == nil {
		log.Info("Squad has no members, marking orphaned", "squad", squad.Name)
		if err := r.updateSquadStatus(ctx, squad, func(s *komputerv1alpha1.KomputerSquadStatus) {
			s.Phase = komputerv1alpha1.SquadPhaseOrphaned
			s.OrphanedSince = &now
			s.Message = "Squad has no members"
		}); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: ttl}, nil
	}

	if now.Time.After(squad.Status.OrphanedSince.Time.Add(ttl)) {
		log.Info("Squad orphan TTL elapsed, deleting squad", "squad", squad.Name)
		return ctrl.Result{}, r.Delete(ctx, squad)
	}

	remaining := squad.Status.OrphanedSince.Time.Add(ttl).Sub(now.Time)
	if remaining < 0 {
		remaining = 0
	}
	return ctrl.Result{RequeueAfter: remaining}, nil
}

// handleSingleMemberShrinkage dissolves the squad when exactly 1 member remains.
// Clears Phase=Squad on the lone agent (so the agent controller picks it up),
// deletes the squad Pod, and deletes the squad CR.
func (r *KomputerSquadReconciler) handleSingleMemberShrinkage(ctx context.Context, squad *komputerv1alpha1.KomputerSquad) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if len(squad.Spec.Members) != 1 || squad.Spec.Members[0].Ref == nil {
		// Shouldn't happen, but guard against it
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	ref := squad.Spec.Members[0].Ref
	ns := ref.Namespace
	if ns == "" {
		ns = squad.Namespace
	}

	log.Info("Squad has only 1 member, dissolving", "squad", squad.Name, "agent", ref.Name)

	// 1. Delete the squad pod FIRST so the RWO PVC is released before the agent
	//    controller tries to schedule a new solo pod for the lone agent.
	podName := squad.Name + "-pod"
	pod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: squad.Namespace}, pod); err == nil {
		if delErr := r.Delete(ctx, pod); delErr != nil && !apierrors.IsNotFound(delErr) {
			log.Error(delErr, "Failed to delete squad pod during single-member shrinkage", "pod", podName)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}

	// 2. Clear Phase=Squad on the lone agent so the agent controller picks it up.
	if err := r.clearSquadPhase(ctx, ref.Name, ns); err != nil {
		log.Error(err, "Failed to clear Squad phase on lone member", "agent", ref.Name)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// 3. Delete the squad CR.
	return ctrl.Result{}, r.Delete(ctx, squad)
}

// clearSquadPhase sets agent.status.phase = "" so the agent controller resets it to Pending.
func (r *KomputerSquadReconciler) clearSquadPhase(ctx context.Context, agentName, namespace string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	if err := r.Get(ctx, types.NamespacedName{Name: agentName, Namespace: namespace}, agent); err != nil {
		if apierrors.IsNotFound(err) {
			return nil // agent already gone, nothing to clear
		}
		return err
	}
	if agent.Status.Phase != komputerv1alpha1.KomputerAgentPhaseSquad {
		return nil // phase already cleared
	}
	original := agent.DeepCopy()
	agent.Status.Phase = ""
	agent.Status.Message = ""
	return r.Status().Patch(ctx, agent, client.MergeFrom(original))
}

// getSquadConfig returns the singleton KomputerConfig in the cluster.
func (r *KomputerSquadReconciler) getSquadConfig(ctx context.Context) (*komputerv1alpha1.KomputerConfig, error) {
	list := &komputerv1alpha1.KomputerConfigList{}
	if err := r.List(ctx, list); err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("no KomputerConfig found in the cluster")
	}
	return &list.Items[0], nil
}

// updateSquadStatus patches squad.status using a mutator function.
func (r *KomputerSquadReconciler) updateSquadStatus(ctx context.Context, squad *komputerv1alpha1.KomputerSquad, mutate func(*komputerv1alpha1.KomputerSquadStatus)) error {
	original := squad.DeepCopy()
	mutate(&squad.Status)
	return r.Status().Patch(ctx, squad, client.MergeFrom(original))
}

// SetupWithManager sets up the squad controller with the Manager.
func (r *KomputerSquadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&komputerv1alpha1.KomputerSquad{}).
		// Watch KomputerAgent changes to react to member phase/status transitions.
		Watches(
			&komputerv1alpha1.KomputerAgent{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				agent, ok := obj.(*komputerv1alpha1.KomputerAgent)
				if !ok {
					return nil
				}
				// Find any squad that lists this agent as a member.
				squadList := &komputerv1alpha1.KomputerSquadList{}
				if err := mgr.GetClient().List(ctx, squadList, client.InNamespace(agent.Namespace)); err != nil {
					return nil
				}
				var reqs []reconcile.Request
				for _, sq := range squadList.Items {
					for _, m := range sq.Spec.Members {
						if m.Ref == nil {
							continue
						}
						refNs := m.Ref.Namespace
						if refNs == "" {
							refNs = sq.Namespace
						}
						if m.Ref.Name == agent.Name && refNs == agent.Namespace {
							reqs = append(reqs, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Name:      sq.Name,
									Namespace: sq.Namespace,
								},
							})
							break
						}
					}
				}
				return reqs
			}),
		).
		Named("komputersquad").
		Complete(r)
}
