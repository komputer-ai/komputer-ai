package main

// repriceInferenceProfileCost corrects cost_usd on a terminal task event whose
// configured model is an opaque Bedrock application-inference-profile ARN.
//
// cost_usd is Claude Code's own estimate, priced by model name. An ARN names no
// model, so that estimate can't be trusted to use the right rates. The agent also
// publishes model_usage (per-model tokens, keyed by the model name Claude Code
// saw) and response_model (the model the API reported), and those let us price
// the task ourselves from the same table backfill uses.
//
// The event is rewritten in place so every consumer — status totals, the cost
// metric, history and the cost breakdown — sees one number. Claude Code's value is
// kept as sdk_cost_usd. Events for a recognisable model, or where no model in the
// payload can be priced, are left untouched.
func repriceInferenceProfileCost(event *AgentEvent) {
	if event.Type != "task_completed" && event.Type != "task_cancelled" && event.Type != "error" {
		return
	}
	p := event.Payload
	sdkCost, ok := p["cost_usd"].(float64)
	if !ok {
		return
	}
	configured, _ := p["model"].(string)
	if _, known := lookupRates(configured); known {
		return
	}
	responseModel, _ := p["response_model"].(string)

	cost, ok := costFromModelUsage(p["model_usage"], responseModel)
	if !ok {
		return
	}
	p["sdk_cost_usd"] = sdkCost
	p["cost_usd"] = cost
	if cost != sdkCost {
		Logger.Infow("repriced inference-profile task cost",
			"agent_name", event.AgentName, "model", configured, "response_model", responseModel,
			"sdk_cost_usd", sdkCost, "cost_usd", cost)
	}
}

// costFromModelUsage prices Claude Code's per-model usage map. Each entry is
// priced by its own key when that names a model, else by responseModel. Returns
// false if the map is missing or any entry can't be priced, so a partial sum is
// never reported as the task's cost. Cache writes use the 5-minute rate; the map
// doesn't split 5-minute from 1-hour writes.
func costFromModelUsage(raw interface{}, responseModel string) (float64, bool) {
	usage, ok := raw.(map[string]interface{})
	if !ok || len(usage) == 0 {
		return 0, false
	}
	var total float64
	for model, v := range usage {
		entry, ok := v.(map[string]interface{})
		if !ok {
			return 0, false
		}
		rates, ok := lookupRates(model)
		if !ok {
			if rates, ok = lookupRates(responseModel); !ok {
				return 0, false
			}
		}
		num := func(k string) float64 { f, _ := entry[k].(float64); return f }
		total += (num("inputTokens")*rates.inputPerM +
			num("outputTokens")*rates.outputPerM +
			num("cacheReadInputTokens")*rates.cacheReadPerM +
			num("cacheCreationInputTokens")*rates.cacheWritePerM) / 1_000_000
	}
	return total, true
}
