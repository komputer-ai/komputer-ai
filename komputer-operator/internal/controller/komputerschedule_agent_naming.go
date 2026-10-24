package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// createRunAgent fires a per-run schedule: it creates a fresh agent from the
// template and points status.AgentName at it. The run is skipped (skipped=true)
// while the previous run's agent is still working.
func (r *KomputerScheduleReconciler) createRunAgent(ctx context.Context, base, schedule *komputerv1alpha1.KomputerSchedule, now time.Time) (ctrl.Result, bool, error) {
	log := logf.FromContext(ctx)

	if prev := schedule.Status.AgentName; prev != "" {
		agent := &komputerv1alpha1.KomputerAgent{}
		err := r.Get(ctx, client.ObjectKey{Name: prev, Namespace: schedule.Namespace}, agent)
		if err != nil && !errors.IsNotFound(err) {
			return ctrl.Result{}, false, err
		}
		if err == nil && taskInProgress(agent.Status.TaskStatus) {
			log.Info("Previous run's agent is busy, skipping scheduled run", "agent", prev)
			if !schedule.Spec.AutoDelete {
				nextTime, err := computeNextRunTime(schedule.Spec.Schedule, schedule.Spec.Timezone, now)
				if err != nil {
					return ctrl.Result{}, false, err
				}
				schedule.Status.NextRunTime = nextTime
				schedule.Status.Message = "Skipped: previous run still in progress"
			}
			if err := r.patchScheduleStatus(ctx, base, schedule); err != nil {
				return ctrl.Result{}, false, err
			}
			if schedule.Status.NextRunTime != nil {
				return ctrl.Result{RequeueAfter: schedule.Status.NextRunTime.Time.Sub(now)}, true, nil
			}
			return ctrl.Result{RequeueAfter: 15 * time.Second}, true, nil
		}
	}

	// Names derive from the scheduled run time, so a reconcile retried after the
	// agent was created (e.g. the status write conflicted) finds it instead of
	// starting the task twice.
	runTime := schedule.Status.NextRunTime.Time
	runKey := runTime.UTC().Format(time.RFC3339)
	seed := string(schedule.UID) + "/" + runKey
	name := komputerv1alpha1.RunAgentName(schedule.Name, schedule.Spec.Timezone, schedule.Spec.AgentNaming, runTime, komputerv1alpha1.NameSuffix(seed, 5))
	// The second name is for when another run took the first (e.g. a manual trigger in the same minute).
	for _, candidate := range []string{name, name + "-" + komputerv1alpha1.NameSuffix(seed+"/2", 4)} {
		err := r.createScheduledAgent(ctx, schedule, candidate, runKey)
		if err == nil {
			log.Info("Created agent for scheduled run", "agent", candidate)
			schedule.Status.AgentName = candidate
			return ctrl.Result{}, false, nil
		}
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "Failed to create agent for scheduled run", "agent", candidate)
			return ctrl.Result{}, false, err
		}
		existing := &komputerv1alpha1.KomputerAgent{}
		if err := r.Get(ctx, client.ObjectKey{Name: candidate, Namespace: schedule.Namespace}, existing); err != nil {
			return ctrl.Result{}, false, err // not in the cache yet; retry
		}
		if existing.Annotations[scheduleRunAnnotation] == runKey && metav1.IsControlledBy(existing, schedule) {
			schedule.Status.AgentName = candidate
			return ctrl.Result{}, false, nil
		}
	}
	return ctrl.Result{}, false, fmt.Errorf("agent names for run %s are taken by other agents", runKey)
}

// scheduleRunAnnotation marks a per-run agent with the scheduled run time it was created for.
const scheduleRunAnnotation = "komputer.ai/schedule-run"

func (r *KomputerScheduleReconciler) createScheduledAgent(ctx context.Context, schedule *komputerv1alpha1.KomputerSchedule, name, runKey string) error {
	agent := buildScheduledAgent(schedule, name)
	agent.Annotations = map[string]string{scheduleRunAnnotation: runKey}
	if err := ctrl.SetControllerReference(schedule, agent, r.Scheme); err != nil {
		return err
	}
	return r.Create(ctx, agent)
}
