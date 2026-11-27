package main

import (
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func observeWarnings(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zapcore.WarnLevel)
	prev := Logger
	Logger = zap.New(core).Sugar()
	warnedUnpriced = sync.Map{}
	t.Cleanup(func() { Logger = prev; warnedUnpriced = sync.Map{} })
	return logs
}

const testProfileARN = "arn:aws:bedrock:eu-central-1:123456789012:application-inference-profile/abc"

func completedWith(agent string, usage map[string]interface{}) AgentEvent {
	return AgentEvent{AgentName: agent, Namespace: "default", Type: "task_completed",
		Payload: map[string]interface{}{"cost_usd": 0.065, "model_usage": usage}}
}

// The shape Claude Code reports when the role lacks bedrock:GetInferenceProfile.
var unpricedARNUsage = map[string]interface{}{
	testProfileARN: map[string]interface{}{"costUSD": 0.065, "canonicalModel": testProfileARN, "costBasis": "unknown"},
}

func TestWarnUnpricedModels_WarnsOncePerAgentAndModel(t *testing.T) {
	logs := observeWarnings(t)

	warnUnpricedModels(completedWith("a", unpricedARNUsage))
	warnUnpricedModels(completedWith("a", unpricedARNUsage))
	warnUnpricedModels(completedWith("b", unpricedARNUsage))

	got := logs.All()
	if len(got) != 2 {
		t.Fatalf("got %d warnings, want one each for agents a and b", len(got))
	}
	if !strings.Contains(got[0].Message, "bedrock:GetInferenceProfile") {
		t.Errorf("warning for an inference-profile ARN should name the permission: %q", got[0].Message)
	}
	if m := got[0].ContextMap()["model"]; m != testProfileARN {
		t.Errorf("model = %v, want the ARN", m)
	}
}

func TestWarnUnpricedModels_SilentWhenPriced(t *testing.T) {
	logs := observeWarnings(t)

	priced := map[string]interface{}{
		testProfileARN: map[string]interface{}{"costUSD": 0.022, "canonicalModel": "claude-haiku-4-5", "costBasis": "list"},
	}
	warnUnpricedModels(completedWith("a", priced))
	warnUnpricedModels(completedWith("a", nil))                                                                                    // older agent: no model_usage
	warnUnpricedModels(AgentEvent{AgentName: "a", Type: "text", Payload: map[string]interface{}{"model_usage": unpricedARNUsage}}) // not a terminal event

	if n := logs.Len(); n != 0 {
		t.Fatalf("got %d warnings, want none", n)
	}
}

func TestWarnUnpricedModels_OtherUnknownModel(t *testing.T) {
	logs := observeWarnings(t)

	warnUnpricedModels(AgentEvent{AgentName: "a", Namespace: "default", Type: "error", Payload: map[string]interface{}{
		"model_usage": map[string]interface{}{"my-gateway-model": map[string]interface{}{"costBasis": "unknown"}},
	}})

	got := logs.All()
	if len(got) != 1 || strings.Contains(got[0].Message, "GetInferenceProfile") {
		t.Fatalf("want one generic warning for a non-ARN model, got %+v", got)
	}
}
