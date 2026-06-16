package main

import "testing"

// TestValidateBedrockModel verifies that friendly Anthropic model names are
// rejected when the deployment runs against AWS Bedrock (where the SDK needs a
// region-prefixed inference-profile ID or an ARN), while valid Bedrock IDs and
// any model on non-Bedrock deployments pass through.
func TestValidateBedrockModel(t *testing.T) {
	cases := []struct {
		name     string
		bedrock  bool
		region   string
		model    string
		required bool
		wantErr  bool
	}{
		// Non-Bedrock: friendly names are correct, never rejected.
		{"non-bedrock friendly name", false, "", "claude-sonnet-4-6", false, false},
		{"non-bedrock empty required", false, "", "", true, false},

		// Bedrock: friendly names are the bug — reject them.
		{"bedrock friendly name", true, "us-east-1", "claude-sonnet-4-6", false, true},
		{"bedrock tier alias", true, "us-east-1", "sonnet", false, true},
		{"bedrock opus friendly", true, "eu-west-1", "claude-opus-4-7", true, true},

		// Bedrock: valid identifiers pass.
		{"bedrock us inference profile", true, "us-east-1", "us.anthropic.claude-sonnet-4-6", false, false},
		{"bedrock eu inference profile", true, "eu-west-1", "eu.anthropic.claude-sonnet-4-5-20250929-v1:0", false, false},
		{"bedrock apac inference profile", true, "ap-southeast-1", "apac.anthropic.claude-sonnet-4-6", false, false},
		{"bedrock arn", true, "us-east-1", "arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-sonnet-4-6", false, false},

		// Bedrock empty: rejected only when the model is required (new agent).
		{"bedrock empty required", true, "us-east-1", "", true, true},
		{"bedrock empty optional", true, "us-east-1", "", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.bedrock {
				t.Setenv("CLAUDE_CODE_USE_BEDROCK", "1")
			} else {
				t.Setenv("CLAUDE_CODE_USE_BEDROCK", "")
			}
			t.Setenv("AWS_REGION", tc.region)

			err := validateBedrockModel(tc.model, tc.required)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateBedrockModel(%q, required=%v) error = %v, wantErr = %v", tc.model, tc.required, err, tc.wantErr)
			}
		})
	}
}

// TestBedrockExampleRegionPrefix verifies the example model ID in error messages
// uses the correct region prefix so the guidance is actionable.
func TestBedrockExampleRegionPrefix(t *testing.T) {
	cases := map[string]string{
		"us-east-1":      "us.",
		"eu-west-1":      "eu.",
		"ap-southeast-1": "apac.",
		"us-gov-east-1":  "us-gov.",
		"":               "us.",
	}
	for region, wantPrefix := range cases {
		t.Run(region, func(t *testing.T) {
			t.Setenv("AWS_REGION", region)
			got := bedrockExampleModel()
			if len(got) < len(wantPrefix) || got[:len(wantPrefix)] != wantPrefix {
				t.Fatalf("bedrockExampleModel() for region %q = %q, want prefix %q", region, got, wantPrefix)
			}
		})
	}
}
