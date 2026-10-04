package main

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLastSpecUpdate(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	specEditOlder := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	specEditNewer := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	statusWriteNewest := time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		obj  *metav1.ObjectMeta
		want time.Time
	}{
		{
			name: "spec edit newer than status write wins",
			obj: &metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(created),
				ManagedFields: []metav1.ManagedFieldsEntry{
					{
						Subresource: "",
						Time:        timePtr(specEditNewer),
						FieldsV1:    rawFields(`{"f:spec":{"f:content":{}}}`),
					},
					{
						Subresource: "status",
						Time:        timePtr(statusWriteNewest),
						FieldsV1:    rawFields(`{"f:status":{"f:attachedAgents":{}}}`),
					},
				},
			},
			want: specEditNewer,
		},
		{
			name: "only status writes falls back to creation time",
			obj: &metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(created),
				ManagedFields: []metav1.ManagedFieldsEntry{
					{
						Subresource: "status",
						Time:        timePtr(statusWriteNewest),
						FieldsV1:    rawFields(`{"f:status":{"f:attachedAgents":{},"f:agentNames":{}}}`),
					},
				},
			},
			want: created,
		},
		{
			name: "status-subresource entry newest is ignored, older spec edit wins",
			obj: &metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(created),
				ManagedFields: []metav1.ManagedFieldsEntry{
					{
						Subresource: "",
						Time:        timePtr(specEditOlder),
						FieldsV1:    rawFields(`{"f:spec":{"f:content":{}}}`),
					},
					{
						Subresource: "status",
						Time:        timePtr(statusWriteNewest),
						FieldsV1:    rawFields(`{"f:status":{"f:attachedAgents":{}}}`),
					},
				},
			},
			want: specEditOlder,
		},
		{
			name: `subresource:"" entry touching only f:status is ignored`,
			obj: &metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(created),
				ManagedFields: []metav1.ManagedFieldsEntry{
					{
						Subresource: "",
						Time:        timePtr(specEditOlder),
						FieldsV1:    rawFields(`{"f:spec":{"f:content":{}}}`),
					},
					{
						// e.g. a client without a status subresource, or an older
						// client that wrote status through the main resource.
						Subresource: "",
						Time:        timePtr(statusWriteNewest),
						FieldsV1:    rawFields(`{"f:status":{"f:attachedAgents":{}}}`),
					},
				},
			},
			want: specEditOlder,
		},
		{
			name: "multiple spec edits, latest wins",
			obj: &metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(created),
				ManagedFields: []metav1.ManagedFieldsEntry{
					{
						Subresource: "",
						Time:        timePtr(specEditOlder),
						FieldsV1:    rawFields(`{"f:spec":{"f:content":{}}}`),
					},
					{
						Subresource: "",
						Time:        timePtr(specEditNewer),
						FieldsV1:    rawFields(`{"f:metadata":{"f:labels":{}},"f:spec":{"f:description":{}}}`),
					},
				},
			},
			want: specEditNewer,
		},
		{
			name: "empty managedFields falls back to creation time",
			obj: &metav1.ObjectMeta{
				CreationTimestamp: metav1.NewTime(created),
			},
			want: created,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastSpecUpdate(tt.obj)
			if !got.Equal(tt.want) {
				t.Errorf("lastSpecUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func timePtr(t time.Time) *metav1.Time {
	mt := metav1.NewTime(t)
	return &mt
}

func rawFields(s string) *metav1.FieldsV1 {
	return &metav1.FieldsV1{Raw: []byte(s)}
}
