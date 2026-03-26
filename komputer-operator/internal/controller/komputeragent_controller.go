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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
// +kubebuilder:rbac:groups=komputer.komputer.ai,resources=komputerredisconfigs,verbs=get;list;watch
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

	// 2. Resolve the template
	templateRef := agent.Spec.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}
	template := &komputerv1alpha1.KomputerAgentTemplate{}
	if err := r.Get(ctx, types.NamespacedName{Name: templateRef, Namespace: agent.Namespace}, template); err != nil {
		log.Error(err, "Failed to get KomputerAgentTemplate", "templateRef", templateRef)
		return ctrl.Result{}, r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
			s.Phase = komputerv1alpha1.AgentPhasePending
			s.Message = fmt.Sprintf("Template %q not found", templateRef)
		})
	}

	// 3. Auto-discover the singleton KomputerRedisConfig in the namespace
	redisConfig, err := r.getRedisConfig(ctx, agent.Namespace)
	if err != nil {
		log.Error(err, "Failed to get KomputerRedisConfig")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
			s.Phase = komputerv1alpha1.AgentPhasePending
			s.Message = "No KomputerRedisConfig found in namespace"
		})
	}

	// 4. Ensure PVC exists
	pvcName := agent.Name + "-pvc"
	if err := r.ensurePVC(ctx, agent, template, pvcName); err != nil {
		log.Error(err, "Failed to ensure PVC")
		return ctrl.Result{}, err
	}

	// 5. Ensure ConfigMap exists
	configMapName := agent.Name + "-pod-config"
	if err := r.ensureConfigMap(ctx, agent, redisConfig, configMapName); err != nil {
		log.Error(err, "Failed to ensure ConfigMap")
		return ctrl.Result{}, err
	}

	// 6. Ensure Pod exists
	podName := agent.Name + "-pod"
	pod, err := r.ensurePod(ctx, agent, template, pvcName, configMapName, podName)
	if err != nil {
		log.Error(err, "Failed to ensure Pod")
		return ctrl.Result{}, err
	}

	// 8. Update CR status based on pod state
	if err := r.reconcileStatus(ctx, agent, pod, pvcName, podName); err != nil {
		log.Error(err, "Failed to reconcile status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// getRedisConfig lists KomputerRedisConfig in namespace and returns the first one.
func (r *KomputerAgentReconciler) getRedisConfig(ctx context.Context, namespace string) (*komputerv1alpha1.KomputerRedisConfig, error) {
	list := &komputerv1alpha1.KomputerRedisConfigList{}
	if err := r.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("no KomputerRedisConfig found in namespace %s", namespace)
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
					corev1.ResourceStorage: resource.MustParse(size),
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
func (r *KomputerAgentReconciler) ensureConfigMap(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, redisConfig *komputerv1alpha1.KomputerRedisConfig, configMapName string) error {
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: agent.Namespace}, cm)
	if err == nil {
		return nil // already exists
	}
	if !errors.IsNotFound(err) {
		return err
	}

	// Build config.json content
	configData := map[string]interface{}{
		"redis": map[string]interface{}{
			"address":  redisConfig.Spec.Address,
			"password": "",
			"db":       redisConfig.Spec.DB,
			"queue":    redisConfig.Spec.Queue,
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

// ensurePod creates a Pod if it does not exist, or deletes it if it is in a terminal state.
func (r *KomputerAgentReconciler) ensurePod(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, template *komputerv1alpha1.KomputerAgentTemplate, pvcName, configMapName, podName string) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: agent.Namespace}, pod)
	if err == nil {
		// 7. If pod is Failed/Succeeded, delete it (next reconcile recreates)
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			if err := r.Delete(ctx, pod); err != nil {
				return nil, err
			}
			return nil, nil
		}
		return pod, nil
	}
	if !errors.IsNotFound(err) {
		return nil, err
	}

	// Build and create the pod
	pod = r.buildPod(agent, template, pvcName, configMapName, podName)

	if err := ctrl.SetControllerReference(agent, pod, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, pod); err != nil {
		return nil, err
	}

	return pod, nil
}

// buildPod deep copies the template PodSpec and injects env/volumes/mounts.
func (r *KomputerAgentReconciler) buildPod(agent *komputerv1alpha1.KomputerAgent, template *komputerv1alpha1.KomputerAgentTemplate, pvcName, configMapName, podName string) *corev1.Pod {
	podSpec := *template.Spec.PodSpec.DeepCopy()

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
		corev1.Volume{
			Name: "komputer-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		},
	)

	// Inject into first container
	if len(podSpec.Containers) > 0 {
		container := &podSpec.Containers[0]

		// Add env vars
		container.Env = append(container.Env,
			corev1.EnvVar{Name: "KOMPUTER_INSTRUCTIONS", Value: agent.Spec.Instructions},
			corev1.EnvVar{Name: "KOMPUTER_MODEL", Value: agent.Spec.Model},
			corev1.EnvVar{Name: "KOMPUTER_AGENT_NAME", Value: agent.Name},
		)

		// Add volume mounts
		container.VolumeMounts = append(container.VolumeMounts,
			corev1.VolumeMount{
				Name:      "workspace",
				MountPath: "/workspace",
			},
			corev1.VolumeMount{
				Name:      "komputer-config",
				MountPath: "/etc/komputer",
				ReadOnly:  true,
			},
		)
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

	return pod
}

// reconcileStatus maps pod phase to agent phase and updates status.
func (r *KomputerAgentReconciler) reconcileStatus(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, pod *corev1.Pod, pvcName, podName string) error {
	return r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
		s.PodName = podName
		s.PvcName = pvcName

		if pod == nil {
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
		Named("komputeragent").
		Complete(r)
}
