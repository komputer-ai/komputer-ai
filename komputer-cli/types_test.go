package main

import "testing"

func TestStorageOverrideIsEphemeral(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name string
		s    *StorageOverride
		want bool
	}{
		{"nil receiver", nil, false},
		{"nil Ephemeral field", &StorageOverride{Size: "5Gi"}, false},
		{"Ephemeral explicitly false", &StorageOverride{Ephemeral: &falseVal}, false},
		{"Ephemeral explicitly true", &StorageOverride{Ephemeral: &trueVal}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.IsEphemeral(); got != tc.want {
				t.Errorf("IsEphemeral() = %v, want %v", got, tc.want)
			}
		})
	}
}
