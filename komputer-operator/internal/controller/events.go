package controller

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// recordEvent emits a Normal Event on the agent for a TTL or lifecycle
// transition, with the same reason its condition records. A nil recorder (unit
// tests that don't care) is a no-op. Events also outlive a deleted agent, which
// is the only trace deleteTTL leaves.
func recordEvent(rec record.EventRecorder, obj runtime.Object, reason, message string) {
	if rec == nil {
		return
	}
	rec.Event(obj, corev1.EventTypeNormal, reason, message)
}

func deleteTTLMessage(deleteTTL *metav1.Duration) string {
	return fmt.Sprintf("Deleted — deleteTTL (%s) elapsed since creation.", deleteTTL.Duration)
}

func taskTimedOutMessage(taskTimeout *metav1.Duration) string {
	return fmt.Sprintf("Task cancelled — it ran longer than taskTimeout (%s).", taskTimeout.Duration)
}
