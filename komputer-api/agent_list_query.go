package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MaxAgentListLimit caps the `limit` query param on GET /agents so a single
// page request can't demand an unbounded response.
const MaxAgentListLimit = 200

// AgentListQuery holds the optional filter/sort/pagination parameters for
// GET /agents, already parsed and validated by the handler.
type AgentListQuery struct {
	// Status filters by agent phase, case-insensitive. "" matches everything.
	Status string
	// Labels are "key=value" filters, ANDed together.
	Labels map[string]string
	// Q is a case-insensitive substring match against the agent name.
	Q string
	// Sort is one of "created", "name", "status", "model", "lastActive", "cost",
	// optionally prefixed with "-" for descending. "" defaults to "-created".
	Sort string
	// Limit is nil when the caller omitted it — meaning "return everything"
	// (the pre-pagination, backward-compatible behavior). Offset is ignored
	// in that case.
	Limit  *int
	Offset int
}

// AgentListFacets lists every distinct namespace and "key=value" label pair
// present across the agents visible in the request's namespace scope — BEFORE
// status/label/q filtering is applied. This is intentional: it lets the UI's
// filter dropdowns always offer every available option, even once the status/
// label/q filters (or infinite-scroll pagination) have narrowed what's
// actually displayed.
type AgentListFacets struct {
	Namespaces []string `json:"namespaces"`
	Labels     []string `json:"labels"`
}

var validAgentSortKeys = map[string]bool{
	"created":    true,
	"name":       true,
	"status":     true,
	"model":      true,
	"lastActive": true,
	"cost":       true,
}

// parseAgentSort splits a sort query param like "-created" into its key and
// direction. An empty string defaults to "-created" (newest first). Returns
// an error if the key isn't one of the recognized sort keys.
func parseAgentSort(raw string) (key string, desc bool, err error) {
	if raw == "" {
		return "created", true, nil
	}
	desc = strings.HasPrefix(raw, "-")
	key = strings.TrimPrefix(raw, "-")
	if !validAgentSortKeys[key] {
		return "", false, fmt.Errorf("invalid sort key %q", key)
	}
	return key, desc, nil
}

// parseAgentListLimit validates the `limit` query param. An empty string
// returns (nil, nil) — "no limit requested". Otherwise the value must be a
// positive integer no greater than MaxAgentListLimit.
func parseAgentListLimit(raw string) (*int, error) {
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid limit %q: must be an integer", raw)
	}
	if v <= 0 {
		return nil, fmt.Errorf("invalid limit %q: must be positive", raw)
	}
	if v > MaxAgentListLimit {
		return nil, fmt.Errorf("invalid limit %q: must not exceed %d", raw, MaxAgentListLimit)
	}
	return &v, nil
}

// parseAgentListOffset validates the `offset` query param. An empty string
// defaults to 0. Negative values are rejected.
func parseAgentListOffset(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid offset %q: must be an integer", raw)
	}
	if v < 0 {
		return 0, fmt.Errorf("invalid offset %q: must not be negative", raw)
	}
	return v, nil
}

func computeAgentFacets(agents []AgentResponse) AgentListFacets {
	nsSeen := map[string]bool{}
	labelSeen := map[string]bool{}
	var namespaces, labels []string
	for _, a := range agents {
		if a.Namespace != "" && !nsSeen[a.Namespace] {
			nsSeen[a.Namespace] = true
			namespaces = append(namespaces, a.Namespace)
		}
		for k, v := range a.Labels {
			pair := k + "=" + v
			if !labelSeen[pair] {
				labelSeen[pair] = true
				labels = append(labels, pair)
			}
		}
	}
	sort.Strings(namespaces)
	sort.Strings(labels)
	return AgentListFacets{Namespaces: namespaces, Labels: labels}
}

func matchesAgentLabels(agent AgentResponse, filter map[string]string) bool {
	if len(filter) == 0 {
		return true
	}
	for k, v := range filter {
		if agent.Labels[k] != v {
			return false
		}
	}
	return true
}

