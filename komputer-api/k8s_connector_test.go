package main

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestResolveConnectorMCPConfigsSkipsDisabled verifies that a disabled connector is
// left out of the resolved MCP server config map entirely — its tools must not be
// available to the agent that receives this config (new pod, wake, or live PATCH).
func TestResolveConnectorMCPConfigsSkipsDisabled(t *testing.T) {
	enabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "github", URL: "https://mcp.example.com/github"},
	}
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	k8s := newFakeK8s(t, enabled, disabled)

	result := k8s.ResolveConnectorMCPConfigs(context.Background(), "default", []string{"github", "slack"})

	if _, ok := result["github"]; !ok {
		t.Error("expected enabled connector 'github' to be present in resolved config")
	}
	if _, ok := result["slack"]; ok {
		t.Error("expected disabled connector 'slack' to be skipped from resolved config")
	}
}

// TestResolveConnectorMCPConfigsSkipsDisabledCrossNamespace verifies the "namespace/name"
// connector reference form is honored by the disabled check too.
func TestResolveConnectorMCPConfigsSkipsDisabledCrossNamespace(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-slack", Namespace: "shared"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	k8s := newFakeK8s(t, disabled)

	result := k8s.ResolveConnectorMCPConfigs(context.Background(), "default", []string{"shared/shared-slack"})

	if _, ok := result["shared-slack"]; ok {
		t.Error("expected disabled cross-namespace connector to be skipped from resolved config")
	}
}
