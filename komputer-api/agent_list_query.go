package main

import (
	"fmt"
	"slices"
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
	// HideArchived leaves out Archived agents when Status is empty, so lists
	// show them only when asked for explicitly.
	HideArchived bool
	// TaskStatuses filters by task status, case-insensitive; an agent matches
	// if it has any of them. Empty matches everything.
	TaskStatuses []string
	// Labels are "key=value" filters, ANDed together.
	Labels map[string]string
	// FilterLabels optionally maps "namespace/name" to the agent CR's metadata
	// labels. Label filters match these when present (falling back to
	// AgentResponse.Labels, i.e. spec labels), so operator-set metadata labels
	// such as komputer.ai/office keep matching as they did when the filter was
	// pushed down to the K8s list call.
	FilterLabels map[string]map[string]string
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

// AgentListStats summarizes every agent matching the filters, before the
// limit/offset slice, so a page of agents can come with whole-list counts.
type AgentListStats struct {
	// ByStatus counts agents per phase. Agents with no phase yet aren't counted.
	ByStatus map[string]int `json:"byStatus"`
	// TotalCostUSD is the summed totalCostUSD, formatted like the agent field.
	TotalCostUSD string `json:"totalCostUSD"`
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
	// Empty, not nil: the UI maps over both lists, and nil encodes as null.
	namespaces, labels := []string{}, []string{}
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

func matchesAgentLabels(agent AgentResponse, filter map[string]string, filterLabels map[string]map[string]string) bool {
	if len(filter) == 0 {
		return true
	}
	labels := agent.Labels
	if m, ok := filterLabels[agent.Namespace+"/"+agent.Name]; ok {
		labels = m
	}
	for k, v := range filter {
		if labels[k] != v {
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

// agentSortKey holds an agent's sort values, parsed once before sorting so
// comparisons don't re-parse costs and timestamps.
type agentSortKey struct {
	a        *AgentResponse
	cost     float64
	created  time.Time
	active   time.Time
	isActive bool
}

func newAgentSortKey(a *AgentResponse) agentSortKey {
	return agentSortKey{
		a:        a,
		cost:     agentCostValue(*a),
		created:  parseAgentTime(a.CreatedAt),
		active:   parseAgentTime(a.LastActivityAt),
		isActive: a.LastActivityAt != "",
	}
}

// compareAgents orders a and b for the given key/direction. Every key ties
// back to Name ascending. The lastActive key is special: agents that have
// never been active always sort after agents that have, in BOTH directions —
// only the ordering *within* each of those two groups flips with direction.
func compareAgents(a, b *agentSortKey, key string, desc bool) int {
	var cmp int
	switch key {
	case "name":
		cmp = compareByString(a.a.Name, b.a.Name, desc)
	case "status":
		cmp = compareByString(a.a.Status, b.a.Status, desc)
	case "model":
		cmp = compareByString(a.a.Model, b.a.Model, desc)
	case "cost":
		cmp = compareByFloat(a.cost, b.cost, desc)
	case "lastActive":
		if a.isActive != b.isActive {
			if a.isActive {
				return -1 // active-before-never, regardless of desc
			}
			return 1
		}
		if a.isActive {
			cmp = compareByTime(a.active, b.active, desc)
		}
	default: // "created"
		cmp = compareByTime(a.created, b.created, desc)
	}
	if cmp != 0 {
		return cmp
	}
	return strings.Compare(a.a.Name, b.a.Name)
}

// filterSortPaginateAgents is the pure function behind GET /agents: it
// applies status/taskStatus/label/q filters, sorts by the requested key,
// summarizes the matches into stats, and slices the result to
// [offset, offset+limit). It is deliberately free of any K8s or
// HTTP dependency so it can be exhaustively table-tested.
//
// facets are computed from `agents` as given — i.e. over whatever namespace
// scope the caller already applied — BEFORE status/label/q filtering (see
// AgentListFacets doc).
//
// A nil query.Limit returns the full filtered+sorted list (offset is
// ignored) — this is what preserves exact backward compatibility for
// existing callers that never pass `limit`.
func filterSortPaginateAgents(agents []AgentResponse, query AgentListQuery) (AgentListResponse, error) {
	resp := AgentListResponse{Facets: computeAgentFacets(agents)}

	sortKey, desc, err := parseAgentSort(query.Sort)
	if err != nil {
		return resp, err
	}

	q := strings.ToLower(strings.TrimSpace(query.Q))
	// Filter and sort keys pointing into agents, so sorting moves small keys
	// instead of whole AgentResponses; only the returned agents are copied.
	keys := make([]agentSortKey, 0, len(agents))
	for i := range agents {
		a := &agents[i]
		if query.Status == "" && query.HideArchived && a.Status == "Archived" {
			continue
		}
		if query.Status != "" && !strings.EqualFold(query.Status, a.Status) {
			continue
		}
		if len(query.TaskStatuses) > 0 && !slices.ContainsFunc(query.TaskStatuses, func(s string) bool { return strings.EqualFold(s, a.TaskStatus) }) {
			continue
		}
		if !matchesAgentLabels(*a, query.Labels, query.FilterLabels) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(a.Name), q) {
			continue
		}
		keys = append(keys, newAgentSortKey(a))
	}

	slices.SortStableFunc(keys, func(x, y agentSortKey) int {
		return compareAgents(&x, &y, sortKey, desc)
	})

	resp.Stats = AgentListStats{ByStatus: map[string]int{}}
	var totalCost float64
	for i := range keys {
		totalCost += keys[i].cost
		if s := keys[i].a.Status; s != "" {
			resp.Stats.ByStatus[s]++
		}
	}
	resp.Stats.TotalCostUSD = fmt.Sprintf("%.4f", totalCost)

	total := len(keys)
	resp.Total = total
	offset, end := 0, total
	if query.Limit != nil {
		offset = min(max(query.Offset, 0), total)
		end = min(offset+*query.Limit, total)
		if end < total {
			n := end
			resp.NextOffset = &n
		}
	}
	resp.Agents = make([]AgentResponse, 0, end-offset)
	for _, k := range keys[offset:end] {
		resp.Agents = append(resp.Agents, *k.a)
	}
	return resp, nil
}
