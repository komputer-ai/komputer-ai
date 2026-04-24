// Package metrics holds the operator's custom Prometheus metrics.
// They register on controller-runtime's global metrics registry, so they
// appear at the same /metrics endpoint as the built-in reconcile metrics.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var TemplateCapReachedTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "komputer_operator_template_cap_reached_total",
		Help: "Number of times agent admission was denied because the template's maxConcurrentAgents was reached.",
	},
	[]string{"namespace", "template"},
)

func init() {
	ctrlmetrics.Registry.MustRegister(TemplateCapReachedTotal)
}
