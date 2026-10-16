package main

import "fmt"

// validEfforts are the reasoning-effort values the Claude Agent SDK accepts via
// ClaudeAgentOptions.effort / the CLI's --effort flag. Empty string means
// "unset" (left out of ClaudeAgentOptions entirely), which keeps the model's
// own default effort — this must stay valid everywhere effort is accepted.
var validEfforts = map[string]bool{
	"":       true,
	"low":    true,
	"medium": true,
	"high":   true,
	"xhigh":  true,
	"max":    true,
}

// validateEffort rejects any effort value the Claude Agent SDK doesn't accept.
// "" is always valid and means unset (model default) — it is never defaulted
// to any other value.
func validateEffort(effort string) error {
	if validEfforts[effort] {
		return nil
	}
	return fmt.Errorf("invalid effort %q: must be one of \"low\", \"medium\", \"high\", \"xhigh\", \"max\", or \"\" (unset)", effort)
}
