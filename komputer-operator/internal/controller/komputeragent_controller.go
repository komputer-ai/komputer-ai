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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// KomputerAgentReconciler reconciles a KomputerAgent object
type KomputerAgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputeragents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputeragents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputeragents/finalizers,verbs=update
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputeragenttemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputeragentclustertemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputerconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile moves the cluster state toward the desired state for a KomputerAgent.
func (r *KomputerAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch the KomputerAgent CR
	agent := &komputerv1alpha1.KomputerAgent{}
	if err := r.Get(ctx, req.NamespacedName, agent); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle office-cleanup finalizer for managers being deleted.
	// Note: We delete the Office first (which has its own finalizer to clean up members),
	// then remove this finalizer. The Office finalizer handles member cleanup independently.
	if !agent.DeletionTimestamp.IsZero() && controllerutil.ContainsFinalizer(agent, "komputer.ai/office-cleanup") {
		officeName := agent.Name + "-office"
		office := &komputerv1alpha1.KomputerOffice{}
		if err := r.Get(ctx, types.NamespacedName{Name: officeName, Namespace: agent.Namespace}, office); err == nil {
			if err := r.Delete(ctx, office); err != nil && !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete office for manager", "office", officeName)
				return ctrl.Result{}, err
			}
		}
		controllerutil.RemoveFinalizer(agent, "komputer.ai/office-cleanup")
		if err := r.Update(ctx, agent); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// 2. Resolve the template
	templateRef := agent.Spec.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}
	// Resolve template: namespaced KomputerAgentTemplate first, then cluster-scoped KomputerAgentClusterTemplate.
	template := &komputerv1alpha1.KomputerAgentTemplate{}
	if err := r.Get(ctx, types.NamespacedName{Name: templateRef, Namespace: agent.Namespace}, template); err != nil {
		// Fall back to cluster-scoped template.
		clusterTemplate := &komputerv1alpha1.KomputerAgentClusterTemplate{}
		if clusterErr := r.Get(ctx, types.NamespacedName{Name: templateRef}, clusterTemplate); clusterErr != nil {
			log.Error(err, "Failed to get template", "templateRef", templateRef, "namespace", agent.Namespace)
			_ = r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
				s.Phase = komputerv1alpha1.AgentPhasePending
				s.Message = fmt.Sprintf("Template %q not found (checked namespace %q and cluster scope)", templateRef, agent.Namespace)
			})
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		// Convert cluster template to template format for buildPod.
		template = &komputerv1alpha1.KomputerAgentTemplate{
			Spec: *clusterTemplate.Spec.DeepCopy(),
		}
	}

	// 3. Auto-discover the singleton cluster-scoped KomputerConfig
	komputerConfig, err := r.getConfig(ctx)
	if err != nil {
		log.Error(err, "Failed to get KomputerConfig")
		_ = r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
			s.Phase = komputerv1alpha1.AgentPhasePending
			s.Message = "No KomputerConfig found in the cluster"
		})
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// 4. Ensure PVC exists
	pvcName := agent.Name + "-pvc"
	if err := r.ensurePVC(ctx, agent, template, pvcName); err != nil {
		log.Error(err, "Failed to ensure PVC")
		return ctrl.Result{}, err
	}

	// 5. Set owner references on agent secrets so they're garbage-collected on deletion
	for _, secretName := range agent.Spec.Secrets {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, secret); err != nil {
			continue // secret may not exist yet
		}
		if len(secret.OwnerReferences) == 0 {
			if err := ctrl.SetControllerReference(agent, secret, r.Scheme); err == nil {
				if err := r.Update(ctx, secret); err != nil {
					log.Error(err, "Failed to set owner reference on secret", "secret", secretName)
				}
			}
		}
	}

	// 6. Ensure Pod exists
	podName := agent.Name + "-pod"
	pod, err := r.ensurePod(ctx, agent, template, pvcName, podName, komputerConfig)
	if err != nil {
		log.Error(err, "Failed to ensure Pod")
		return ctrl.Result{}, err
	}

	// 8. Update CR status based on pod state
	if err := r.reconcileStatus(ctx, agent, pod, pvcName, podName); err != nil {
		log.Error(err, "Failed to reconcile status")
		return ctrl.Result{}, err
	}

	// 9. Create KomputerOffice if this agent has an officeManager
	if agent.Spec.OfficeManager != "" {
		officeName := agent.Spec.OfficeManager + "-office"
		office := &komputerv1alpha1.KomputerOffice{}
		err := r.Get(ctx, types.NamespacedName{Name: officeName, Namespace: agent.Namespace}, office)
		if errors.IsNotFound(err) {
			// Create the office
			now := metav1.Now()
			office = &komputerv1alpha1.KomputerOffice{
				ObjectMeta: metav1.ObjectMeta{
					Name:       officeName,
					Namespace:  agent.Namespace,
					Finalizers: []string{"komputer.ai/office-members"},
				},
				Spec: komputerv1alpha1.KomputerOfficeSpec{
					Manager: agent.Spec.OfficeManager,
				},
				Status: komputerv1alpha1.KomputerOfficeStatus{
					CreatedAt: &now,
				},
			}
			if createErr := r.Create(ctx, office); createErr != nil && !errors.IsAlreadyExists(createErr) {
				log.Error(createErr, "Failed to create KomputerOffice", "office", officeName)
			}
		}

		// Ensure the office label is on this agent
		if agent.Labels == nil {
			agent.Labels = map[string]string{}
		}
		if agent.Labels["komputer.ai/office"] != officeName {
			original := agent.DeepCopy()
			agent.Labels["komputer.ai/office"] = officeName
			if err := r.Patch(ctx, agent, client.MergeFrom(original)); err != nil {
				log.Error(err, "Failed to add office label to agent", "agent", agent.Name)
				return ctrl.Result{}, err
			}
		}
	}

	// 10. Add office-cleanup finalizer and office label to manager agents that have an office
	if agent.Spec.Role == "manager" {
		officeName := agent.Name + "-office"
		office := &komputerv1alpha1.KomputerOffice{}
		if err := r.Get(ctx, types.NamespacedName{Name: officeName, Namespace: agent.Namespace}, office); err == nil {
			needsUpdate := false
			// Ensure the manager also has the office label (issue: manager was missing it)
			if agent.Labels == nil {
				agent.Labels = map[string]string{}
			}
			if agent.Labels["komputer.ai/office"] != officeName {
				agent.Labels["komputer.ai/office"] = officeName
				needsUpdate = true
			}
			// Ensure finalizer
			if !controllerutil.ContainsFinalizer(agent, "komputer.ai/office-cleanup") {
				controllerutil.AddFinalizer(agent, "komputer.ai/office-cleanup")
				needsUpdate = true
			}
			if needsUpdate {
				if err := r.Update(ctx, agent); err != nil {
					log.Error(err, "Failed to update manager with office label/finalizer")
					return ctrl.Result{}, err
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

// getConfig lists cluster-scoped KomputerConfig resources and returns the first one.
func (r *KomputerAgentReconciler) getConfig(ctx context.Context) (*komputerv1alpha1.KomputerConfig, error) {
	list := &komputerv1alpha1.KomputerConfigList{}
	if err := r.List(ctx, list); err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("no KomputerConfig found in the cluster")
	}
	return &list.Items[0], nil
}

// ensurePVC creates a PVC if it does not exist.
func (r *KomputerAgentReconciler) ensurePVC(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, template *komputerv1alpha1.KomputerAgentTemplate, pvcName string) error {
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: agent.Namespace}, pvc)
	if err == nil {
		return nil // already exists
	}
	if !errors.IsNotFound(err) {
		return err
	}

	size := template.Spec.Storage.Size
	if size == "" {
		size = "5Gi"
	}

	storageQty, err := resource.ParseQuantity(size)
	if err != nil {
		return fmt.Errorf("invalid storage size %q: %w", size, err)
	}

	pvc = &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: agent.Namespace,
			Labels: map[string]string{
				"komputer.ai/agent-name": agent.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageQty,
				},
			},
		},
	}

	if template.Spec.Storage.StorageClassName != nil {
		pvc.Spec.StorageClassName = template.Spec.Storage.StorageClassName
	}

	// Set owner reference
	if err := ctrl.SetControllerReference(agent, pvc, r.Scheme); err != nil {
		return err
	}

	return r.Create(ctx, pvc)
}

