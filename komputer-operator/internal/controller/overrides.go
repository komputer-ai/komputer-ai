package controller

import (
	corev1 "k8s.io/api/core/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// applyAgentOverrides returns a deep copy of the template with the agent's
// per-container fields and storage overrides merged in. Pure function — does
// NOT mutate inputs. Overrides only affect future pod / PVC creation;
// existing pods and PVCs are not modified.
//
// Container merging is by name: for each container in the agent's PodSpec,
// the matching container in the template (by name) gets its non-zero fields
// overlaid. Containers in the agent's PodSpec that don't exist in the
// template are appended. Top-level PodSpec fields (nodeSelector, tolerations,
// etc.) from the agent override their template counterparts when set.
func applyAgentOverrides(template *komputerv1alpha1.KomputerAgentTemplate, agent *komputerv1alpha1.KomputerAgent) *komputerv1alpha1.KomputerAgentTemplate {
	out := template.DeepCopy()
	if agent.Spec.Storage != nil {
		out.Spec.Storage = mergeStorageSpec(out.Spec.Storage, agent.Spec.Storage)
	}
	if agent.Spec.PodSpec != nil {
		mergePodSpec(&out.Spec.PodSpec, agent.Spec.PodSpec)
	}
	if agent.Spec.SleepTTL != nil {
		out.Spec.SleepTTL = agent.Spec.SleepTTL.DeepCopy()
	}
	if agent.Spec.DeleteTTL != nil {
		out.Spec.DeleteTTL = agent.Spec.DeleteTTL.DeepCopy()
	}
	if agent.Spec.ArchiveTTL != nil {
		out.Spec.ArchiveTTL = agent.Spec.ArchiveTTL.DeepCopy()
	}
	if agent.Spec.TaskTimeout != nil {
		out.Spec.TaskTimeout = agent.Spec.TaskTimeout.DeepCopy()
	}
	if agent.Spec.BudgetUSD != "" {
		out.Spec.BudgetUSD = agent.Spec.BudgetUSD
	}
	if agent.Spec.TaskBudgetUSD != "" {
		out.Spec.TaskBudgetUSD = agent.Spec.TaskBudgetUSD
	}
	return out
}

// mergeStorageSpec overlays non-zero fields from `override` onto `base`, field by
// field, so setting one storage field (e.g. size) doesn't clear another (e.g.
// ephemeral) that the override left unset.
func mergeStorageSpec(base komputerv1alpha1.StorageSpec, override *komputerv1alpha1.StorageSpec) komputerv1alpha1.StorageSpec {
	merged := *base.DeepCopy()
	if override == nil {
		return merged
	}
	if override.Size != "" {
		merged.Size = override.Size
	}
	if override.StorageClassName != nil {
		merged.StorageClassName = override.StorageClassName
	}
	if override.Ephemeral != nil {
		merged.Ephemeral = override.Ephemeral
	}
	return merged
}

// isEphemeralStorage resolves a (possibly merged) StorageSpec's effective ephemeral
// value: nil (unset) resolves to false, preserving today's "every agent gets a PVC"
// behavior for the overwhelming majority of agents that never set this field.
func isEphemeralStorage(s komputerv1alpha1.StorageSpec) bool {
	return s.Ephemeral != nil && *s.Ephemeral
}

// mergePodSpec overlays non-zero fields from `override` onto `base`. Containers
// are merged by name; new containers are appended.
func mergePodSpec(base *corev1.PodSpec, override *corev1.PodSpec) {
	for _, oc := range override.Containers {
		idx := -1
		for i, bc := range base.Containers {
			if bc.Name == oc.Name {
				idx = i
				break
			}
		}
		if idx == -1 {
			base.Containers = append(base.Containers, *oc.DeepCopy())
			continue
		}
		mergeContainer(&base.Containers[idx], &oc)
	}

	if override.NodeSelector != nil {
		base.NodeSelector = override.NodeSelector
	}
	if override.Tolerations != nil {
		base.Tolerations = override.Tolerations
	}
	if override.Affinity != nil {
		base.Affinity = override.Affinity
	}
	if override.PriorityClassName != "" {
		base.PriorityClassName = override.PriorityClassName
	}
	if override.RuntimeClassName != nil {
		base.RuntimeClassName = override.RuntimeClassName
	}
	if override.ServiceAccountName != "" {
		base.ServiceAccountName = override.ServiceAccountName
	}
}

// mergeContainer overlays non-zero fields from `override` onto `base`.
func mergeContainer(base *corev1.Container, override *corev1.Container) {
	if override.Image != "" {
		base.Image = override.Image
	}
	if len(override.Command) > 0 {
		base.Command = override.Command
	}
	if len(override.Args) > 0 {
		base.Args = override.Args
	}
	if override.WorkingDir != "" {
		base.WorkingDir = override.WorkingDir
	}
	if len(override.Resources.Limits) > 0 {
		if base.Resources.Limits == nil {
			base.Resources.Limits = corev1.ResourceList{}
		}
		for k, v := range override.Resources.Limits {
			base.Resources.Limits[k] = v
		}
	}
	if len(override.Resources.Requests) > 0 {
		if base.Resources.Requests == nil {
			base.Resources.Requests = corev1.ResourceList{}
		}
		for k, v := range override.Resources.Requests {
			base.Resources.Requests[k] = v
		}
	}
	for _, oe := range override.Env {
		idx := -1
		for i, be := range base.Env {
			if be.Name == oe.Name {
				idx = i
				break
			}
		}
		if idx == -1 {
			base.Env = append(base.Env, oe)
		} else {
			base.Env[idx] = oe
		}
	}
}
