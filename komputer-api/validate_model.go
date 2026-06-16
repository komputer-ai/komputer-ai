package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// bedrockModelRe matches a region-prefixed Bedrock cross-region inference-profile
// ID for Claude, e.g. "us.anthropic.claude-sonnet-4-6". The region prefix
// (us/eu/apac/us-gov) is what Bedrock requires and what distinguishes a real
// Bedrock identifier from an Anthropic API friendly name like "claude-sonnet-4-6".
var bedrockModelRe = regexp.MustCompile(`^(us|eu|apac|us-gov)\.anthropic\.`)

// isValidBedrockModel reports whether model is a usable Bedrock identifier:
// a region-prefixed inference-profile ID or a full Bedrock ARN.
func isValidBedrockModel(model string) bool {
	return strings.HasPrefix(model, "arn:aws:bedrock:") || bedrockModelRe.MatchString(model)
}

// validateBedrockModel rejects Anthropic API friendly model names (e.g.
// "claude-sonnet-4-6") when the API runs against AWS Bedrock, where the Claude
// SDK needs a region-prefixed inference-profile ID or an ARN. Returning a
// human-actionable error here means manager agents see it via the create_agent
// MCP tool response and self-correct, instead of spawning a sub-agent that fails
// at runtime with "400 The provided model identifier is invalid".
//
// It is a no-op on non-Bedrock deployments (friendly names are correct there).
// required=true also rejects an empty model, used on the new-agent create path
// where an empty model would otherwise fall back to a friendly default that
// Bedrock rejects; wake/patch paths pass required=false (empty = "no change").
func validateBedrockModel(model string, required bool) error {
	if os.Getenv("CLAUDE_CODE_USE_BEDROCK") == "" {
		return nil
	}
	if model == "" {
		if required {
			return fmt.Errorf("model is required on AWS Bedrock: pass a region-prefixed inference-profile ID like %q or a full ARN", bedrockExampleModel())
		}
		return nil
	}
	if isValidBedrockModel(model) {
		return nil
	}
	return fmt.Errorf("invalid Bedrock model %q: pass a region-prefixed inference-profile ID like %q or a full ARN, not an Anthropic API name", model, bedrockExampleModel())
}

// bedrockExampleModel returns an example inference-profile ID using the region
// prefix derived from AWS_REGION, so error messages give actionable guidance.
func bedrockExampleModel() string {
	prefix := "us"
	region := os.Getenv("AWS_REGION")
	switch {
	case strings.HasPrefix(region, "us-gov-"):
		prefix = "us-gov"
	case strings.HasPrefix(region, "eu-"):
		prefix = "eu"
	case strings.HasPrefix(region, "ap-"):
		prefix = "apac"
	}
	return prefix + ".anthropic.claude-sonnet-4-6"
}
