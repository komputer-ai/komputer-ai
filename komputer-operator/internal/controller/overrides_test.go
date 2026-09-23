package controller

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func tplFixture() *komputerv1alpha1.KomputerAgentTemplate {
	return &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
			Storage: komputerv1alpha1.StorageSpec{Size: "5Gi"},
			PodSpec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "agent", Image: "img:v1"}},
			},
		},
	}
}

func TestApplyAgentOverrides_NoOverrides(t *testing.T) {
	tpl := tplFixture()
	agent := &komputerv1alpha1.KomputerAgent{}
	out := applyAgentOverrides(tpl, agent)
	if out.Spec.Storage.Size != "5Gi" {
		t.Fatalf("expected storage from template, got %q", out.Spec.Storage.Size)
	}
	if out.Spec.PodSpec.Containers[0].Image != "img:v1" {
		t.Fatalf("expected image from template, got %q", out.Spec.PodSpec.Containers[0].Image)
	}
}

func TestApplyAgentOverrides_StorageOverride(t *testing.T) {
	tpl := tplFixture()
	agent := &komputerv1alpha1.KomputerAgent{
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Storage: &komputerv1alpha1.StorageSpec{Size: "20Gi"},
			},
		},
	}
	out := applyAgentOverrides(tpl, agent)
	if out.Spec.Storage.Size != "20Gi" {
		t.Fatalf("storage not overridden: %s", out.Spec.Storage.Size)
	}
}

func TestApplyAgentOverrides_PodSpecOverride(t *testing.T) {
	tpl := tplFixture()
	agent := &komputerv1alpha1.KomputerAgent{
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				PodSpec: &corev1.PodSpec{
					Containers: []corev1.Container{{Name: "agent", Image: "custom:latest"}},
				},
			},
		},
	}
	out := applyAgentOverrides(tpl, agent)
	if out.Spec.PodSpec.Containers[0].Image != "custom:latest" {
		t.Fatalf("podSpec not overridden: %s", out.Spec.PodSpec.Containers[0].Image)
	}
}

func TestApplyAgentOverrides_PartialContainerMerge_PreservesImage(t *testing.T) {
	tpl := tplFixture()
	agent := &komputerv1alpha1.KomputerAgent{
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				PodSpec: &corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: "agent",
						Resources: corev1.ResourceRequirements{
							Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("4Gi")},
							Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("4Gi")},
						},
					}},
				},
			},
		},
	}
	out := applyAgentOverrides(tpl, agent)
	c := out.Spec.PodSpec.Containers[0]
	if c.Image != "img:v1" {
		t.Fatalf("image should be preserved from template, got %q", c.Image)
	}
	if got := c.Resources.Limits[corev1.ResourceMemory]; got.String() != "4Gi" {
		t.Fatalf("memory limit not merged, got %s", got.String())
	}
}

func TestApplyAgentOverrides_DoesNotMutateInput(t *testing.T) {
	tpl := tplFixture()
	agent := &komputerv1alpha1.KomputerAgent{
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Storage: &komputerv1alpha1.StorageSpec{Size: "20Gi"},
				PodSpec: &corev1.PodSpec{Containers: []corev1.Container{{Name: "agent", Image: "x:1"}}},
			},
		},
	}
	_ = applyAgentOverrides(tpl, agent)
	if tpl.Spec.Storage.Size != "5Gi" {
		t.Fatal("template storage was mutated")
	}
	if tpl.Spec.PodSpec.Containers[0].Image != "img:v1" {
		t.Fatal("template podSpec was mutated")
	}
}

