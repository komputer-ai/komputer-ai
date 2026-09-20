package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func newFakeK8s(t *testing.T, objs ...client.Object) *K8sClient {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := komputerv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
	return &K8sClient{client: c, defaultNamespace: "default"}
}

func patchConnector(t *testing.T, k8s *K8sClient, name, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/connectors/:name", updateConnector(k8s))
	req := httptest.NewRequest(http.MethodPatch, "/connectors/"+name, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdateConnectorTokenRewritesOnlyReferencedKey(t *testing.T) {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerConnectorSpec{
			Service:  "github",
			URL:      "https://mcp.example.com",
			AuthType: "token",
			AuthSecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "shared-creds"},
				Key:                  "gh",
			},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-creds", Namespace: "default"},
		Data:       map[string][]byte{"gh": []byte("old"), "other": []byte("keep-me")},
	}
	k8s := newFakeK8s(t, conn, secret)

	w := patchConnector(t, k8s, "github", `{"token":"new-token"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	got := &corev1.Secret{}
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "shared-creds", Namespace: "default"}, got); err != nil {
		t.Fatal(err)
	}
	if string(got.Data["gh"]) != "new-token" {
		t.Errorf("gh = %q, want new-token", got.Data["gh"])
	}
	// A connector may point at one key of a multi-key secret; the others must survive.
	if string(got.Data["other"]) != "keep-me" {
		t.Errorf("other = %q, want keep-me (unrelated keys must not be clobbered)", got.Data["other"])
	}
}

func TestUpdateConnectorTokenCreatesSecretWhenMissing(t *testing.T) {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "jira", Namespace: "default", UID: "uid-1"},
		Spec: komputerv1alpha1.KomputerConnectorSpec{
			Service:  "jira",
			URL:      "https://mcp.example.com",
			AuthType: "none",
		},
	}
	k8s := newFakeK8s(t, conn)

	w := patchConnector(t, k8s, "jira", `{"token":"brand-new"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp ConnectorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.AuthSecretName != "jira-credentials" || resp.AuthSecretKey != "token" {
		t.Errorf("response secret ref = %s/%s, want jira-credentials/token", resp.AuthSecretName, resp.AuthSecretKey)
	}
	if resp.AuthType != "token" {
		t.Errorf("authType = %q, want token", resp.AuthType)
	}

	secret := &corev1.Secret{}
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "jira-credentials", Namespace: "default"}, secret); err != nil {
		t.Fatalf("secret not created: %v", err)
	}
	if string(secret.Data["token"]) != "brand-new" {
		t.Errorf("token = %q, want brand-new", secret.Data["token"])
	}
	if len(secret.OwnerReferences) != 1 || secret.OwnerReferences[0].Name != "jira" {
		t.Errorf("owner refs = %+v, want one pointing at the connector", secret.OwnerReferences)
	}

	updated := &komputerv1alpha1.KomputerConnector{}
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "jira", Namespace: "default"}, updated); err != nil {
		t.Fatal(err)
	}
	if updated.Spec.AuthSecretKeyRef == nil || updated.Spec.AuthSecretKeyRef.Name != "jira-credentials" {
		t.Errorf("CR authSecretKeyRef = %+v, want jira-credentials", updated.Spec.AuthSecretKeyRef)
	}
}

func TestUpdateConnectorTokenKeepsHeaderAuthType(t *testing.T) {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "custom", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerConnectorSpec{
			Service:    "custom",
			URL:        "https://mcp.example.com",
			AuthType:   "header",
			HeaderName: "X-API-Key",
		},
	}
	k8s := newFakeK8s(t, conn)

	w := patchConnector(t, k8s, "custom", `{"token":"k"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp ConnectorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.AuthType != "header" {
		t.Errorf("authType = %q, want header preserved", resp.AuthType)
	}
}

func TestUpdateConnectorTokenRejectsOAuth(t *testing.T) {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "notion", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerConnectorSpec{
			Service:  "notion",
			URL:      "https://mcp.example.com",
			AuthType: "oauth",
		},
	}
	k8s := newFakeK8s(t, conn)

	w := patchConnector(t, k8s, "notion", `{"token":"x"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestUpdateConnectorTokenValidation(t *testing.T) {
	k8s := newFakeK8s(t)

	if w := patchConnector(t, k8s, "nope", `{"token":"x"}`); w.Code != http.StatusNotFound {
		t.Errorf("unknown connector: status = %d, want 404", w.Code)
	}
	if w := patchConnector(t, k8s, "nope", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("missing token: status = %d, want 400", w.Code)
	}
}