// ensureConfigMap creates a ConfigMap with config.json containing redis config.
func (r *KomputerAgentReconciler) ensureConfigMap(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, config *komputerv1alpha1.KomputerConfig, configMapName string) error {
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: agent.Namespace}, cm)
	if err == nil {
		return nil // already exists
	}
	if !errors.IsNotFound(err) {
		return err
	}

	redis := config.Spec.Redis

	// Resolve the Redis password from a Kubernetes Secret if configured.
	password := ""
	if redis.PasswordSecret != nil {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      redis.PasswordSecret.Name,
			Namespace: agent.Namespace,
		}, secret); err != nil {
			return fmt.Errorf("failed to get redis password secret %q: %w", redis.PasswordSecret.Name, err)
		}
		if val, ok := secret.Data[redis.PasswordSecret.Key]; ok {
			password = string(val)
		} else {
			return fmt.Errorf("key %q not found in secret %q", redis.PasswordSecret.Key, redis.PasswordSecret.Name)
		}
	}

	// Build config.json content
	configData := map[string]interface{}{
		"redis": map[string]interface{}{
			"address":       redis.Address,
			"password":      password,
			"db":            redis.DB,
			"stream_prefix": redis.StreamPrefix,
		},
	}

	configJSON, err := json.Marshal(configData)
	if err != nil {
		return err
	}

	cm = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: agent.Namespace,
			Labels: map[string]string{
				"komputer.ai/agent-name": agent.Name,
			},
		},
		Data: map[string]string{
			"config.json": string(configJSON),
		},
	}

	if err := ctrl.SetControllerReference(agent, cm, r.Scheme); err != nil {
		return err
	}

	return r.Create(ctx, cm)
}

