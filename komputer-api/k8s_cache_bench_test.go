package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestListCacheAtScale measures the list cache against a real API server with
// many agents: memory, first-sync time, and GET /agents latency and size with
// and without the cache. It's a measurement, not a check, so it only runs when
// asked:
//
//	KOMPUTER_CACHE_BENCH=1000,5000,10000 go test -run TestListCacheAtScale -v -timeout 60m .
func TestListCacheAtScale(t *testing.T) {
	raw := os.Getenv("KOMPUTER_CACHE_BENCH")
	if raw == "" {
		t.Skip("set KOMPUTER_CACHE_BENCH=1000,5000,10000 to run")
	}
	var sizes []int
	for _, s := range strings.Split(raw, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			t.Fatalf("bad size %q", s)
		}
		sizes = append(sizes, n)
	}
	sort.Ints(sizes)

	cfg := startEnvtest(t)
	cfg.QPS = -1 // as in production (ctrl.GetConfig): no client-side throttling
	k := newEnvtestK8sClient(t, cfg)
	ctx := t.Context()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/agents", listAgents(k))

	sample := benchAgent(0)
	b, _ := json.Marshal(sample)
	fmt.Printf("agent object size: %d bytes (instructions %d, systemPrompt %d, internalSystemPrompt %d)\n",
		len(b), len(sample.Spec.Instructions), len(sample.Spec.SystemPrompt), len(sample.Spec.InternalSystemPrompt))

	const dashboard = "?fields=status,taskStatus,totalCostUSD,lastTaskMessage,createdAt"
	created := 0
	for _, n := range sizes {
		start := time.Now()
		createBenchAgents(t, ctx, k, created, n)
		created = n
		fmt.Printf("\n=== %d agents (created in %s)\n", n, time.Since(start).Round(time.Second))

		// Before the cache: every request lists full agents from the API server.
		k.cache.Store(nil)
		lat, size := timeGet(t, r, "")
		fmt.Printf("no cache,   GET /agents (full)        %8s  %8s\n", lat, kb(size))
		lat, size = timeGet(t, r, dashboard)
		fmt.Printf("no cache,   GET /agents?fields=…      %8s  %8s\n", lat, kb(size))

		// Cache that keeps whole agents (managedFields stripped only).
		mem, sync := measureCache(t, ctx, k, func(cctx context.Context) {
			c, err := cache.New(k.restConfig, cache.Options{Scheme: k.client.Scheme(), DefaultTransform: cache.TransformStripManagedFields()})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.GetInformer(cctx, &komputerv1alpha1.KomputerAgent{}); err != nil {
				t.Fatal(err)
			}
			go func() { _ = c.Start(cctx) }()
			if !c.WaitForCacheSync(cctx) {
				t.Fatal("full cache did not sync")
			}
		})
		fmt.Printf("full cache:     heap %8s  first sync %8s\n", mb(mem), sync)

		// The cache as shipped: heavy agent fields stripped.
		var shipped context.CancelFunc
		mem, sync = measureCache(t, ctx, k, func(cctx context.Context) {
			k.StartCache(cctx, 10*time.Minute)
			if k.cache.Load() == nil {
				t.Fatal("cache did not sync")
			}
		}, &shipped)
		fmt.Printf("stripped cache: heap %8s  first sync %8s\n", mb(mem), sync)
		lat, size = timeGet(t, r, dashboard)
		fmt.Printf("cache,      GET /agents?fields=…      %8s  %8s\n", lat, kb(size))
		lat, size = timeGet(t, r, "")
		fmt.Printf("cache,      GET /agents (full)        %8s  %8s  (reads the API server)\n", lat, kb(size))
		lat, size = timeGet(t, r, "?limit=50&sort=-cost"+strings.Replace(dashboard, "?", "&", 1))
		fmt.Printf("cache,      GET /agents page of 50    %8s  %8s\n", lat, kb(size))
		shipped()
		k.cache.Store(nil)
	}
}

