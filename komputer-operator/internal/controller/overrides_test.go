package controller

import (
	"testing"

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
