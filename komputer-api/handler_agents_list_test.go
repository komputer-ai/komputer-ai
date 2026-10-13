package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func newTestAgent(name, ns string) *komputerv1alpha1.KomputerAgent {
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			CreationTimestamp: metav1.Now(),
		},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Model: "claude-sonnet",
			},
		},
	}
}

func serveListAgents(t *testing.T, k8s *K8sClient, query string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/agents", listAgents(k8s))
	req := httptest.NewRequest(http.MethodGet, "/agents"+query, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestListAgents_NoLimitReturnsEverything is the backward-compatibility
// contract: existing callers (CLI, manager tools, SDK) never pass `limit`,
// and must keep getting the full list back exactly as before.
func TestListAgents_NoLimitReturnsEverything(t *testing.T) {
	var objs []client.Object
	for i := 0; i < 5; i++ {
		objs = append(objs, newTestAgent(fmt.Sprintf("agent-%d", i), "default"))
	}
	k8s := newFakeK8s(t, objs...)

	w := serveListAgents(t, k8s, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp AgentListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Agents) != 5 {
		t.Errorf("len(Agents) = %d, want 5", len(resp.Agents))
	}
	if resp.Total != 5 {
		t.Errorf("Total = %d, want 5", resp.Total)
	}
	if resp.NextOffset != nil {
		t.Errorf("NextOffset = %v, want nil", *resp.NextOffset)
	}
}

func TestListAgents_Pagination(t *testing.T) {
	var objs []client.Object
	for i := 0; i < 5; i++ {
		objs = append(objs, newTestAgent(fmt.Sprintf("agent-%d", i), "default"))
	}
	k8s := newFakeK8s(t, objs...)

	w := serveListAgents(t, k8s, "?limit=2&offset=0&sort=name")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp AgentListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Agents) != 2 {
		t.Fatalf("len(Agents) = %d, want 2", len(resp.Agents))
	}
	if resp.Agents[0].Name != "agent-0" || resp.Agents[1].Name != "agent-1" {
		t.Errorf("unexpected page contents: %+v", resp.Agents)
	}
	if resp.Total != 5 {
		t.Errorf("Total = %d, want 5", resp.Total)
	}
	if resp.NextOffset == nil || *resp.NextOffset != 2 {
		t.Errorf("NextOffset = %v, want 2", resp.NextOffset)
	}
}

func TestListAgents_FacetsPresentEvenWhenFiltered(t *testing.T) {
	a := newTestAgent("match", "default")
	a.Spec.Labels = map[string]string{"env": "prod"}
	b := newTestAgent("other", "default")
	b.Spec.Labels = map[string]string{"env": "dev"}
	k8s := newFakeK8s(t, a, b)

	w := serveListAgents(t, k8s, "?q=match")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp AgentListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Agents) != 1 || resp.Agents[0].Name != "match" {
		t.Fatalf("expected only 'match' agent, got %+v", resp.Agents)
	}
	if len(resp.Facets.Labels) != 2 {
		t.Errorf("Facets.Labels = %v, want both env=prod and env=dev despite the q filter", resp.Facets.Labels)
	}
}

func TestListAgents_ValidationErrors(t *testing.T) {
	k8s := newFakeK8s(t)

	cases := []struct {
		name  string
		query string
	}{
		{"bad sort key", "?sort=bogus"},
		{"negative offset", "?offset=-1"},
		{"non-numeric limit", "?limit=abc"},
		{"limit too large", "?limit=99999"},
		{"zero limit", "?limit=0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := serveListAgents(t, k8s, tc.query)
			if w.Code != http.StatusBadRequest {
				t.Errorf("query %q: status = %d, want 400 (body: %s)", tc.query, w.Code, w.Body.String())
			}
		})
	}
}

// TestListAgents_LabelFilterMatchesMetadataLabels guards the pre-pagination
// behavior where `label` was pushed down to the K8s list call and so matched
// the CR's metadata labels — including operator-set ones like
// komputer.ai/office that never appear in Spec.Labels.
func TestListAgents_LabelFilterMatchesMetadataLabels(t *testing.T) {
	a := newTestAgent("in-office", "default")
	a.Labels = map[string]string{"komputer.ai/office": "mgr-office"}
	b := newTestAgent("elsewhere", "default")
	k8s := newFakeK8s(t, a, b)

	for _, q := range []string{"?label=komputer.ai/office=mgr-office", "?label=komputer.ai/office=mgr-office&limit=10"} {
		w := serveListAgents(t, k8s, q)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, body = %s", q, w.Code, w.Body.String())
		}
		var resp AgentListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(resp.Agents) != 1 || resp.Agents[0].Name != "in-office" {
			t.Errorf("%s: got %+v, want only in-office", q, resp.Agents)
		}
	}
}
