package main

import (
	"fmt"
	"path"
)

// maxAgentsPerSubscription caps how many explicit agent names can be in a
// single multi-WS subscription. Wildcards aren't expanded ahead of time, so
// this only limits the explicit list. The overall fan-in is bounded at
// runtime by the worker reading streams that exist.
const maxAgentsPerSubscription = 200

// agentMatcher decides whether a (agentName, namespace) pair should be
// delivered to a multi-agent WS subscription.
type agentMatcher struct {
	patterns  []string        // validated glob patterns (path.Match syntax)
	names     map[string]bool // explicit set
	namespace string          // empty = no filter
}

// compileMatcher builds a matcher from raw user input. Returns an error if
// the subscription is empty (no patterns and no names), if any glob is
// invalid, or if the explicit list exceeds maxAgentsPerSubscription.
func compileMatcher(patterns, agents []string, namespace string) (*agentMatcher, error) {
	if len(patterns) == 0 && len(agents) == 0 {
		return nil, fmt.Errorf("subscription must include at least one match pattern or agent name")
	}
	if len(agents) > maxAgentsPerSubscription {
		return nil, fmt.Errorf("too many agents in subscription: %d > %d", len(agents), maxAgentsPerSubscription)
	}
	for _, p := range patterns {
		// Validate by running path.Match against an empty string; it returns
		// an error only on malformed patterns.
		if _, err := path.Match(p, ""); err != nil {
			return nil, fmt.Errorf("bad pattern %q: %w", p, err)
		}
	}
	names := make(map[string]bool, len(agents))
	for _, a := range agents {
		if a != "" {
			names[a] = true
		}
	}
	return &agentMatcher{patterns: patterns, names: names, namespace: namespace}, nil
}

func (m *agentMatcher) matches(agentName, namespace string) bool {
	if m.namespace != "" && m.namespace != namespace {
		return false
	}
	if m.names[agentName] {
		return true
	}
	for _, p := range m.patterns {
		ok, _ := path.Match(p, agentName)
		if ok {
			return true
		}
	}
	return false
}