// agentCostValue parses TotalCostUSD as a float; empty or unparseable values
// are treated as 0 so they sort alongside genuinely-zero-cost agents rather
// than causing an error.
func agentCostValue(a AgentResponse) float64 {
	if a.TotalCostUSD == "" {
		return 0
	}
	v, err := strconv.ParseFloat(a.TotalCostUSD, 64)
	if err != nil {
		return 0
	}
	return v
}

// parseAgentTime parses an RFC3339 timestamp, treating an empty or
// unparseable value as the zero time so comparisons still have a total order.
func parseAgentTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

func compareByString(av, bv string, desc bool) int {
	if av == bv {
		return 0
	}
	if (av < bv) != desc {
		return -1
	}
	return 1
}

func compareByFloat(av, bv float64, desc bool) int {
	if av == bv {
		return 0
	}
	if (av < bv) != desc {
		return -1
	}
	return 1
}

func compareByTime(at, bt time.Time, desc bool) int {
	if at.Equal(bt) {
		return 0
	}
	if at.Before(bt) != desc {
		return -1
	}
	return 1
}

// agentLessForSort reports whether a sorts strictly before b for the given
// key/direction. Every key ties back to Name ascending for stability. The
// lastActive key is special: agents that have never been active always sort
// after agents that have, in BOTH directions — only the ordering *within*
// each of those two groups flips with direction.
func agentLessForSort(a, b AgentResponse, key string, desc bool) bool {
	var cmp int
	switch key {
	case "name":
		cmp = compareByString(a.Name, b.Name, desc)
	case "status":
		cmp = compareByString(a.Status, b.Status, desc)
	case "model":
		cmp = compareByString(a.Model, b.Model, desc)
	case "cost":
		cmp = compareByFloat(agentCostValue(a), agentCostValue(b), desc)
	case "lastActive":
		aActive := a.LastActivityAt != ""
		bActive := b.LastActivityAt != ""
		if aActive != bActive {
			return aActive // active-before-never, regardless of desc
		}
		if aActive {
			cmp = compareByTime(parseAgentTime(a.LastActivityAt), parseAgentTime(b.LastActivityAt), desc)
		}
	default: // "created"
		cmp = compareByTime(parseAgentTime(a.CreatedAt), parseAgentTime(b.CreatedAt), desc)
	}
	if cmp != 0 {
		return cmp < 0
	}
	return a.Name < b.Name
}

// filterSortPaginateAgents is the pure function behind GET /agents: it
// applies status/label/q filters, sorts by the requested key, and slices the
// result to [offset, offset+limit). It is deliberately free of any K8s or
// HTTP dependency so it can be exhaustively table-tested.
//
// facets are computed from `agents` as given — i.e. over whatever namespace
// scope the caller already applied — BEFORE status/label/q filtering (see
// AgentListFacets doc).
//
// A nil query.Limit returns the full filtered+sorted list (offset is
// ignored) — this is what preserves exact backward compatibility for
// existing callers that never pass `limit`.
func filterSortPaginateAgents(agents []AgentResponse, query AgentListQuery) (page []AgentResponse, total int, nextOffset *int, facets AgentListFacets, err error) {
	facets = computeAgentFacets(agents)

	sortKey, desc, err := parseAgentSort(query.Sort)
	if err != nil {
		return nil, 0, nil, facets, err
	}

	q := strings.ToLower(strings.TrimSpace(query.Q))
	filtered := make([]AgentResponse, 0, len(agents))
	for _, a := range agents {
		if query.Status != "" && !strings.EqualFold(query.Status, a.Status) {
			continue
		}
		if !matchesAgentLabels(a, query.Labels) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(a.Name), q) {
			continue
		}
		filtered = append(filtered, a)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return agentLessForSort(filtered[i], filtered[j], sortKey, desc)
	})

	total = len(filtered)

	if query.Limit == nil {
		return filtered, total, nil, facets, nil
	}

	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + *query.Limit
	if end > total {
		end = total
	}
	page = filtered[offset:end]
	if end < total {
		n := end
		nextOffset = &n
	}
	return page, total, nextOffset, facets, nil
}
