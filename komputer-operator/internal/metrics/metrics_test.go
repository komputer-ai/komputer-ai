package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestTemplateCapReachedRegistered(t *testing.T) {
	TemplateCapReachedTotal.WithLabelValues("default", "small").Inc()
	count := testutil.ToFloat64(TemplateCapReachedTotal.WithLabelValues("default", "small"))
	if count < 1 {
		t.Errorf("expected at least 1, got %v", count)
	}
}
