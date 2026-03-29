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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// KomputerScheduleReconciler reconciles a KomputerSchedule object
type KomputerScheduleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputerschedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputerschedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputerschedules/finalizers,verbs=update

// Reconcile moves the cluster state toward the desired state for a KomputerSchedule.
func (r *KomputerScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch the KomputerSchedule CR
	schedule := &komputerv1alpha1.KomputerSchedule{}
	if err := r.Get(ctx, req.NamespacedName, schedule); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. If suspended or being deleted, set phase Suspended, clear nextRunTime
	if schedule.Spec.Suspended || !schedule.DeletionTimestamp.IsZero() {
		schedule.Status.Phase = komputerv1alpha1.SchedulePhaseSuspended
		schedule.Status.NextRunTime = nil
		schedule.Status.Message = "Schedule is suspended"
		if err := r.Status().Update(ctx, schedule); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// 3. If lastRunStatus is InProgress, check if the agent finished
	if schedule.Status.LastRunStatus == "InProgress" {
		return r.reconcileAgentCompletion(ctx, schedule)
	}

	// 4. Parse the cron expression
	cronParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := cronParser.Parse(schedule.Spec.Schedule); err != nil {
		schedule.Status.Phase = komputerv1alpha1.SchedulePhaseError
		schedule.Status.Message = fmt.Sprintf("Invalid cron expression: %v", err)
		if statusErr := r.Status().Update(ctx, schedule); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}

	// 5. Compute and store nextRunTime if not set; set phase to Active; set agentName
	agentName := schedule.Spec.AgentName
	if agentName == "" {
		agentName = schedule.Name + "-agent"
	}
	schedule.Status.AgentName = agentName

	if schedule.Status.NextRunTime == nil {
		// Determine the "after" time: use lastRunTime if available, otherwise now
		after := time.Now().UTC()
		if schedule.Status.LastRunTime != nil {
			after = schedule.Status.LastRunTime.Time
		}
		nextTime, err := computeNextRunTime(schedule.Spec.Schedule, schedule.Spec.Timezone, after)
		if err != nil {
			schedule.Status.Phase = komputerv1alpha1.SchedulePhaseError
			schedule.Status.Message = fmt.Sprintf("Failed to compute next run time: %v", err)
			if statusErr := r.Status().Update(ctx, schedule); statusErr != nil {
				return ctrl.Result{}, statusErr
			}
			return ctrl.Result{}, nil
		}
		schedule.Status.NextRunTime = nextTime
	}

	schedule.Status.Phase = komputerv1alpha1.SchedulePhaseActive
	schedule.Status.Message = ""

	// 6. If now < nextRunTime, requeue after the delay
	now := time.Now().UTC()
	if now.Before(schedule.Status.NextRunTime.Time) {
		delay := schedule.Status.NextRunTime.Time.Sub(now)
		if err := r.Status().Update(ctx, schedule); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Waiting for next run time", "nextRunTime", schedule.Status.NextRunTime.Time, "delay", delay)
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	// 7. FIRE - trigger the agent
	agent := &komputerv1alpha1.KomputerAgent{}
	agentKey := client.ObjectKey{Name: agentName, Namespace: schedule.Namespace}
	agentErr := r.Get(ctx, agentKey, agent)

	if errors.IsNotFound(agentErr) {
		if schedule.Spec.Agent != nil {
			// Agent doesn't exist + spec.Agent is set: create from template
			agent = &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      agentName,
					Namespace: schedule.Namespace,
					Labels: map[string]string{
						"komputer.ai/schedule": schedule.Name,
					},
				},
				Spec: komputerv1alpha1.KomputerAgentSpec{
					Instructions: schedule.Spec.Instructions,
					Model:        schedule.Spec.Agent.Model,
					Lifecycle:    schedule.Spec.Agent.Lifecycle,
					Role:         schedule.Spec.Agent.Role,
					TemplateRef:  schedule.Spec.Agent.TemplateRef,
					Secrets:      schedule.Spec.Agent.Secrets,
				},
			}
			// Set ownerReference to the schedule
			if err := ctrl.SetControllerReference(schedule, agent, r.Scheme); err != nil {
				log.Error(err, "Failed to set owner reference on agent")
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, agent); err != nil {
				log.Error(err, "Failed to create agent", "agent", agentName)
				return ctrl.Result{}, err
			}
			log.Info("Created agent from schedule template", "agent", agentName)
		} else {
			// Agent doesn't exist + spec.AgentName set: error
			schedule.Status.Phase = komputerv1alpha1.SchedulePhaseError
			schedule.Status.Message = fmt.Sprintf("Referenced agent %q not found", agentName)
			if statusErr := r.Status().Update(ctx, schedule); statusErr != nil {
				return ctrl.Result{}, statusErr
			}
			return ctrl.Result{}, nil
		}
	} else if agentErr != nil {
		return ctrl.Result{}, agentErr
	} else {
		// Agent exists
		if agent.Status.TaskStatus == komputerv1alpha1.AgentTaskInProgress {
			// Agent is busy, skip this run
			log.Info("Agent is busy, skipping scheduled run", "agent", agentName)
			// Still compute next run time so we don't get stuck
			if !schedule.Spec.AutoDelete {
				nextTime, err := computeNextRunTime(schedule.Spec.Schedule, schedule.Spec.Timezone, now)
				if err != nil {
					return ctrl.Result{}, err
				}
				schedule.Status.NextRunTime = nextTime
				schedule.Status.Message = "Skipped: agent was busy"
			}
			if err := r.Status().Update(ctx, schedule); err != nil {
				return ctrl.Result{}, err
			}
			if schedule.Status.NextRunTime != nil {
				return ctrl.Result{RequeueAfter: schedule.Status.NextRunTime.Time.Sub(now)}, nil
			}
			return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}

		if agent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
			// Sleeping agent: patch instructions and wake it (creates new pod)
			original := agent.DeepCopy()
			agent.Spec.Instructions = schedule.Spec.Instructions
			if err := r.Patch(ctx, agent, client.MergeFrom(original)); err != nil {
				log.Error(err, "Failed to patch agent instructions", "agent", agentName)
				return ctrl.Result{}, err
			}
			if err := r.Get(ctx, types.NamespacedName{Name: agentName, Namespace: schedule.Namespace}, agent); err != nil {
				return ctrl.Result{}, err
			}
			agent.Status.Phase = komputerv1alpha1.AgentPhasePending
			agent.Status.TaskStatus = ""
			agent.Status.LastTaskMessage = ""
			if err := r.Status().Update(ctx, agent); err != nil {
				log.Error(err, "Failed to wake sleeping agent", "agent", agentName)
				return ctrl.Result{}, err
			}
			log.Info("Woke sleeping agent for scheduled run", "agent", agentName)
		} else if agent.Status.PodName != "" {
			// Running + idle agent: forward task to the pod.
			// Try direct HTTP to pod IP first (works in-cluster), fall back to API (works locally).
			bodyJSON, _ := json.Marshal(map[string]string{
				"instructions": schedule.Spec.Instructions,
			})
			forwarded := false

			// Try direct pod IP
			pod := &corev1.Pod{}
			if err := r.Get(ctx, types.NamespacedName{Name: agent.Status.PodName, Namespace: agent.Namespace}, pod); err == nil && pod.Status.PodIP != "" {
				podURL := fmt.Sprintf("http://%s:8000/task", pod.Status.PodIP)
				httpClient := &http.Client{Timeout: 5 * time.Second}
				resp, err := httpClient.Post(podURL, "application/json", bytes.NewReader(bodyJSON))
				if err == nil {
					resp.Body.Close()
					if resp.StatusCode == http.StatusConflict {
						log.Info("Agent pod is busy, skipping scheduled run", "agent", agentName)
						return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
					}
					forwarded = true
					log.Info("Forwarded scheduled task to pod directly", "agent", agentName)
				}
			}

			// Fallback: forward via komputer-api
			if !forwarded {
				apiURL, err := r.getAPIURL(ctx)
				if err != nil {
					log.Error(err, "Failed to get API URL for task forwarding")
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}
				apiBody, _ := json.Marshal(map[string]interface{}{
					"name":         agentName,
					"instructions": schedule.Spec.Instructions,
					"namespace":    schedule.Namespace,
				})
				taskURL := fmt.Sprintf("%s/api/v1/agents", apiURL)
				httpClient := &http.Client{Timeout: 30 * time.Second}
				resp, err := httpClient.Post(taskURL, "application/json", bytes.NewReader(apiBody))
				if err != nil {
					log.Error(err, "Failed to forward task via API", "agent", agentName)
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}
				resp.Body.Close()
				if resp.StatusCode == http.StatusConflict {
					log.Info("Agent is busy, skipping scheduled run", "agent", agentName)
					return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
				}
				log.Info("Forwarded scheduled task to agent via API fallback", "agent", agentName)
			}
		} else {
			// No pod, not sleeping — agent in weird state, retry
			log.Info("Agent has no pod and is not sleeping, retrying", "agent", agentName, "phase", agent.Status.Phase)
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	// 8. Update status: lastRunTime, runCount++, lastRunStatus="InProgress"
	nowMeta := metav1.NewTime(now)
	schedule.Status.LastRunTime = &nowMeta
	schedule.Status.RunCount++
	schedule.Status.LastRunStatus = "InProgress"

	// 9. If autoDelete, don't compute next
	if schedule.Spec.AutoDelete {
		schedule.Status.NextRunTime = nil
		schedule.Status.Message = "One-time schedule: will auto-delete after completion"
	} else {
		// 10. Compute next run time
		nextTime, err := computeNextRunTime(schedule.Spec.Schedule, schedule.Spec.Timezone, now)
		if err != nil {
			return ctrl.Result{}, err
		}
		schedule.Status.NextRunTime = nextTime
		schedule.Status.Message = ""
	}

	if err := r.Status().Update(ctx, schedule); err != nil {
		return ctrl.Result{}, err
	}

	// 11. Requeue after 15s to check agent completion
	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

// reconcileAgentCompletion checks if the triggered agent has finished its task.
func (r *KomputerScheduleReconciler) reconcileAgentCompletion(ctx context.Context, schedule *komputerv1alpha1.KomputerSchedule) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	agent := &komputerv1alpha1.KomputerAgent{}
	agentKey := client.ObjectKey{Name: schedule.Status.AgentName, Namespace: schedule.Namespace}
	agentErr := r.Get(ctx, agentKey, agent)

	if errors.IsNotFound(agentErr) {
		// Agent not found (AutoDelete agent deleted itself): count as success
		log.Info("Agent not found (likely auto-deleted), counting as success", "agent", schedule.Status.AgentName)
		schedule.Status.SuccessfulRuns++
		schedule.Status.LastRunStatus = "Success"
		schedule.Status.Message = "Agent completed and was auto-deleted"
		return r.handlePostCompletion(ctx, schedule)
	}
	if agentErr != nil {
		return ctrl.Result{}, agentErr
	}

	switch agent.Status.TaskStatus {
	case komputerv1alpha1.AgentTaskComplete:
		schedule.Status.SuccessfulRuns++
		schedule.Status.LastRunStatus = "Success"
		schedule.Status.Message = ""
		// Update costs
		if agent.Status.LastTaskCostUSD != "" {
			schedule.Status.LastRunCostUSD = agent.Status.LastTaskCostUSD
			// Accumulate totalCostUSD
			var currentTotal float64
			if schedule.Status.TotalCostUSD != "" {
				currentTotal, _ = strconv.ParseFloat(schedule.Status.TotalCostUSD, 64)
			}
			if lastCost, err := strconv.ParseFloat(agent.Status.LastTaskCostUSD, 64); err == nil {
				currentTotal += lastCost
				schedule.Status.TotalCostUSD = fmt.Sprintf("%.4f", currentTotal)
			}
		}
		return r.handlePostCompletion(ctx, schedule)

	case komputerv1alpha1.AgentTaskError:
		schedule.Status.FailedRuns++
		schedule.Status.LastRunStatus = "Failed"
		schedule.Status.Message = fmt.Sprintf("Agent error: %s", agent.Status.LastTaskMessage)
		return r.handlePostCompletion(ctx, schedule)

	default:
		// Still in progress, requeue
		if err := r.Status().Update(ctx, schedule); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}
}

// handlePostCompletion handles autoDelete logic and computes next run time after completion.
func (r *KomputerScheduleReconciler) handlePostCompletion(ctx context.Context, schedule *komputerv1alpha1.KomputerSchedule) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if schedule.Spec.AutoDelete {
		// Handle keepAgents: remove ownerReference so agent survives schedule deletion
		if schedule.Spec.KeepAgents {
			agent := &komputerv1alpha1.KomputerAgent{}
			agentKey := client.ObjectKey{Name: schedule.Status.AgentName, Namespace: schedule.Namespace}
			if err := r.Get(ctx, agentKey, agent); err == nil {
				// Filter out ownerReferences pointing to this schedule
				var filtered []metav1.OwnerReference
				for _, ref := range agent.OwnerReferences {
					if ref.UID != schedule.UID {
						filtered = append(filtered, ref)
					}
				}
				if len(filtered) != len(agent.OwnerReferences) {
					agent.OwnerReferences = filtered
					if err := r.Update(ctx, agent); err != nil {
						log.Error(err, "Failed to remove ownerReference from agent", "agent", agent.Name)
						return ctrl.Result{}, err
					}
					log.Info("Removed schedule ownerReference from agent", "agent", agent.Name)
				}
			} else if !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}

		// Delete the schedule CR
		log.Info("Auto-deleting schedule after completion", "schedule", schedule.Name)
		if err := r.Delete(ctx, schedule); err != nil && !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Compute next run time
	now := time.Now().UTC()
	nextTime, err := computeNextRunTime(schedule.Spec.Schedule, schedule.Spec.Timezone, now)
	if err != nil {
		schedule.Status.Phase = komputerv1alpha1.SchedulePhaseError
		schedule.Status.Message = fmt.Sprintf("Failed to compute next run time: %v", err)
		if statusErr := r.Status().Update(ctx, schedule); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}
	schedule.Status.NextRunTime = nextTime

	if err := r.Status().Update(ctx, schedule); err != nil {
		return ctrl.Result{}, err
	}

	delay := nextTime.Time.Sub(now)
	return ctrl.Result{RequeueAfter: delay}, nil
}

// getAPIURL returns the API URL. Checks KOMPUTER_API_URL env var first (for local dev),
// then falls back to KomputerConfig (for in-cluster).
func (r *KomputerScheduleReconciler) getAPIURL(ctx context.Context) (string, error) {
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

// computeNextRunTime parses a cron expression in the given timezone and returns the next fire time in UTC.
func computeNextRunTime(cronExpr, timezone string, after time.Time) (*metav1.Time, error) {
	cronParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := cronParser.Parse(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}

	loc := time.UTC
	if timezone != "" {
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %q: %w", timezone, err)
		}
	}

	// Convert "after" to the target timezone, compute next, then convert back to UTC
	afterInTZ := after.In(loc)
	next := sched.Next(afterInTZ)
	nextUTC := next.UTC()

	t := metav1.NewTime(nextUTC)
	return &t, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KomputerScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&komputerv1alpha1.KomputerSchedule{}).
		Owns(&komputerv1alpha1.KomputerAgent{}).
		Complete(r)
}
