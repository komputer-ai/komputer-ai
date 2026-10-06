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
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestBuildAgentEnvVarsSkipsDisabledConnector verifies that a disabled connector's
// MCP server config is not injected into the agent pod — its tools must not be
// available to the agent. An enabled connector referenced alongside it must still
// be included.
func TestBuildAgentEnvVarsSkipsDisabledConnector(t *testing.T) {
	scheme := newTestScheme(t)
	enabledConn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "github", URL: "https://mcp.example.com/github"},
	}
	disabledConn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(enabledConn, disabledConn).Build()

	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Connectors: []string{"github", "slack"},
			},
			Instructions: "do stuff",
		},
	}
	config := &komputerv1alpha1.KomputerConfig{}

	envVars, err := buildAgentEnvVars(context.Background(), c, agent, config)
	if err != nil {
		t.Fatalf("buildAgentEnvVars returned error: %v", err)
	}

	val, ok := toolPolicyEnvValue(envVars, "KOMPUTER_MCP_SERVERS")
	if !ok {
		t.Fatal("KOMPUTER_MCP_SERVERS missing — expected the enabled connector to still be injected")
	}
	if !strings.Contains(val, `"github"`) {
		t.Errorf("KOMPUTER_MCP_SERVERS = %q, want it to include the enabled connector %q", val, "github")
	}
	if strings.Contains(val, `"slack"`) {
		t.Errorf("KOMPUTER_MCP_SERVERS = %q, must not include the disabled connector %q", val, "slack")
	}
}

// TestBuildAgentEnvVarsOmitsMCPServersWhenAllDisabled verifies that if every
// attached connector is disabled, no KOMPUTER_MCP_SERVERS env var is emitted at all.
func TestBuildAgentEnvVarsOmitsMCPServersWhenAllDisabled(t *testing.T) {
	scheme := newTestScheme(t)
	disabledConn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(disabledConn).Build()

	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Connectors: []string{"slack"},
			},
			Instructions: "do stuff",
		},
	}
	config := &komputerv1alpha1.KomputerConfig{}

	envVars, err := buildAgentEnvVars(context.Background(), c, agent, config)
	if err != nil {
		t.Fatalf("buildAgentEnvVars returned error: %v", err)
	}

	if _, ok := toolPolicyEnvValue(envVars, "KOMPUTER_MCP_SERVERS"); ok {
		t.Error("KOMPUTER_MCP_SERVERS must be omitted when every attached connector is disabled")
	}
}