// ensurePod creates a Pod if it does not exist, or deletes it if it is in a terminal state
// and the agent status has already been persisted as terminal (two-phase deletion).
func (r *KomputerAgentReconciler) ensurePod(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, template *komputerv1alpha1.KomputerAgentTemplate, pvcName, podName string, config *komputerv1alpha1.KomputerConfig) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: agent.Namespace}, pod)
	if err == nil {
		// If pod is in a terminal state (Failed/Succeeded), return it so
		// reconcileStatus can persist the terminal phase. Don't delete or
		// recreate — a failed pod stays failed until a new task is sent
		// via the API (which creates a new agent or wakes a sleeping one).
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return pod, nil
		}
		return pod, nil
	}
	if !errors.IsNotFound(err) {
		return nil, err
	}

	// Don't create a pod for a sleeping agent — wait for wake-up via API
	if agent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		return nil, nil
	}

	// Build and create the pod
	pod, err = r.buildPod(ctx, agent, template, pvcName, podName, config)
	if err != nil {
		return nil, err
	}

	if err := ctrl.SetControllerReference(agent, pod, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, pod); err != nil {
		return nil, err
	}

	return pod, nil
}

// buildPod deep copies the template PodSpec and injects env/volumes/mounts.
func (r *KomputerAgentReconciler) buildPod(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, template *komputerv1alpha1.KomputerAgentTemplate, pvcName, podName string, config *komputerv1alpha1.KomputerConfig) (*corev1.Pod, error) {
	log := logf.FromContext(ctx)
	podSpec := *template.Spec.PodSpec.DeepCopy()

	if len(podSpec.Containers) == 0 {
		return nil, fmt.Errorf("template %q has no containers defined", template.Name)
	}

	// Set restart policy
	podSpec.RestartPolicy = corev1.RestartPolicyNever

	// Add volumes
	podSpec.Volumes = append(podSpec.Volumes,
		corev1.Volume{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		},
	)

	// Inject into first container
	container := &podSpec.Containers[0]

	// Add env vars
	redis := config.Spec.Redis
	envVars := []corev1.EnvVar{
		{Name: "KOMPUTER_INSTRUCTIONS", Value: agent.Spec.Instructions},
		{Name: "KOMPUTER_MODEL", Value: agent.Spec.Model},
		{Name: "KOMPUTER_AGENT_NAME", Value: agent.Name},
		{Name: "KOMPUTER_NAMESPACE", Value: agent.Namespace},
		{Name: "CLAUDE_CONFIG_DIR", Value: "/workspace/.claude"},
		// Redis config as env vars (no ConfigMap needed).
		{Name: "KOMPUTER_REDIS_ADDRESS", Value: redis.Address},
		{Name: "KOMPUTER_REDIS_DB", Value: fmt.Sprintf("%d", redis.DB)},
		{Name: "KOMPUTER_REDIS_STREAM_PREFIX", Value: redis.StreamPrefix},
	}
	// Redis password from Secret (stays as a Secret, not plaintext).
	if redis.PasswordSecret != nil && redis.PasswordSecret.Name != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name: "KOMPUTER_REDIS_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: redis.PasswordSecret.Name},
					Key:                  redis.PasswordSecret.Key,
				},
			},
		})
	}
	if agent.Spec.Role == "manager" {
		envVars = append(envVars,
			corev1.EnvVar{Name: "KOMPUTER_ROLE", Value: agent.Spec.Role},
			corev1.EnvVar{Name: "KOMPUTER_API_URL", Value: config.Spec.APIURL},
		)
	}
	// Inject env vars from agent secrets.
	for _, secretName := range agent.Spec.Secrets {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, secret); err != nil {
			log.Error(err, "Failed to get agent secret", "secret", secretName)
			continue
		}
		for key := range secret.Data {
			envVars = append(envVars, corev1.EnvVar{
				Name: key,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
						Key:                  key,
					},
				},
			})
		}
	}

	// Dedup env vars: agent secrets override template env vars with same name.
	existingKeys := make(map[string]bool, len(container.Env))
	for _, env := range container.Env {
		existingKeys[env.Name] = true
	}
	for _, env := range envVars {
		if existingKeys[env.Name] {
			// Replace existing env var from template.
			for i, existing := range container.Env {
				if existing.Name == env.Name {
					container.Env[i] = env
					break
				}
			}
		} else {
			container.Env = append(container.Env, env)
		}
	}

	// Add volume mounts
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      "workspace",
			MountPath: "/workspace",
		},
	)

	// Inject health probes (override any existing ones from the template).
	container.LivenessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/healthz",
				Port: intstr.FromInt(8000),
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       30,
	}
	container.ReadinessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/readyz",
				Port: intstr.FromInt(8000),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       10,
	}

	// Graceful shutdown: cancel running task and wait for cleanup before pod termination.
	container.Lifecycle = &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/shutdown",
				Port: intstr.FromInt(8000),
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: agent.Namespace,
			Labels: map[string]string{
				"komputer.ai/agent-name": agent.Name,
			},
		},
		Spec: podSpec,
	}

	return pod, nil
}

