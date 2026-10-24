package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// newGetCountingK8s counts the single-object reads (Gets) the API server sees.
func newGetCountingK8s(t *testing.T, gets *int, objs ...client.Object) *K8sClient {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := komputerv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				*gets++
				return c.Get(ctx, key, obj, opts...)
			},
		}).Build()
	return &K8sClient{client: c, defaultNamespace: "default"}
}

// Listing a page of agents mustn't read every agent's Secrets and template:
// those reads go to the API server one by one, so a page load grew with the
// total number of agents.
func TestListAgents_PageReadsDontGrowWithAgentCount(t *testing.T) {
	objs := []client.Object{
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "default"}, Data: map[string][]byte{"API_KEY": []byte("x")}},
		&komputerv1alpha1.KomputerAgentTemplate{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
			Spec: komputerv1alpha1.KomputerAgentTemplateSpec{BudgetUSD: "1.00"}},
	}
	for i := 0; i < 40; i++ {
		a := newTestAgent(fmt.Sprintf("agent-%02d", i), "default")
		a.Spec.Secrets = []string{"shared"}
		a.Status.TotalCostUSD = "2.00" // over the template's $1 budget
		objs = append(objs, a)
	}
	var gets int
	k8s := newGetCountingK8s(t, &gets, objs...)

	w := serveListAgents(t, k8s, "?limit=5&sort=name")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp AgentListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Agents) != 5 || resp.Total != 40 {
		t.Fatalf("page = %d agents of %d, want 5 of 40", len(resp.Agents), resp.Total)
	}
	// The page still carries what those reads are for.
	for _, a := range resp.Agents {
		if len(a.Secrets) != 1 || a.Secrets[0] != "API_KEY" || !a.BudgetExhausted {
			t.Errorf("%s: secrets %v, budgetExhausted %v; want [API_KEY], true", a.Name, a.Secrets, a.BudgetExhausted)
		}
	}
	// One read of the shared Secret and one of the shared template, not one per agent.
	if gets > 2 {
		t.Errorf("listing a 5-agent page of 40 made %d Get calls, want at most 2", gets)
	}
}
