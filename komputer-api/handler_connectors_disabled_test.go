package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestUpdateConnectorDisabledTogglesSpec verifies PATCH .../connectors/:name with
// {"disabled": true} flips KomputerConnectorSpec.Disabled and reflects it in the
// response, without requiring a token.
func TestUpdateConnectorDisabledTogglesSpec(t *testing.T) {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "github", URL: "https://mcp.example.com"},
	}
	k8s := newFakeK8s(t, conn)

	w := patchConnector(t, k8s, "github", `{"disabled":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp ConnectorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Disabled {
		t.Error("response Disabled = false, want true")
	}

	got := &komputerv1alpha1.KomputerConnector{}
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "github", Namespace: "default"}, got); err != nil {
		t.Fatal(err)
	}
	if !got.Spec.Disabled {
		t.Error("CR Spec.Disabled = false, want true")
	}
}

// TestUpdateConnectorCanReEnable verifies disabled -> enabled works, from any caller
// (no restriction on who re-enables).
func TestUpdateConnectorCanReEnable(t *testing.T) {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "github", URL: "https://mcp.example.com", Disabled: true},
	}
	k8s := newFakeK8s(t, conn)

	w := patchConnector(t, k8s, "github", `{"disabled":false}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp ConnectorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Disabled {
		t.Error("response Disabled = true, want false after re-enabling")
	}
}

// TestUpdateConnectorDisabledWorksForOAuth verifies that toggling `disabled` succeeds
// even for an OAuth connector, which rejects token updates.
func TestUpdateConnectorDisabledWorksForOAuth(t *testing.T) {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "notion", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "notion", URL: "https://mcp.example.com", AuthType: "oauth"},
	}
	k8s := newFakeK8s(t, conn)

	w := patchConnector(t, k8s, "notion", `{"disabled":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp ConnectorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Disabled {
		t.Error("response Disabled = false, want true")
	}

	// Token updates on the same OAuth connector must still be rejected.
	w = patchConnector(t, k8s, "notion", `{"token":"x"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("OAuth token update: status = %d, want 400", w.Code)
	}
}

// TestUpdateConnectorRequiresTokenOrDisabled verifies the body must set at least one
// of the two updatable fields.
func TestUpdateConnectorRequiresTokenOrDisabled(t *testing.T) {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "github", URL: "https://mcp.example.com"},
	}
	k8s := newFakeK8s(t, conn)

	w := patchConnector(t, k8s, "github", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestUpdateConnectorTokenStillWorksWithoutDisabled is the control case — a
// token-only PATCH (no `disabled` field) must behave exactly as before.
func TestUpdateConnectorTokenStillWorksWithoutDisabled(t *testing.T) {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "jira", Namespace: "default", UID: "uid-1"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "jira", URL: "https://mcp.example.com", AuthType: "none"},
	}
	k8s := newFakeK8s(t, conn)

	w := patchConnector(t, k8s, "jira", `{"token":"brand-new"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp ConnectorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Disabled {
		t.Error("Disabled must default to false when not set in the request")
	}
}
