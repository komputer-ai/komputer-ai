package main

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Populated at build time via -ldflags "-X main.version=... -X main.commit=..."
var (
	version = "dev"
	commit  = "unknown"
)

// perAgentLabelsEnabled controls whether agent_name appears as a real value or ""
// on per-agent metrics. Set once at startup.
var perAgentLabelsEnabled bool

// Two separate registries — kept apart so /api/metrics and /agent/metrics
// can be scraped by different ServiceMonitors with different retention/cardinality
// budgets if the operator wants.
var (
	apiRegistry   *prometheus.Registry
	agentRegistry *prometheus.Registry
)

// Per-registry metric handles. Initialized eagerly so they are always non-nil.
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "komputer_api_http_requests_total",
			Help: "Total HTTP requests received by the API.",
		},
		[]string{"method", "path", "status"},
	)
	agentTasksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "komputer_agent_tasks_total",
			Help: "Total agent task lifecycle transitions.",
		},
		[]string{"namespace", "model", "outcome", "agent_name"},
	)

	agentTaskDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "komputer_agent_task_duration_seconds",
			Help:    "Wall-clock duration of completed agent tasks.",
			Buckets: prometheus.ExponentialBuckets(1, 2, 12), // 1s, 2s, ... ~1h
		},
		[]string{"namespace", "model", "agent_name"},
	)

	agentTaskCostUSD = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "komputer_agent_task_cost_usd_total",
			Help: "Total cost in USD across all completed agent tasks.",
		},
		[]string{"namespace", "model", "agent_name"},
	)

	agentTaskTokens = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "komputer_agent_task_tokens_total",
			Help: "Total tokens used across all completed agent tasks.",
		},
		[]string{"namespace", "model", "kind", "agent_name"},
	)

	agentToolInvocations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "komputer_agent_tool_invocations_total",
			Help: "Total tool invocations by tool name and outcome.",
		},
		[]string{"namespace", "tool", "outcome", "agent_name"},
	)

	agentToolDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "komputer_agent_tool_duration_seconds",
			Help:    "Tool execution duration, derived from tool_call/tool_result event pairs.",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 12), // 100ms, 200ms, ... ~7min
		},
		[]string{"namespace", "tool", "agent_name"},
	)

	agentActionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "komputer_agent_actions_total",
			Help: "Agent management actions taken via the API (create/delete/cancel/sleep/wake/patch).",
		},
		[]string{"action", "result"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "komputer_api_http_request_duration_seconds",
			Help:    "Wall-clock duration of HTTP requests handled by the API.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	wsConnectionsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "komputer_api_ws_connections_active",
			Help: "Currently open WebSocket connections to /agents/:name/ws.",
		},
		[]string{"mode"}, // broadcast or group
	)

	wsDispatchTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "komputer_api_ws_dispatch_total",
			Help: "Events dispatched to WebSocket clients.",
		},
		[]string{"mode", "result"}, // mode=broadcast|group, result=delivered|claimed_by_other|write_failed
	)

	redisXreadMessagesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "komputer_api_redis_xread_messages_total",
			Help: "Total messages read from Redis streams by the broadcast worker.",
		},
	)

	// Build-info gauges (value=1) give each registry at least one active metric
	// and expose version information to dashboards.
	apiBuildInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "komputer_api_build_info",
			Help: "Always 1; exposes build metadata labels for dashboards.",
		},
		[]string{"version", "commit"},
	)
	agentBuildInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "komputer_agent_build_info",
			Help: "Always 1; exposes build metadata labels for dashboards.",
		},
		[]string{"version", "commit"},
	)
	crCollectorErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "komputer_api_cr_collector_errors_total",
			Help: "Failed CR list calls during /agent/metrics scrape, by resource.",
		},
		[]string{"resource"},
	)
)

// newMetricsRegistries creates fresh registries and registers all metric handles.
// Called once from SetupRoutes. The perAgentLabels flag controls whether
// agent_name appears as a real value or as the empty string on per-agent metrics.
func newMetricsRegistries(perAgentLabels bool) (*prometheus.Registry, *prometheus.Registry) {
	perAgentLabelsEnabled = perAgentLabels

	apiRegistry = prometheus.NewRegistry()
	agentRegistry = prometheus.NewRegistry()

	apiRegistry.MustRegister(httpRequestsTotal)
	apiRegistry.MustRegister(httpRequestDuration)
	apiRegistry.MustRegister(wsConnectionsActive)
	apiRegistry.MustRegister(wsDispatchTotal)
	apiRegistry.MustRegister(redisXreadMessagesTotal)
	apiRegistry.MustRegister(apiBuildInfo)
	agentRegistry.MustRegister(agentTasksTotal)
	agentRegistry.MustRegister(agentTaskDuration)
	agentRegistry.MustRegister(agentTaskCostUSD)
	agentRegistry.MustRegister(agentTaskTokens)
	agentRegistry.MustRegister(agentToolInvocations)
	agentRegistry.MustRegister(agentToolDuration)
	agentRegistry.MustRegister(agentActionsTotal)
	agentRegistry.MustRegister(agentBuildInfo)
	agentRegistry.MustRegister(crCollectorErrorsTotal)

	apiBuildInfo.WithLabelValues(version, commit).Set(1)
	agentBuildInfo.WithLabelValues(version, commit).Set(1)

	return apiRegistry, agentRegistry
}