// reconcileStatus maps pod phase to agent phase and updates status.
// Also handles lifecycle transitions (Sleep → delete pod, AutoDelete → delete CR).
func (r *KomputerAgentReconciler) reconcileStatus(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, pod *corev1.Pod, pvcName, podName string) error {
	log := logf.FromContext(ctx)

	// Sleep mode: delete pod when task is complete, keep PVC
	if agent.Spec.Lifecycle == komputerv1alpha1.AgentLifecycleSleep &&
		agent.Status.TaskStatus == komputerv1alpha1.AgentTaskComplete &&
		pod != nil {
		log.Info("Sleep mode: deleting pod after task completion", "agent", agent.Name)
		if err := r.Delete(ctx, pod); err != nil {
			return err
		}
		return r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
			s.Phase = komputerv1alpha1.AgentPhaseSleeping
			s.PodName = ""
			s.PvcName = pvcName
			s.Message = "Sleeping — pod deleted, workspace preserved. Send a new task to wake up."
		})
	}

	// AutoDelete mode: delete the entire agent CR when task is complete
	if agent.Spec.Lifecycle == komputerv1alpha1.AgentLifecycleAutoDelete &&
		agent.Status.TaskStatus == komputerv1alpha1.AgentTaskComplete {
		log.Info("AutoDelete mode: deleting agent after task completion", "agent", agent.Name)
		return r.Delete(ctx, agent)
	}

	return r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
		s.PodName = podName
		s.PvcName = pvcName

		if pod == nil {
			// If agent is sleeping, preserve the sleeping state
			if agent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
				s.PodName = ""
				return
			}
			s.Phase = komputerv1alpha1.AgentPhasePending
			s.Message = "Pod is being created"
			return
		}

		switch pod.Status.Phase {
		case corev1.PodRunning:
			s.Phase = komputerv1alpha1.AgentPhaseRunning
			s.Message = "Agent is running"
			if s.StartTime == nil {
				now := metav1.Now()
				s.StartTime = &now
			}
		case corev1.PodSucceeded:
			s.Phase = komputerv1alpha1.AgentPhaseSucceeded
			s.Message = "Agent completed successfully"
			if s.CompletionTime == nil {
				now := metav1.Now()
				s.CompletionTime = &now
			}
		case corev1.PodFailed:
			s.Phase = komputerv1alpha1.AgentPhaseFailed
			s.Message = "Agent failed"
			if s.CompletionTime == nil {
				now := metav1.Now()
				s.CompletionTime = &now
			}
		default:
			s.Phase = komputerv1alpha1.AgentPhasePending
			s.Message = fmt.Sprintf("Pod phase: %s", pod.Status.Phase)
		}
	})
}

// updateStatus uses variadic extras pattern for status updates.
func (r *KomputerAgentReconciler) updateStatus(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, extras ...func(*komputerv1alpha1.KomputerAgentStatus)) error {
	for _, fn := range extras {
		fn(&agent.Status)
	}
	return r.Status().Update(ctx, agent)
}

// SetupWithManager sets up the controller with the Manager.
func (r *KomputerAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&komputerv1alpha1.KomputerAgent{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.ConfigMap{}).
		Named("komputeragent").
		Complete(r)
}