func TestApplyAgentOverrides_TTLs(t *testing.T) {
	t.Run("agent TTLs override template defaults", func(t *testing.T) {
		tpl := tplFixture()
		tpl.Spec.SleepTTL = &metav1.Duration{Duration: time.Hour}
		tpl.Spec.DeleteTTL = &metav1.Duration{Duration: 48 * time.Hour}
		agent := &komputerv1alpha1.KomputerAgent{
			Spec: komputerv1alpha1.KomputerAgentSpec{
				AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
					SleepTTL:  &metav1.Duration{Duration: 30 * time.Minute},
					DeleteTTL: &metav1.Duration{Duration: 24 * time.Hour},
				},
			},
		}
		out := applyAgentOverrides(tpl, agent)
		if out.Spec.SleepTTL.Duration != 30*time.Minute {
			t.Errorf("sleepTTL = %v, want 30m", out.Spec.SleepTTL.Duration)
		}
		if out.Spec.DeleteTTL.Duration != 24*time.Hour {
			t.Errorf("deleteTTL = %v, want 24h", out.Spec.DeleteTTL.Duration)
		}
	})

	t.Run("template TTLs are inherited when the agent sets none", func(t *testing.T) {
		tpl := tplFixture()
		tpl.Spec.SleepTTL = &metav1.Duration{Duration: time.Hour}
		tpl.Spec.DeleteTTL = &metav1.Duration{Duration: 48 * time.Hour}
		out := applyAgentOverrides(tpl, &komputerv1alpha1.KomputerAgent{})
		if out.Spec.SleepTTL.Duration != time.Hour {
			t.Errorf("sleepTTL = %v, want the template's 1h", out.Spec.SleepTTL.Duration)
		}
		if out.Spec.DeleteTTL.Duration != 48*time.Hour {
			t.Errorf("deleteTTL = %v, want the template's 48h", out.Spec.DeleteTTL.Duration)
		}
	})

	t.Run("each TTL is overridden independently", func(t *testing.T) {
		tpl := tplFixture()
		tpl.Spec.SleepTTL = &metav1.Duration{Duration: time.Hour}
		tpl.Spec.DeleteTTL = &metav1.Duration{Duration: 48 * time.Hour}
		agent := &komputerv1alpha1.KomputerAgent{
			Spec: komputerv1alpha1.KomputerAgentSpec{
				AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
					SleepTTL: &metav1.Duration{Duration: 5 * time.Minute},
				},
			},
		}
		out := applyAgentOverrides(tpl, agent)
		if out.Spec.SleepTTL.Duration != 5*time.Minute {
			t.Errorf("sleepTTL = %v, want the agent's 5m", out.Spec.SleepTTL.Duration)
		}
		if out.Spec.DeleteTTL.Duration != 48*time.Hour {
			t.Errorf("deleteTTL = %v, want the template's 48h to survive", out.Spec.DeleteTTL.Duration)
		}
	})

	t.Run("does not mutate the template", func(t *testing.T) {
		tpl := tplFixture()
		tpl.Spec.SleepTTL = &metav1.Duration{Duration: time.Hour}
		agent := &komputerv1alpha1.KomputerAgent{
			Spec: komputerv1alpha1.KomputerAgentSpec{
				AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
					SleepTTL: &metav1.Duration{Duration: 5 * time.Minute},
				},
			},
		}
		_ = applyAgentOverrides(tpl, agent)
		if tpl.Spec.SleepTTL.Duration != time.Hour {
			t.Errorf("template sleepTTL was mutated to %v", tpl.Spec.SleepTTL.Duration)
		}
	})
}

func TestApplyAgentOverridesTaskTimeout(t *testing.T) {
	t.Run("agent value overrides the template", func(t *testing.T) {
		tpl := &komputerv1alpha1.KomputerAgentTemplate{
			Spec: komputerv1alpha1.KomputerAgentTemplateSpec{TaskTimeout: dur(2 * time.Hour)},
		}
		agent := &komputerv1alpha1.KomputerAgent{
			Spec: komputerv1alpha1.KomputerAgentSpec{
				AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{TaskTimeout: dur(30 * time.Minute)},
			},
		}
		out := applyAgentOverrides(tpl, agent)
		if out.Spec.TaskTimeout == nil || out.Spec.TaskTimeout.Duration != 30*time.Minute {
			t.Errorf("TaskTimeout = %v, want 30m", out.Spec.TaskTimeout)
		}
		if tpl.Spec.TaskTimeout.Duration != 2*time.Hour {
			t.Error("applyAgentOverrides mutated the input template")
		}
	})

	t.Run("template value survives when the agent has none", func(t *testing.T) {
		tpl := &komputerv1alpha1.KomputerAgentTemplate{
			Spec: komputerv1alpha1.KomputerAgentTemplateSpec{TaskTimeout: dur(2 * time.Hour)},
		}
		agent := &komputerv1alpha1.KomputerAgent{}
		out := applyAgentOverrides(tpl, agent)
		if out.Spec.TaskTimeout == nil || out.Spec.TaskTimeout.Duration != 2*time.Hour {
			t.Errorf("TaskTimeout = %v, want 2h", out.Spec.TaskTimeout)
		}
	})
}
