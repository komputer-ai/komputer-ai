package main

import (
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// lastSpecUpdate returns the most recent time any managedFields entry edited
// the resource's spec/metadata, as opposed to its status. Memories, skills,
// and connectors all have a status subresource that the operator writes
// continuously (status.attachedAgents / status.agentNames) — those writes
// must not look like an edit to the resource itself, so "updatedAt" in API
// responses is derived from this instead of a plain LastUpdateTime field.
//
// Falls back to CreationTimestamp if no qualifying managedFields entry
// exists (e.g. none were recorded at all, as with the fake client used in
// unit tests).
func lastSpecUpdate(obj metav1.Object) time.Time {
	latest := obj.GetCreationTimestamp().Time
	for _, mf := range obj.GetManagedFields() {
		// The common case: with +kubebuilder:subresource:status, a write to
		// .status goes through the status subresource and is recorded here
		// with Subresource == "status". Skip those outright.
		if mf.Subresource == "status" {
			continue
		}
		if mf.FieldsV1 == nil || fieldsTouchOnlyStatus(mf.FieldsV1.Raw) {
			continue
		}
		if mf.Time != nil && mf.Time.Time.After(latest) {
			latest = mf.Time.Time
		}
	}
	return latest
}

// fieldsTouchOnlyStatus reports whether a managedFields entry's FieldsV1
// payload touches only the top-level "f:status" key. This is a
// belt-and-suspenders check for an entry with Subresource == "" whose
// payload is nonetheless status-only — e.g. a CRD without a status
// subresource, or a client that wrote status through the main resource
// rather than the status subresource.
//
// We only treat the exact {"f:status": ...} shape as status-only. An entry
// that also touches "f:metadata" (labels/annotations, etc.) is conservatively
// treated as a real edit rather than guessed to be pure bookkeeping, since
// metadata changes (e.g. a renamed label) can be a legitimate edit to the
// resource that a user would expect to bump "updatedAt".
func fieldsTouchOnlyStatus(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || len(fields) == 0 {
		return false
	}
	for key := range fields {
		if key != "f:status" {
			return false
		}
	}
	return true
}
