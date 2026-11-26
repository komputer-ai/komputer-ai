package main

import (
	"os"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// startEnvtest runs a real API server with the chart's CRDs. CI installs the
// binaries (setup-envtest) and sets KUBEBUILDER_ASSETS.
func startEnvtest(t *testing.T) *rest.Config {
	t.Helper()
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("KUBEBUILDER_ASSETS not set; run: export KUBEBUILDER_ASSETS=$(setup-envtest use -p path)")
	}
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{"../helm/komputer-ai/crds"},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	t.Cleanup(func() { _ = env.Stop() })
	return cfg
}

func newEnvtestK8sClient(t *testing.T, cfg *rest.Config) *K8sClient {
	t.Helper()
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(komputerv1alpha1.AddToScheme(scheme))
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return &K8sClient{client: c, restConfig: cfg, defaultNamespace: "default"}
}

func TestListCacheServesAgentLists(t *testing.T) {
	cfg := startEnvtest(t)
	k := newEnvtestK8sClient(t, cfg)
	ctx := t.Context()

	k.StartCache(ctx, 30*time.Second)
	if k.cache.Load() == nil {
		t.Fatal("cache did not sync")
	}

	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "cached-agent", Namespace: "default"},
	}
	agent.Spec.Instructions = "do the work"
	if err := k.client.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	// The watch delivers the new agent to the cache shortly after the write.
	deadline := time.Now().Add(5 * time.Second)
	for {
		agents, err := k.ListAgentSummaries(ctx, "", nil)
		if err != nil {
			t.Fatalf("list agent summaries: %v", err)
		}
		if len(agents) == 1 && agents[0].Name == "cached-agent" {
			if agents[0].ManagedFields != nil {
				t.Error("cached agents should have managedFields stripped")
			}
			if agents[0].Spec.Instructions != "" {
				t.Error("cached agents should have instructions stripped")
			}
			full, err := k.ListAgents(ctx, "", nil)
			if err != nil {
				t.Fatalf("list agents: %v", err)
			}
			if len(full) != 1 || full[0].Spec.Instructions != "do the work" {
				t.Errorf("ListAgents should return full agents from the API server, got %+v", full)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("cache never saw the new agent; got %d agents", len(agents))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestListCacheFallsBackWhenWatchIsForbidden(t *testing.T) {
	cfg := startEnvtest(t)
	// A user with no RBAC can't list or watch, so the informers never sync.
	noRBAC := rest.CopyConfig(cfg)
	noRBAC.Impersonate = rest.ImpersonationConfig{UserName: "no-rbac"}
	k := newEnvtestK8sClient(t, noRBAC)

	k.StartCache(t.Context(), 2*time.Second)
	if k.cache.Load() != nil {
		t.Fatal("cache should stay disabled when informers can't sync")
	}
	if k.reader() != client.Reader(k.client) {
		t.Fatal("reads should fall back to the direct client")
	}
}
