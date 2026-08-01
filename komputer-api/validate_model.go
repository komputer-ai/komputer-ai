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

// vertexModelRe matches a pinned Vertex AI model ID, e.g.
// "claude-sonnet-4-5@20250929". The "@<date>" suffix is what Vertex expects and
// what distinguishes a pinned ID from an Anthropic API alias like
// "claude-sonnet-4-6" (which on Vertex silently resolves to a lagging default
// that may not even be enabled in the project).
var vertexModelRe = regexp.MustCompile(`@\d{6,}$`)

// isValidVertexModel reports whether model is a pinned Vertex model ID.
func isValidVertexModel(model string) bool {
	return vertexModelRe.MatchString(model)
}

// validateVertexModel rejects Anthropic API aliases when the API runs against
// Google Vertex AI, where the Claude SDK needs a pinned "name@YYYYMMDD" ID.
// No-op off Vertex. required=true also rejects an empty model on the new-agent
// create path; wake/patch paths pass required=false (empty = "no change").
func validateVertexModel(model string, required bool) error {
	if os.Getenv("CLAUDE_CODE_USE_VERTEX") == "" {
		return nil
	}
	if model == "" {
		if required {
			return fmt.Errorf("model is required on Google Vertex AI: pass a pinned model ID like %q", vertexExampleModel())
		}
		return nil
	}
	if isValidVertexModel(model) {
		return nil
	}
	return fmt.Errorf("invalid Vertex AI model %q: pass a pinned model ID like %q (name@YYYYMMDD), not an Anthropic API alias", model, vertexExampleModel())
}

func vertexExampleModel() string { return "claude-sonnet-4-5@20250929" }

// validateModel validates the model against whichever managed model provider the
// deployment targets (AWS Bedrock or Google Vertex AI). It is a no-op on the
// direct Anthropic API, where friendly aliases are correct. Each provider
// validator no-ops when its own env var is unset, so calling both is safe.
func validateModel(model string, required bool) error {
	if err := validateBedrockModel(model, required); err != nil {
		return err
	}
	return validateVertexModel(model, required)
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
