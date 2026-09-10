package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func toolPolicyEnvValue(envVars []corev1.EnvVar, name string) (string, bool) {
	for _, e := range envVars {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}

func TestToolPolicyEnvVarsOmittedWhenUnset(t *testing.T) {
	envVars := buildToolPolicyEnvVars(&komputerv1alpha1.KomputerAgentSpec{})

	if _, ok := toolPolicyEnvValue(envVars, "KOMPUTER_ALLOWED_TOOLS"); ok {
		t.Error("KOMPUTER_ALLOWED_TOOLS must be omitted when AllowedTools is empty")
	}
	if _, ok := toolPolicyEnvValue(envVars, "KOMPUTER_DISALLOWED_TOOLS"); ok {
		t.Error("KOMPUTER_DISALLOWED_TOOLS must be omitted when DisallowedTools is empty")
	}
}

func TestToolPolicyEnvVarsSerializeAsJSONArrays(t *testing.T) {
	envVars := buildToolPolicyEnvVars(&komputerv1alpha1.KomputerAgentSpec{
		AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
			AllowedTools:    []string{"Read", "mcp__figma__*"},
			DisallowedTools: []string{"Bash"},
		},
	})

	got, ok := toolPolicyEnvValue(envVars, "KOMPUTER_ALLOWED_TOOLS")
	if !ok {
		t.Fatal("KOMPUTER_ALLOWED_TOOLS missing")
	}
	if want := `["Read","mcp__figma__*"]`; got != want {
		t.Errorf("allowed = %q, want %q", got, want)
	}

	got, ok = toolPolicyEnvValue(envVars, "KOMPUTER_DISALLOWED_TOOLS")
	if !ok {
		t.Fatal("KOMPUTER_DISALLOWED_TOOLS missing")
	}
	if want := `["Bash"]`; got != want {
		t.Errorf("disallowed = %q, want %q", got, want)
	}
}

func TestToolPolicyEnvVarsIndependent(t *testing.T) {
	envVars := buildToolPolicyEnvVars(&komputerv1alpha1.KomputerAgentSpec{
		AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
			DisallowedTools: []string{"mcp__figma__use_figma"},
		},
	})
	if _, ok := toolPolicyEnvValue(envVars, "KOMPUTER_ALLOWED_TOOLS"); ok {
		t.Error("allowed must stay omitted when only disallowed is set")
	}
	if _, ok := toolPolicyEnvValue(envVars, "KOMPUTER_DISALLOWED_TOOLS"); !ok {
		t.Error("disallowed must be present")
	}
}
