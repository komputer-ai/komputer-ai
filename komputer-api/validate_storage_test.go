package main

import (
	"testing"
	"time"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateEphemeralStorage(t *testing.T) {
	tests := []struct {
		name      string
		ephemeral bool
		lifecycle string
		sleepTTL  *metav1.Duration
		wantErr   bool
	}{
		{name: "non-ephemeral with Sleep is fine", ephemeral: false, lifecycle: "Sleep", wantErr: false},
		{name: "non-ephemeral with sleepTTL is fine", ephemeral: false, sleepTTL: &metav1.Duration{Duration: 30 * time.Minute}, wantErr: false},
		{name: "ephemeral with no lifecycle/sleepTTL is fine", ephemeral: true, wantErr: false},
		{name: "ephemeral with AutoDelete is fine", ephemeral: true, lifecycle: "AutoDelete", wantErr: false},
		{name: "ephemeral with Sleep is rejected", ephemeral: true, lifecycle: "Sleep", wantErr: true},
		{name: "ephemeral with sleepTTL is rejected", ephemeral: true, sleepTTL: &metav1.Duration{Duration: 30 * time.Minute}, wantErr: true},
		{name: "ephemeral with zero-duration sleepTTL is fine (treated as unset)", ephemeral: true, sleepTTL: &metav1.Duration{Duration: 0}, wantErr: false},
		{name: "ephemeral with both Sleep and sleepTTL is rejected", ephemeral: true, lifecycle: "Sleep", sleepTTL: &metav1.Duration{Duration: time.Hour}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEphemeralStorage(tc.ephemeral, tc.lifecycle, tc.sleepTTL, false)
			if tc.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateEphemeralImmutable(t *testing.T) {
	tests := []struct {
		name       string
		before     bool
		newStorage *komputerv1alpha1.StorageSpec
		wantErr    bool
	}{
		{name: "nil storage in patch is always allowed", before: true, newStorage: nil, wantErr: false},
		{name: "storage set without touching ephemeral is allowed", before: true, newStorage: &komputerv1alpha1.StorageSpec{Size: "10Gi"}, wantErr: false},
		{name: "flipping false to true is rejected", before: false, newStorage: &komputerv1alpha1.StorageSpec{Ephemeral: boolPtrAPI(true)}, wantErr: true},
		{name: "flipping true to false is rejected", before: true, newStorage: &komputerv1alpha1.StorageSpec{Ephemeral: boolPtrAPI(false)}, wantErr: true},
		{name: "re-sending the same value is allowed", before: true, newStorage: &komputerv1alpha1.StorageSpec{Ephemeral: boolPtrAPI(true)}, wantErr: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEphemeralImmutable(tc.before, tc.newStorage)
			if tc.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func boolPtrAPI(b bool) *bool { return &b }
