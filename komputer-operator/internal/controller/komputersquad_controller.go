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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

const defaultOrphanTTL = 10 * time.Minute

// squadCleanupFinalizer blocks K8s from removing a KomputerSquad until the
// reconciler has cleared Status.Squad on every member agent. Without it, a
// squad delete that races the operator (e.g. operator down) leaves members
// stuck with Status.Squad=true and no controller reconciling them.
const squadCleanupFinalizer = "komputer.ai/squad-cleanup"

// KomputerSquadReconciler reconciles a KomputerSquad object.
type KomputerSquadReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputersquads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputersquads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputersquads/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods/ephemeralcontainers,verbs=get;patch;update

// Reconcile moves the cluster state toward the desired state for a KomputerSquad.
func (r *KomputerSquadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("squad", req.NamespacedName)

	// 1. Fetch the KomputerSquad CR
	squad := &komputerv1alpha1.KomputerSquad{}
	if err := r.Get(ctx, req.NamespacedName, squad); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Handle deletion: clear Status.Squad on all member agents before the squad is gone.
	// We use a finalizer so this runs even if the operator was down at delete time —
	// K8s blocks the actual deletion until we remove the finalizer.
	if !squad.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(squad, squadCleanupFinalizer) {
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
					return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
				}
			}
			controllerutil.RemoveFinalizer(squad, squadCleanupFinalizer)
			if err := r.Update(ctx, squad); err != nil {
				return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure the cleanup finalizer is set on every live squad.
	if !controllerutil.ContainsFinalizer(squad, squadCleanupFinalizer) {
		controllerutil.AddFinalizer(squad, squadCleanupFinalizer)
		if err := r.Update(ctx, squad); err != nil {
			return ctrl.Result{}, fmt.Errorf("add finalizer: %w", err)
		}
		// Requeue with the patched object on the next loop.
		return ctrl.Result{Requeue: true}, nil
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

		// Use the user-supplied name when set, otherwise fall back to <squad-name>-member-<index>
		agentName := member.Name
		if agentName == "" {
			agentName = fmt.Sprintf("%s-member-%d", squad.Name, i)
		}

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
// resolved KomputerAgent objects. Members whose KomputerAgent CR no longer exists
// are pruned from squad.Spec.Members so the squad doesn't loop forever trying to
// reconcile a phantom member.
func (r *KomputerSquadReconciler) markMembersAsSquad(ctx context.Context, squad *komputerv1alpha1.KomputerSquad) ([]*komputerv1alpha1.KomputerAgent, error) {
	log := logf.FromContext(ctx)
	agents := make([]*komputerv1alpha1.KomputerAgent, 0, len(squad.Spec.Members))
	prunedMembers := make([]komputerv1alpha1.KomputerSquadMember, 0, len(squad.Spec.Members))
	pruned := false

	// Assign per-member ports stably by member index. buildSquadPodSpec uses the
	// same scheme (8000 + position-in-agents-slice), and since we append agents
	// in member order, the indices match.
	for memberIdx, member := range squad.Spec.Members {
		if member.Ref == nil {
			prunedMembers = append(prunedMembers, member)
			continue
		}
		ns := member.Ref.Namespace
		if ns == "" {
			ns = squad.Namespace
		}

		agent := &komputerv1alpha1.KomputerAgent{}
		if err := r.Get(ctx, types.NamespacedName{Name: member.Ref.Name, Namespace: ns}, agent); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("Member agent no longer exists; removing from squad spec", "agent", member.Ref.Name)
				pruned = true
				continue
			}
			return nil, fmt.Errorf("get agent %s: %w", member.Ref.Name, err)
		}

		desiredPort := int32(8000 + memberIdx)

		// Mark agent as squad-managed. The actual Phase is set later by
		// updateSquadMemberStatus based on the squad pod's state.
		if !agent.Status.Squad || agent.Status.Port != desiredPort {
			original := agent.DeepCopy()
			agent.Status.Squad = true
			agent.Status.Port = desiredPort
			agent.Status.Message = fmt.Sprintf("Managed by squad %s", squad.Name)
			if err := r.Status().Patch(ctx, agent, client.MergeFrom(original)); err != nil {
				return nil, fmt.Errorf("patch agent %s squad=true: %w", agent.Name, err)
			}
		}

		agents = append(agents, agent)
		prunedMembers = append(prunedMembers, member)
	}

	if pruned {
		original := squad.DeepCopy()
		squad.Spec.Members = prunedMembers
		if err := r.Patch(ctx, squad, client.MergeFrom(original)); err != nil {
			return nil, fmt.Errorf("prune missing members from squad %s: %w", squad.Name, err)
		}
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
				Labels: mergeLabels(agent.Spec.Labels, map[string]string{
					"komputer.ai/agent-name": agent.Name,
					"komputer.ai/squad":      squad.Name,
				}),
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
//
// Membership change strategy:
//   - If the Pod does not exist → create it.
//   - If the Pod exists but is NOT Running → delete + recreate (fallback path).
//   - If the Pod is Running:
//   - Added members → inject ephemeral containers via the pods/ephemeralcontainers subresource.
//   - Removed members → cancel their in-flight task via the API; the container
//     remains in the pod until the next restart (k8s cannot remove containers
//     from a running pod without killing it).
//
// NOTE — ephemeral container volume limitation:
// Kubernetes does not allow adding volumes to a running pod. This means a newly
// injected ephemeral container can only mount volumes that were declared when the
// pod was first created (i.e. the original members' PVC-backed volumes). The new
// member's own PVC volume ("<agentName>-workspace") is NOT in the pod's volumes
// list and therefore CANNOT be mounted by the ephemeral container. As a result:
//
//   - The new member will start without /workspace (its own persistent workspace).
//   - It WILL be able to read all existing siblings' workspaces at
//     /agents/<sibling>/workspace.
//
// This is an acceptable v1 trade-off: the agent is functional and collaborative,
// but loses its own workspace mount until the squad pod is next restarted (which
// will include its PVC in the volumes block).
func (r *KomputerSquadReconciler) reconcileSquadPod(ctx context.Context, squad *komputerv1alpha1.KomputerSquad, agents []*komputerv1alpha1.KomputerAgent, config *komputerv1alpha1.KomputerConfig) error {
	log := logf.FromContext(ctx)
	podName := squad.Name + "-pod"
	ns := squad.Namespace

	existing := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: ns}, existing)

	// If every member is Sleeping, the squad has nothing to do — delete the pod
	// (PVCs are kept). The pod is rebuilt when any member wakes (the API clears
	// that member's Phase=Sleeping before forwarding a task).
	allSleeping := len(agents) > 0
	for _, a := range agents {
		if a.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
			allSleeping = false
			break
		}
	}
	if allSleeping {
		// If a break-up was requested, dissolve the squad now: deleting the squad
		// CR triggers the cleanup finalizer which clears Status.Squad on every
		// member, returning them to the agent controller as solo agents. PVCs
		// survive (they're owned by the agent CR, not the squad).
		if squad.Spec.BreakUpRequested {
			log.Info("All members sleeping and break-up requested — deleting squad CR", "squad", squad.Name)
			if delErr := r.Delete(ctx, squad); delErr != nil && !apierrors.IsNotFound(delErr) {
				return fmt.Errorf("delete squad %s (break-up): %w", squad.Name, delErr)
			}
			return nil
		}
		if err == nil {
			log.Info("All squad members are Sleeping — deleting squad pod", "pod", podName)
			if delErr := r.Delete(ctx, existing); delErr != nil && !apierrors.IsNotFound(delErr) {
				return fmt.Errorf("delete squad pod %s (all sleeping): %w", podName, delErr)
			}
		}
		// Pod already gone (or just deleted) — nothing more to do until someone wakes.
		return nil
	}

	if err == nil {
		// Pod exists — check if membership changed by comparing container names
		added, removed := membershipDelta(existing, agents)
		if len(added) == 0 && len(removed) == 0 {
			return nil // nothing to do
		}

		// If the pod is not yet Running, fall back to delete + recreate so the new
		// pod is built with all members' PVCs in its volumes block from the start.
		if existing.Status.Phase != corev1.PodRunning {
			log.Info("Squad membership changed (pod not Running), deleting pod for recreation", "pod", podName)
			if delErr := r.Delete(ctx, existing); delErr != nil && !apierrors.IsNotFound(delErr) {
				return fmt.Errorf("delete stale squad pod %s: %w", podName, delErr)
			}
			// Will be created on the next reconcile loop once the pod is gone.
			return nil
		}

		// Pod is Running — use surgical changes to avoid interrupting existing members.

		// Inject ephemeral containers for newly added members.
		for _, agent := range added {
			log.Info("Injecting ephemeral container for new squad member", "pod", podName, "agent", agent.Name)
			if injectErr := r.injectEphemeralContainer(ctx, existing, agent, agents, config); injectErr != nil {
				log.Error(injectErr, "Failed to inject ephemeral container", "agent", agent.Name)
				// Non-fatal: log and continue; controller will retry on next reconcile.
			}
		}

		// Cancel tasks for removed members (container stays until next pod restart).
		for _, agent := range removed {
			log.Info("Cancelling task for removed squad member", "pod", podName, "agent", agent.Name)
			if cancelErr := r.cancelTaskViaAPI(ctx, ns, agent.Name); cancelErr != nil {
				log.Error(cancelErr, "Failed to cancel task for removed member", "agent", agent.Name)
				// Non-fatal: log and continue.
			}
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

// membershipDelta returns the agents that were added to (or removed from) the
// running pod relative to the desired agent set.
//
// "added"   = in desired agents, not in pod.Spec.Containers
// "removed" = in pod.Spec.Containers, not in desired agents
//
// Ephemeral containers that were previously injected are also checked so that
// containers injected in earlier reconciles are not re-injected.
func membershipDelta(pod *corev1.Pod, agents []*komputerv1alpha1.KomputerAgent) (added, removed []*komputerv1alpha1.KomputerAgent) {
	// Build the set of all container names currently in the pod (regular + ephemeral).
	podContainerNames := make(map[string]bool)
	for _, c := range pod.Spec.Containers {
		podContainerNames[c.Name] = true
	}
	for _, ec := range pod.Spec.EphemeralContainers {
		podContainerNames[ec.Name] = true
	}

	// Build the desired-agent set for O(1) lookups.
	desiredAgentNames := make(map[string]*komputerv1alpha1.KomputerAgent, len(agents))
	for _, a := range agents {
		desiredAgentNames[a.Name] = a
	}

	// Agents desired but not yet in pod.
	for _, a := range agents {
		if !podContainerNames[a.Name] {
			added = append(added, a)
		}
	}

	// Regular containers in pod that are no longer desired.
	// (We do not attempt to remove ephemeral containers — k8s does not support it.)
	for _, c := range pod.Spec.Containers {
		if _, ok := desiredAgentNames[c.Name]; !ok {
			removed = append(removed, &komputerv1alpha1.KomputerAgent{
				// Populate only Name/Namespace — enough for the cancel call.
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.Name,
					Namespace: pod.Namespace,
				},
			})
		}
	}

	return added, removed
}

// injectEphemeralContainer injects a new ephemeral container into a running squad
// pod for the given agent via the pods/ephemeralcontainers subresource.
//
// Volume limitation: ephemeral containers can only mount volumes that already
// exist in the pod's volumes list at pod creation time. The new agent's own PVC
// volume is not present, so /workspace (its own) cannot be mounted. It CAN mount
// all sibling workspaces at /agents/<sibling>/workspace because those volumes
// were declared when the pod was first created. See reconcileSquadPod for the
// full trade-off discussion.
func (r *KomputerSquadReconciler) injectEphemeralContainer(ctx context.Context, pod *corev1.Pod, agent *komputerv1alpha1.KomputerAgent, allAgents []*komputerv1alpha1.KomputerAgent, config *komputerv1alpha1.KomputerConfig) error {
	// Resolve the agent's image from its template.
	templateRef := agent.Spec.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}
	template := &komputerv1alpha1.KomputerAgentTemplate{}
	if err := r.Get(ctx, types.NamespacedName{Name: templateRef, Namespace: agent.Namespace}, template); err != nil {
		clusterTemplate := &komputerv1alpha1.KomputerAgentClusterTemplate{}
		if clusterErr := r.Get(ctx, types.NamespacedName{Name: templateRef}, clusterTemplate); clusterErr != nil {
			return fmt.Errorf("template %q not found for agent %s", templateRef, agent.Name)
		}
		template = &komputerv1alpha1.KomputerAgentTemplate{
			Spec: *clusterTemplate.Spec.DeepCopy(),
		}
	}
	if len(template.Spec.PodSpec.Containers) == 0 {
		return fmt.Errorf("template %q for agent %s has no containers defined", templateRef, agent.Name)
	}
	baseContainer := template.Spec.PodSpec.Containers[0]

	// Build env vars the same way as a regular squad container.
	envVars, err := buildAgentEnvVars(ctx, r.Client, agent, config)
	if err != nil {
		return fmt.Errorf("build env vars for ephemeral container %s: %w", agent.Name, err)
	}
	// AGENT_PORT must be the per-member port assigned by markMembersAsSquad —
	// otherwise the ephemeral container defaults to 8000 and collides with the
	// first regular squad member that already binds 8000.
	agentPort := agent.Status.Port
	if agentPort == 0 {
		agentPort = 8000
	}
	envVars = append(envVars, corev1.EnvVar{Name: "AGENT_PORT", Value: fmt.Sprintf("%d", agentPort)})
	// If this member is currently Sleeping, start the EC in wake-idle mode so it
	// exposes its HTTP server without auto-running its previous instructions.
	if agent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		envVars = append(envVars, corev1.EnvVar{Name: "KOMPUTER_WAKE_IDLE", Value: "true"})
	}
	envVars = mergeEnvVars(baseContainer.Env, envVars)

	// Build volume mounts: only sibling workspaces (own PVC volume is not in the
	// pod's volumes list — see volume limitation note above).
	podVolumeNames := make(map[string]bool, len(pod.Spec.Volumes))
	for _, v := range pod.Spec.Volumes {
		podVolumeNames[v.Name] = true
	}

	var volumeMounts []corev1.VolumeMount
	for _, sibling := range allAgents {
		if sibling.Name == agent.Name {
			continue
		}
		volumeName := sibling.Name + "-workspace"
		if podVolumeNames[volumeName] {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      volumeName,
				MountPath: "/agents/" + sibling.Name + "/workspace",
				ReadOnly:  false,
			})
		}
	}

	ec := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:            agent.Name,
			Image:           baseContainer.Image,
			ImagePullPolicy: baseContainer.ImagePullPolicy,
			Env:             envVars,
			VolumeMounts:    volumeMounts,
			// Resources intentionally omitted: k8s rejects ResourceRequirements
			// on ephemeral containers (they cannot be used for resource accounting).
		},
	}

	// Use a strategic merge patch — the apiserver merges by container name and
	// handles the case where /spec/ephemeralContainers does not yet exist on the
	// pod (a JSONPatch `add /spec/ephemeralContainers/-` fails with a bare 422
	// when the field is absent).
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"ephemeralContainers": []corev1.EphemeralContainer{ec},
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal ephemeral container patch: %w", err)
	}

	if err := r.SubResource("ephemeralcontainers").Patch(ctx, pod, client.RawPatch(types.StrategicMergePatchType, patchBytes)); err != nil {
		// Surface the full K8s status (controller-runtime trims it to a generic message).
		if statusErr, ok := err.(*apierrors.StatusError); ok {
			return fmt.Errorf("inject ephemeral container %q: %s (reason=%s, code=%d, details=%v)",
				agent.Name, statusErr.ErrStatus.Message, statusErr.ErrStatus.Reason, statusErr.ErrStatus.Code, statusErr.ErrStatus.Details)
		}
		return fmt.Errorf("inject ephemeral container %q: %w", agent.Name, err)
	}

	// Patch the per-member pod label so the per-agent Service
	// (selector: komputer.ai/member-<name>=true) starts routing to this container.
	// Without this, the EC is reachable on the pod IP but the Service has no endpoints.
	memberLabel := "komputer.ai/member-" + agent.Name
	if pod.Labels[memberLabel] != "true" {
		labelPatch := map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": map[string]string{memberLabel: "true"},
			},
		}
		labelBytes, _ := json.Marshal(labelPatch)
		if err := r.Patch(ctx, pod, client.RawPatch(types.StrategicMergePatchType, labelBytes)); err != nil {
			return fmt.Errorf("patch pod label %s: %w", memberLabel, err)
		}
	}
	return nil
}

