package main

import (
	"fmt"
	"reflect"
	"strings"
)

// agentFieldIndex maps each AgentResponse JSON key to its struct field index
// and whether the key is omitted when empty, so ?fields= output matches the
// full response's encoding for the keys it keeps.
var agentFieldIndex = func() map[string]agentField {
	t := reflect.TypeOf(AgentResponse{})
	m := make(map[string]agentField, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		name, opts, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		m[name] = agentField{index: i, omitEmpty: strings.Contains(opts, "omitempty")}
	}
	return m
}()

type agentField struct {
	index     int
	omitEmpty bool
}

// parseAgentFields validates the comma-separated `fields` query param of
// GET /agents against AgentResponse's JSON keys. "" returns nil, meaning
// every field. name and namespace are always kept so each item stays
// addressable.
func parseAgentFields(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	fields := []string{"name", "namespace"}
	seen := map[string]bool{"name": true, "namespace": true}
	for _, f := range strings.Split(raw, ",") {
		f = strings.TrimSpace(f)
		if f == "" || seen[f] {
			continue
		}
		if _, ok := agentFieldIndex[f]; !ok {
			return nil, fmt.Errorf("invalid field %q", f)
		}
		seen[f] = true
		fields = append(fields, f)
	}
	return fields, nil
}

// projectAgents keeps only the given JSON keys of each agent.
func projectAgents(agents []AgentResponse, fields []string) []map[string]any {
	out := make([]map[string]any, len(agents))
	for i := range agents {
		v := reflect.ValueOf(agents[i])
		m := make(map[string]any, len(fields))
		for _, f := range fields {
			af := agentFieldIndex[f]
			fv := v.Field(af.index)
			if af.omitEmpty && fv.IsZero() {
				continue
			}
			m[f] = fv.Interface()
		}
		out[i] = m
	}
	return out
}

// containsField reports whether fields (as parsed above) asks for f; nil asks
// for everything.
func containsField(fields []string, f string) bool {
	if fields == nil {
		return true
	}
	for _, x := range fields {
		if x == f {
			return true
		}
	}
	return false
}