func benchAgent(i int) *komputerv1alpha1.KomputerAgent {
	a := &komputerv1alpha1.KomputerAgent{ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprintf("agent-%05d", i),
		Namespace: "default",
		Labels:    map[string]string{"team": fmt.Sprintf("team-%d", i%7), "komputer.ai/office": fmt.Sprintf("office-%d", i%20)},
	}}
	// Rough sizes: a task of a few paragraphs, a short custom prompt, and the
	// worker role prompt (~2.3KB) plus a memory or two.
	a.Spec.Instructions = strings.Repeat("Investigate the failing job and open a PR with a fix. ", 28) // ~1.5KB
	a.Spec.SystemPrompt = strings.Repeat("Be concise. ", 40)                                           // ~0.5KB
	a.Spec.InternalSystemPrompt = strings.Repeat("You are a komputer worker agent. ", 110)             // ~3.6KB
	a.Spec.Model = "claude-sonnet-4-6"
	a.Spec.Secrets = []string{"github-token", "slack-token"}
	a.Spec.Labels = map[string]string{"team": fmt.Sprintf("team-%d", i%7)}
	env := make([]corev1.EnvVar, 5)
	for j := range env {
		env[j] = corev1.EnvVar{Name: fmt.Sprintf("SETTING_%d", j), Value: "some-value"}
	}
	a.Spec.PodSpec = &corev1.PodSpec{Containers: []corev1.Container{{
		Name: "agent", Image: "ghcr.io/komputer-ai/komputer-agent:v0.25.0", Env: env,
		Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("2Gi")}},
	}}}
	return a
}

func createBenchAgents(t *testing.T, ctx context.Context, k *K8sClient, from, to int) {
	t.Helper()
	jobs := make(chan int)
	var wg sync.WaitGroup
	var once sync.Once
	for w := 0; w < 64; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				a := benchAgent(i)
				if err := k.client.Create(ctx, a); err != nil {
					once.Do(func() { t.Errorf("create %s: %v", a.Name, err) })
					continue
				}
				a.Status.Phase = []komputerv1alpha1.KomputerAgentPhase{"Sleeping", "Running", "Archived", "Sleeping"}[i%4]
				a.Status.TaskStatus = "Complete"
				a.Status.TotalCostUSD = fmt.Sprintf("%d.%04d", i%40, i%10000)
				a.Status.LastTaskCostUSD = "0.0421"
				a.Status.LastTaskMessage = strings.Repeat("Opened the PR and the checks pass. ", 12) // ~0.4KB
				if err := k.client.Status().Update(ctx, a); err != nil {
					once.Do(func() { t.Errorf("status %s: %v", a.Name, err) })
				}
			}
		}()
	}
	for i := from; i < to; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	if t.Failed() {
		t.FailNow()
	}
}

// measureCache runs start (which must leave a synced cache running on the
// context it's given) and reports the heap it added and how long it took.
// The cache is stopped afterwards unless keep is passed, which receives the stop.
func measureCache(t *testing.T, ctx context.Context, k *K8sClient, start func(context.Context), keep ...*context.CancelFunc) (uint64, time.Duration) {
	t.Helper()
	before := heapAfterGC()
	cctx, stop := context.WithCancel(ctx)
	began := time.Now()
	start(cctx)
	took := time.Since(began)
	after := heapAfterGC()
	if len(keep) > 0 {
		*keep[0] = stop
	} else {
		stop()
		time.Sleep(time.Second) // let the informers exit before the next measurement
	}
	if after < before {
		return 0, took
	}
	return after - before, took.Round(time.Millisecond)
}

func heapAfterGC() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// timeGet returns the median latency of 5 GETs and the response size.
func timeGet(t *testing.T, r *gin.Engine, query string) (time.Duration, int) {
	t.Helper()
	var lats []time.Duration
	size := 0
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		began := time.Now()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/agents"+query, nil))
		lats = append(lats, time.Since(began))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /agents%s: %d %s", query, w.Code, w.Body.String())
		}
		size = w.Body.Len()
	}
	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })
	return lats[2].Round(100 * time.Microsecond), size
}

func kb(n int) string    { return fmt.Sprintf("%.0fKB", float64(n)/1024) }
func mb(n uint64) string { return fmt.Sprintf("%.1fMB", float64(n)/(1<<20)) }