// agentNameLabel returns the agent name when perAgentLabels is enabled, "" otherwise.
// Always include this in the label set on per-agent metrics so dashboards stay schema-stable.
func agentNameLabel(name string) string {
	if perAgentLabelsEnabled {
		return name
	}
	return ""
}

// toolCallTrackerT correlates tool_call → tool_result events to compute tool execution duration.
type toolCallTrackerT struct {
	mu     sync.Mutex
	starts map[string]time.Time // key = "<agent>:<tool_use_id>"
}

var toolCallTracker = &toolCallTrackerT{starts: make(map[string]time.Time)}

func (t *toolCallTrackerT) markStart(agent, toolUseID string, at time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.starts[agent+":"+toolUseID] = at
}

// consumeDuration returns the time between markStart and now, deleting the entry. Returns false if no start was tracked.
func (t *toolCallTrackerT) consumeDuration(agent, toolUseID string, endAt time.Time) (time.Duration, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := agent + ":" + toolUseID
	startAt, ok := t.starts[key]
	if !ok {
		return 0, false
	}
	delete(t.starts, key)
	return endAt.Sub(startAt), true
}

// crCollector lists KomputerAgent and KomputerSchedule CRs at scrape time
// and exposes counts as gauges. No goroutines, no cache drift.
// Implements prometheus.Collector.
type crCollector struct {
	k8s *K8sClient

	tasksInProgress *prometheus.Desc
	schedulesActive *prometheus.Desc
	agentsByPhase   *prometheus.Desc
}

func newCRCollector(k8s *K8sClient) *crCollector {
	return &crCollector{
		k8s: k8s,
		tasksInProgress: prometheus.NewDesc(
			"komputer_tasks_inprogress",
			"Number of agent tasks currently in progress (taskStatus=InProgress).",
			// agent_name is always present so dashboards stay schema-stable.
			// Empty string when KOMPUTER_METRICS_PER_AGENT=false, real name when true.
			[]string{"namespace", "model", "agent_name"}, nil,
		),
		schedulesActive: prometheus.NewDesc(
			"komputer_schedules_active",
			"Number of KomputerSchedule resources defined.",
			[]string{"namespace"}, nil,
		),
		agentsByPhase: prometheus.NewDesc(
			"komputer_agents_active",
			"Number of agents in each lifecycle phase.",
			[]string{"namespace", "phase"}, nil,
		),
	}
}

func (c *crCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.tasksInProgress
	ch <- c.schedulesActive
	ch <- c.agentsByPhase
}

func (c *crCollector) Collect(ch chan<- prometheus.Metric) {
	// 5s timeout: scrapes that take longer get truncated. Caller (Prometheus) typically
	// gives us 10s before giving up entirely, so this leaves headroom.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var agents komputerv1alpha1.KomputerAgentList
	if err := c.k8s.client.List(ctx, &agents, &client.ListOptions{LabelSelector: labels.Everything()}); err != nil {
		log.Printf("crCollector: failed to list KomputerAgents: %v", err)
		crCollectorErrorsTotal.WithLabelValues("agents").Inc()
	} else {
		// tasksInProgress: when perAgentLabels=true, one series per agent (value=1).
		// When false, aggregated by (namespace, model) with agent_name="" (sum is the count).
		inProg := map[[3]string]int{} // (namespace, model, agent_name) -> count
		// agentsByPhase: always aggregated count per (namespace, phase).
		byPhase := map[[2]string]int{} // (namespace, phase) -> count
		for _, a := range agents.Items {
			byPhase[[2]string{a.Namespace, string(a.Status.Phase)}]++
			if a.Status.TaskStatus == "InProgress" {
				inProg[[3]string{a.Namespace, a.Spec.Model, agentNameLabel(a.Name)}]++
			}
		}
		for k, v := range inProg {
			ch <- prometheus.MustNewConstMetric(c.tasksInProgress, prometheus.GaugeValue, float64(v), k[0], k[1], k[2])
		}
		for k, v := range byPhase {
			ch <- prometheus.MustNewConstMetric(c.agentsByPhase, prometheus.GaugeValue, float64(v), k[0], k[1])
		}
	}

	var schedules komputerv1alpha1.KomputerScheduleList
	if err := c.k8s.client.List(ctx, &schedules); err != nil {
		log.Printf("crCollector: failed to list KomputerSchedules: %v", err)
		crCollectorErrorsTotal.WithLabelValues("schedules").Inc()
	} else {
		byNs := map[string]int{}
		for _, s := range schedules.Items {
			byNs[s.Namespace]++
		}
		for ns, v := range byNs {
			ch <- prometheus.MustNewConstMetric(c.schedulesActive, prometheus.GaugeValue, float64(v), ns)
		}
	}
}


// metricsMiddleware records HTTP request count and duration for every handled request.
// Path is the route template (e.g. "/agents/:name") so cardinality stays bounded.
func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
	}
}