// cancelTaskViaAPI calls POST /api/v1/agents/<name>/cancel on the running komputer-api.
// This is used when a member is removed from a running squad: the container cannot
// be removed from the pod without a restart, but the agent's in-flight task can be
// cancelled immediately.
func (r *KomputerSquadReconciler) cancelTaskViaAPI(ctx context.Context, namespace, agentName string) error {
	apiURL, err := r.getAPIURL(ctx)
	if err != nil {
		return err
	}
	cancelURL := fmt.Sprintf("%s/api/v1/agents/%s/cancel", apiURL, agentName)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cancelURL, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cancel returned %d", resp.StatusCode)
	}
	return nil
}

// getAPIURL returns the API URL. Checks KOMPUTER_API_URL env var first (for local dev),
// then falls back to KomputerConfig (for in-cluster).
// Copied from KomputerScheduleReconciler.getAPIURL — intentionally duplicated to
// keep blast radius minimal; a shared helper can be extracted in a future cleanup.
func (r *KomputerSquadReconciler) getAPIURL(ctx context.Context) (string, error) {
	if envURL := os.Getenv("KOMPUTER_API_URL"); envURL != "" {
		return envURL, nil
	}
	configList := &komputerv1alpha1.KomputerConfigList{}
	if err := r.List(ctx, configList); err != nil {
		return "", err
	}
	if len(configList.Items) == 0 {
		return "", fmt.Errorf("no KomputerConfig found")
	}
	url := configList.Items[0].Spec.APIURL
	if url == "" {
		return "", fmt.Errorf("KomputerConfig has no apiURL")
	}
	return url, nil
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
		// Each member's port is stamped on Status.Port by markMembersAsSquad.
		// We trust that here so the per-agent Service (which targets Status.Port)
		// stays in sync with the actual container port.
		agentPort := agent.Status.Port
		if agentPort == 0 {
			agentPort = 8000
		}
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
		// AGENT_PORT tells the agent which port to bind. Squad members each get a
		// unique port so they don't collide on the shared pod network namespace.
		envVars = append(envVars, corev1.EnvVar{Name: "AGENT_PORT", Value: fmt.Sprintf("%d", agentPort)})
		// If this member is currently Sleeping but the pod is being (re)built for a
		// non-sleeping sibling, start the container in wake-idle mode so it exposes
		// its HTTP server without auto-running its previous instructions.
		if agent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
			envVars = append(envVars, corev1.EnvVar{Name: "KOMPUTER_WAKE_IDLE", Value: "true"})
		}
		// Inject ANTHROPIC_API_KEY from the template's AnthropicKeySecretRef.
		// The mirror exists in the squad pod's namespace, so the secretKeyRef
		// resolves locally. Mirrors are reconciled the same way solo agents do.
		envVars = append(envVars, corev1.EnvVar{
			Name: "ANTHROPIC_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: template.Spec.AnthropicKeySecretRef.Name,
					},
					Key: template.Spec.AnthropicKeySecretRef.Key,
				},
			},
		})
		// Strip any user-supplied ANTHROPIC_API_KEY from the template before
		// merging so the operator-injected entry above always wins.
		c.Env = stripEnvVar(c.Env, "ANTHROPIC_API_KEY", logf.FromContext(ctx))
		c.Env = mergeEnvVars(c.Env, envVars)

		// Declare the per-member containerPort. Name must be unique across the
		// squad pod's containers (K8s pod-spec validation), so we suffix it with
		// the port number. The per-agent Service routes to this port numerically.
		c.Ports = []corev1.ContainerPort{{
			Name:          fmt.Sprintf("http-%d", agentPort),
			ContainerPort: agentPort,
			Protocol:      corev1.ProtocolTCP,
		}}

		// Health probes hit the per-member port.
		c.LivenessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(int(agentPort)),
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       30,
		}
		c.ReadinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intstr.FromInt(int(agentPort)),
				},
			},
			// /readyz is a no-op endpoint; poll fast so wake-up latency to
			// "Running" stays under a couple of seconds.
			InitialDelaySeconds: 1,
			PeriodSeconds:       2,
			FailureThreshold:    3,
		}
		c.Lifecycle = &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/shutdown",
					Port: intstr.FromInt(int(agentPort)),
				},
			},
		}

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

	// Pod labels include one "komputer.ai/member-<name>=true" label per squad
	// member so the per-agent Service (selector: komputer.ai/member-<name>=true)
	// can find the pod regardless of which container runs the agent.
	// Build a union of user-supplied labels from all member agents (alphabetical
	// sort for determinism when keys conflict).
	sortedAgents := make([]*komputerv1alpha1.KomputerAgent, len(agents))
	copy(sortedAgents, agents)
	sort.Slice(sortedAgents, func(i, j int) bool { return sortedAgents[i].Name < sortedAgents[j].Name })
	unionUserLabels := map[string]string{}
	for _, m := range sortedAgents {
		for k, v := range m.Spec.Labels {
			unionUserLabels[k] = v
		}
	}
	systemLabels := map[string]string{
		"komputer.ai/squad":      "true",
		"komputer.ai/squad-name": squad.Name,
	}
	for _, agent := range agents {
		systemLabels["komputer.ai/member-"+agent.Name] = "true"
	}
	podLabels := mergeLabels(unionUserLabels, systemLabels)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: squad.Namespace,
			Labels:    podLabels,
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
			// Ephemeral containers (members adopted into a running squad pod) cannot
			// have readiness probes — K8s rejects them — so cs.Ready is always false.
			// Treat ephemeral container as ready when its state is "running".
			if !ready {
				for _, cs := range pod.Status.EphemeralContainerStatuses {
					if cs.Name == agent.Name && cs.State.Running != nil {
						ready = true
						break
					}
				}
			}
		}
		memberStatuses = append(memberStatuses, komputerv1alpha1.KomputerSquadMemberStatus{
			Name:       agent.Name,
			Ready:      ready,
			TaskStatus: string(agent.Status.TaskStatus),
		})

		// Mirror the squad pod state onto the member agent's Status so the UI
		// can show "Running" instead of "Squad".
		// Exception: an explicit Sleeping intent (set by the API sleep handler)
		// is preserved regardless of pod state — the member's container may still
		// be running inside the squad pod, but the agent is logically asleep.
		desiredPhase := komputerv1alpha1.AgentPhasePending
		if agent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
			desiredPhase = komputerv1alpha1.AgentPhaseSleeping
		} else if podExists {
			switch pod.Status.Phase {
			case corev1.PodRunning:
				if ready {
					desiredPhase = komputerv1alpha1.AgentPhaseRunning
				}
			case corev1.PodFailed:
				desiredPhase = komputerv1alpha1.AgentPhaseFailed
			case corev1.PodSucceeded:
				desiredPhase = komputerv1alpha1.AgentPhaseSucceeded
			}
		}
		if agent.Status.Phase != desiredPhase || agent.Status.PodName != podName {
			original := agent.DeepCopy()
			agent.Status.Phase = desiredPhase
			agent.Status.PodName = podName
			if err := r.Status().Patch(ctx, agent, client.MergeFrom(original)); err != nil {
				return fmt.Errorf("patch agent %s phase from squad pod: %w", agent.Name, err)
			}
		}
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

// clearSquadPhase clears Status.Squad and resets Phase so the agent controller picks it up.
func (r *KomputerSquadReconciler) clearSquadPhase(ctx context.Context, agentName, namespace string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	if err := r.Get(ctx, types.NamespacedName{Name: agentName, Namespace: namespace}, agent); err != nil {
		if apierrors.IsNotFound(err) {
			return nil // agent already gone, nothing to clear
		}
		return err
	}
	if !agent.Status.Squad {
		return nil // already cleared
	}
	original := agent.DeepCopy()
	agent.Status.Squad = false
	agent.Status.Phase = ""
	agent.Status.PodName = ""
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
		// Watch the owned squad pod so container readiness changes (Pending → Running)
		// trigger an immediate reconcile instead of waiting up to 30s for the next tick.
		Owns(&corev1.Pod{}).
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
