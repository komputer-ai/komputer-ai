package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRegistriesAreSeparate(t *testing.T) {
	apiReg, agentReg := newMetricsRegistries(false)
	if apiReg == agentReg {
		t.Fatal("api and agent registries must be different instances")
	}
	// Verify the api registry is independently gatherable (not the global default).
	apiMetrics, err := apiReg.Gather()
	if err != nil {
		t.Fatalf("api gather: %v", err)
	}
	if len(apiMetrics) == 0 {
		t.Errorf("expected api registry to have metrics, got 0")
	}
	agentMetrics, err := agentReg.Gather()
	if err != nil {
		t.Fatalf("agent gather: %v", err)
	}
	if len(agentMetrics) == 0 {
		t.Errorf("expected agent registry to have metrics, got 0")
	}
}

func TestPerAgentLabelEnabled(t *testing.T) {
	_, _ = newMetricsRegistries(true)
	if !perAgentLabelsEnabled {
		t.Errorf("expected perAgentLabels=true after construction with true")
	}
}

func TestPerAgentLabelDisabled(t *testing.T) {
	_, _ = newMetricsRegistries(false)
	if perAgentLabelsEnabled {
		t.Errorf("expected perAgentLabels=false after construction with false")
	}
}

func TestHTTPMiddlewareIncrementsCounter(t *testing.T) {
	_, _ = newMetricsRegistries(false)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(metricsMiddleware())
	r.GET("/foo", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

	req := httptest.NewRequest("GET", "/foo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	count := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/foo", "200"))
	if count != 1 {
		t.Errorf("expected 1 request counted, got %v", count)
	}
}

func TestHTTPMiddlewareObservesDuration(t *testing.T) {
	_, _ = newMetricsRegistries(false)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(metricsMiddleware())
	r.GET("/bar", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

	req := httptest.NewRequest("GET", "/bar", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Histogram count should be 1 after one request
	count := testutil.CollectAndCount(httpRequestDuration)
	if count == 0 {
		t.Errorf("expected histogram to have observations")
	}
}

func TestCRCollectorEmitsExpectedDescriptors(t *testing.T) {
	c := newCRCollector(nil) // k8s nil — Collect will silently skip due to error returns
	ch := make(chan *prometheus.Desc, 3)
	c.Describe(ch)
	close(ch)
	descs := []*prometheus.Desc{}
	for d := range ch {
		descs = append(descs, d)
	}
	if len(descs) != 3 {
		t.Errorf("expected 3 descriptors, got %d", len(descs))
	}
}

func TestToolCallTracker(t *testing.T) {
	toolCallTracker = &toolCallTrackerT{starts: make(map[string]time.Time)}
	start := time.Now()
	toolCallTracker.markStart("foo-agent", "tc_01", start)
	dur, ok := toolCallTracker.consumeDuration("foo-agent", "tc_01", start.Add(2*time.Second))
	if !ok {
		t.Fatal("expected to find start time")
	}
	if dur != 2*time.Second {
		t.Errorf("expected 2s, got %v", dur)
	}
	// Second consume should fail (already consumed).
	if _, ok := toolCallTracker.consumeDuration("foo-agent", "tc_01", time.Now()); ok {
		t.Error("expected second consume to fail")
	}
}

// A cancelled task (manual or taskTimeout) still spent money; its cost must reach
// the cost metric the same way a completed task's does, or Grafana spend
// under-counts and disagrees with status.totalCostUSD.
func TestRecordTaskCostMetric_CountsCompletedAndCancelled(t *testing.T) {
	ns, model, agent := "metrics-cost-ns", "claude-opus-5-5", ""
	before := testutil.ToFloat64(agentTaskCostUSD.WithLabelValues(ns, model, agent))

	recordTaskCostMetric(ns, model, agent, map[string]interface{}{"cost_usd": 0.25})               // completed
	recordTaskCostMetric(ns, model, agent, map[string]interface{}{"cost_usd": 0.5, "reason": "x"}) // cancelled
	recordTaskCostMetric(ns, model, agent, map[string]interface{}{"reason": "no result"})          // no cost: no-op

	got := testutil.ToFloat64(agentTaskCostUSD.WithLabelValues(ns, model, agent)) - before
	if got != 0.75 {
		t.Errorf("cost metric delta = %v, want 0.75", got)
	}
}
