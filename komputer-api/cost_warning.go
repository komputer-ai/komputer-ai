package main

import (
	"strings"
	"sync"
)

// warnedUnpriced remembers which agent/model pairs were already warned about,
// so a misconfigured agent logs once per API process rather than every task.
var warnedUnpriced sync.Map

// warnUnpricedModels logs a warning when Claude Code reports that it couldn't
// price a model it used (model_usage costBasis "unknown"). Its cost_usd is then
// at the wrong rates, and so are status totals, the cost metric and budget
// checks. The usual cause is an application inference profile ARN on an agent
// whose AWS role lacks bedrock:GetInferenceProfile, which Claude Code needs to
// find the model behind the ARN.
func warnUnpricedModels(event AgentEvent) {
	if event.Type != "task_completed" && event.Type != "task_cancelled" && event.Type != "error" {
		return
	}
	usage, _ := event.Payload["model_usage"].(map[string]interface{})
	for model, v := range usage {
		entry, _ := v.(map[string]interface{})
		if basis, _ := entry["costBasis"].(string); basis != "unknown" {
			continue
		}
		if _, seen := warnedUnpriced.LoadOrStore(event.Namespace+"/"+event.AgentName+"/"+model, true); seen {
			continue
		}
		hint := "Claude Code has no prices for this model, so the agent's costs and budget checks use the wrong rates"
		if strings.Contains(model, "application-inference-profile") {
			hint = "grant the agent's AWS role bedrock:GetInferenceProfile so Claude Code can find the model behind the ARN; until then the agent's costs and budget checks use the wrong rates"
		}
		Logger.Warnw("task cost is not priced for its model: "+hint,
			"agent_name", event.AgentName, "namespace", event.Namespace, "model", model)
	}
}
