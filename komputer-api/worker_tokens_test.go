package main

import (
	"testing"
)

func TestExtractTotalTokens(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]interface{}
		want    int64
	}{
		{
			name: "both input and output present",
			payload: map[string]interface{}{
				"usage": map[string]interface{}{
					"input_tokens":  float64(100),
					"output_tokens": float64(50),
				},
			},
			want: 150,
		},
		{
			name: "only input present",
			payload: map[string]interface{}{
				"usage": map[string]interface{}{
					"input_tokens": float64(200),
				},
			},
			want: 200,
		},
		{
			name: "usage is nil",
			payload: map[string]interface{}{
				"usage": nil,
			},
			want: 0,
		},
		{
			name:    "no usage key",
			payload: map[string]interface{}{},
			want:    0,
		},
		{
			name: "usage is not a map",
			payload: map[string]interface{}{
				"usage": "invalid",
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTotalTokens(tt.payload)
			if got != tt.want {
				t.Errorf("extractTotalTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}
