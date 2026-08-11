package controller

import (
	"reflect"
	"testing"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildScheduledAgent_CarriesFullConfig(t *testing.T) {
	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerScheduleSpec{
			Instructions: "run the nightly report",
			Agent: &komputerv1alpha1.ScheduleAgentSpec{
				AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
					Model:           "claude-opus-4-6",
					Role:            "worker",
					TemplateRef:     "big",
					Skills:          []string{"sql"},
					Memories:        []string{"schema"},
					Connectors:      []string{"github"},
					Secrets:         []string{"gh-token"},
					AllowedTools:    []string{"Read", "mcp__github__*"},
					DisallowedTools: []string{"Bash"},
					SystemPrompt:    "be terse",
					Priority:        50,
					Lifecycle:       komputerv1alpha1.AgentLifecycleSleep,
					Storage:         &komputerv1alpha1.StorageSpec{Size: "20Gi"},
					PodSpec: &corev1.PodSpec{
						Containers: []corev1.Container{{Name: "agent", Image: "custom:1"}},
					},
					Labels: map[string]string{"team": "data"},
				},
			},
		},
	}

	agent := buildScheduledAgent(schedule, "nightly-agent")

	// Assert the whole config structurally rather than field by field. A
	// field-by-field test silently stops covering AgentConfigSpec the moment
	// someone adds a 16th field, which is exactly the guarantee this function's
	// doc comment makes. Structural equality keeps the test self-maintaining.
	if !reflect.DeepEqual(agent.Spec.AgentConfigSpec, schedule.Spec.Agent.AgentConfigSpec) {
		t.Errorf("agent config does not match the schedule's config\n got: %+v\nwant: %+v",
			agent.Spec.AgentConfigSpec, schedule.Spec.Agent.AgentConfigSpec)
	}

	// Not covered by the equality check above: these two are the schedule's, not
	// the template's.
	if agent.Spec.Instructions != "run the nightly report" {
		t.Errorf("instructions must come from the schedule, got %q", agent.Spec.Instructions)
	}
	if agent.Labels["komputer.ai/schedule"] != "nightly" {
		t.Errorf("schedule label missing: %v", agent.Labels)
	}
}

// The template must not alias the schedule's memory: mutating the produced
// agent must never write back into the schedule CR the informer handed us.
func TestBuildScheduledAgent_DeepCopiesConfig(t *testing.T) {
	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerScheduleSpec{
			Instructions: "go",
			Agent: &komputerv1alpha1.ScheduleAgentSpec{
				AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Skills: []string{"sql"}},
			},
		},
	}

	agent := buildScheduledAgent(schedule, "s-agent")
	agent.Spec.Skills[0] = "mutated"

	if schedule.Spec.Agent.Skills[0] != "sql" {
		t.Errorf("schedule spec was aliased and mutated: %v", schedule.Spec.Agent.Skills)
	}
}
