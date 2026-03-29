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
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

const officeFinalizer = "komputer.ai/office-members"

// KomputerOfficeReconciler reconciles a KomputerOffice object
type KomputerOfficeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputeroffices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputeroffices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputeroffices/finalizers,verbs=update

// Reconcile moves the cluster state toward the desired state for a KomputerOffice.
func (r *KomputerOfficeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Get the KomputerOffice CR
	office := &komputerv1alpha1.KomputerOffice{}
	if err := r.Get(ctx, req.NamespacedName, office); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Handle finalizer
	if !office.DeletionTimestamp.IsZero() {
		// Office is being deleted — ensure all member agents are deleted
		if controllerutil.ContainsFinalizer(office, officeFinalizer) {
			agentList := &komputerv1alpha1.KomputerAgentList{}
			if err := r.List(ctx, agentList, client.InNamespace(office.Namespace), client.MatchingLabels{
				"komputer.ai/office": office.Name,
			}); err != nil {
				return ctrl.Result{}, err
			}

			// Delete all agents that haven't been marked for deletion yet
			for i := range agentList.Items {
				if agentList.Items[i].DeletionTimestamp.IsZero() {
					if err := r.Delete(ctx, &agentList.Items[i]); err != nil && !errors.IsNotFound(err) {
						log.Error(err, "Failed to delete member agent", "agent", agentList.Items[i].Name)
						return ctrl.Result{}, err
					}
				}
			}

			// Re-list to check if any agents still exist (including those being terminated)
			if err := r.List(ctx, agentList, client.InNamespace(office.Namespace), client.MatchingLabels{
				"komputer.ai/office": office.Name,
			}); err != nil {
				return ctrl.Result{}, err
			}
			if len(agentList.Items) > 0 {
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}

			// All agents gone — remove finalizer
			controllerutil.RemoveFinalizer(office, officeFinalizer)
			if err := r.Update(ctx, office); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure finalizer is present
	if !controllerutil.ContainsFinalizer(office, officeFinalizer) {
		controllerutil.AddFinalizer(office, officeFinalizer)
		if err := r.Update(ctx, office); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 3. List all agents with the office label
	agentList := &komputerv1alpha1.KomputerAgentList{}
	if err := r.List(ctx, agentList, client.InNamespace(office.Namespace), client.MatchingLabels{
		"komputer.ai/office": office.Name,
	}); err != nil {
		return ctrl.Result{}, err
	}

	// 4. Get the manager agent
	managerAgent := &komputerv1alpha1.KomputerAgent{}
	managerErr := r.Get(ctx, client.ObjectKey{Name: office.Spec.Manager, Namespace: office.Namespace}, managerAgent)
	if managerErr != nil && !errors.IsNotFound(managerErr) {
		return ctrl.Result{}, managerErr
	}

	// 5. Set ownerReference: manager -> Office (if not already set)
	if managerErr == nil {
		if !hasOwnerReference(managerAgent, office) {
			if err := ctrl.SetControllerReference(office, managerAgent, r.Scheme); err != nil {
				log.Error(err, "Failed to set owner reference on manager agent")
			} else {
				if err := r.Update(ctx, managerAgent); err != nil {
					log.Error(err, "Failed to update manager agent with owner reference")
					return ctrl.Result{}, err
				}
			}
		}
	}

	// 6. Set ownerReference: each worker -> Manager (if not already set)
	if managerErr == nil {
		for i := range agentList.Items {
			worker := &agentList.Items[i]
			if worker.Name == office.Spec.Manager {
				continue // skip the manager itself
			}
			if !hasOwnerReference(worker, managerAgent) {
				if err := ctrl.SetControllerReference(managerAgent, worker, r.Scheme); err != nil {
					log.Error(err, "Failed to set owner reference on worker agent", "worker", worker.Name)
					continue
				}
				if err := r.Update(ctx, worker); err != nil {
					log.Error(err, "Failed to update worker agent with owner reference", "worker", worker.Name)
					return ctrl.Result{}, err
				}
			}
		}
	}

	// 7. Build status: aggregate costs, count active/completed, determine phase.
	// Preserve last known status for members that were deleted (e.g. AutoDelete).
	liveAgents := make(map[string]*komputerv1alpha1.KomputerAgent, len(agentList.Items))
	for i := range agentList.Items {
		liveAgents[agentList.Items[i].Name] = &agentList.Items[i]
	}

	var (
		totalAgents     int
		activeAgents    int
		completedAgents int
		totalCost       float64
		hasInProgress   bool
		hasError        bool
		allComplete     = true
		members         []komputerv1alpha1.OfficeMemberStatus
	)

	// Helper to process an agent (live or preserved snapshot).
	processAgent := func(ms komputerv1alpha1.OfficeMemberStatus) {
		totalAgents++
		switch komputerv1alpha1.AgentTaskStatus(ms.TaskStatus) {
		case komputerv1alpha1.AgentTaskInProgress:
			activeAgents++
			hasInProgress = true
			allComplete = false
		case komputerv1alpha1.AgentTaskComplete:
			completedAgents++
		case komputerv1alpha1.AgentTaskError:
			hasError = true
			allComplete = false
		default:
			allComplete = false
		}
		if ms.LastTaskCostUSD != "" {
			if cost, err := strconv.ParseFloat(ms.LastTaskCostUSD, 64); err == nil {
				totalCost += cost
			}
		}
	}

	// Update manager status from live agent, or preserve last known.
	if a, ok := liveAgents[office.Spec.Manager]; ok {
		office.Status.Manager = komputerv1alpha1.OfficeMemberStatus{
			Name: a.Name, Role: a.Spec.Role,
			TaskStatus: string(a.Status.TaskStatus), LastTaskCostUSD: a.Status.TotalCostUSD,
		}
	}
	// Always count the manager if we have status for it.
	if office.Status.Manager.Name != "" {
		processAgent(office.Status.Manager)
	}

	// Build a set of known member names (from live agents + previous snapshot).
	knownMembers := make(map[string]komputerv1alpha1.OfficeMemberStatus)
	// Start with previous snapshot (preserves deleted members).
	for _, m := range office.Status.Members {
		// If a member was deleted (no longer live) and has no TaskStatus,
		// they were auto-deleted after completion — mark as Complete.
		if _, alive := liveAgents[m.Name]; !alive && m.TaskStatus == "" {
			m.TaskStatus = string(komputerv1alpha1.AgentTaskComplete)
		}
		knownMembers[m.Name] = m
	}
	// Update with live agents (overrides snapshot for agents still alive).
	for _, a := range agentList.Items {
		if a.Name == office.Spec.Manager {
			continue
		}
		knownMembers[a.Name] = komputerv1alpha1.OfficeMemberStatus{
			Name: a.Name, Role: a.Spec.Role,
			TaskStatus: string(a.Status.TaskStatus), LastTaskCostUSD: a.Status.TotalCostUSD,
		}
	}
	// Collect and count members.
	for _, ms := range knownMembers {
		members = append(members, ms)
		processAgent(ms)
	}

	// Determine phase
	var phase komputerv1alpha1.KomputerOfficePhase
	if hasError {
		phase = komputerv1alpha1.OfficePhaseError
	} else if hasInProgress {
		phase = komputerv1alpha1.OfficePhaseInProgress
	} else if allComplete && totalAgents > 0 {
		phase = komputerv1alpha1.OfficePhaseComplete
	} else if totalAgents > 0 {
		phase = komputerv1alpha1.OfficePhaseInProgress
	}

	// 8. Update status (preserve CreatedAt from initial creation)
	office.Status.Phase = phase
	office.Status.Members = members
	office.Status.TotalAgents = totalAgents
	office.Status.ActiveAgents = activeAgents
	office.Status.CompletedAgents = completedAgents
	office.Status.TotalCostUSD = fmt.Sprintf("%.4f", totalCost)
	if office.Status.CreatedAt == nil {
		now := metav1.Now()
		office.Status.CreatedAt = &now
	}

	if err := r.Status().Update(ctx, office); err != nil {
		log.Error(err, "Failed to update office status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// hasOwnerReference checks if the child object already has an ownerReference pointing to the owner.
func hasOwnerReference(child metav1.Object, owner metav1.Object) bool {
	for _, ref := range child.GetOwnerReferences() {
		if ref.UID == owner.GetUID() {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *KomputerOfficeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&komputerv1alpha1.KomputerOffice{}).
		Owns(&komputerv1alpha1.KomputerAgent{}).
		Watches(&komputerv1alpha1.KomputerAgent{}, handler.EnqueueRequestsFromMapFunc(
			func(ctx context.Context, obj client.Object) []ctrl.Request {
				// When any agent changes, if it has an office label, reconcile that office.
				labels := obj.GetLabels()
				officeName := labels["komputer.ai/office"]
				if officeName == "" {
					return nil
				}
				return []ctrl.Request{{
					NamespacedName: client.ObjectKey{
						Name:      officeName,
						Namespace: obj.GetNamespace(),
					},
				}}
			},
		)).
		Complete(r)
}
