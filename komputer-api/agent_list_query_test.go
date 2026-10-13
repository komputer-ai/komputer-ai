package main

import (
	"reflect"
	"testing"
)

func names(agents []AgentResponse) []string {
	out := make([]string, len(agents))
	for i, a := range agents {
		out[i] = a.Name
	}
	return out
}

func TestFilterSortPaginateAgents_SortKeys(t *testing.T) {
	agents := []AgentResponse{
		{Name: "a", Namespace: "ns1", Status: "Running", Model: "claude-3", CreatedAt: "2024-01-01T00:00:00Z", TotalCostUSD: "1.50"},
		{Name: "b", Namespace: "ns1", Status: "Sleeping", Model: "claude-2", CreatedAt: "2024-01-03T00:00:00Z", TotalCostUSD: "0.25"},
		{Name: "c", Namespace: "ns1", Status: "Failed", Model: "claude-1", CreatedAt: "2024-01-02T00:00:00Z", TotalCostUSD: "3.00"},
	}

	cases := []struct {
		sort string
		want []string
	}{
		{"created", []string{"a", "c", "b"}},
		{"-created", []string{"b", "c", "a"}},
		{"name", []string{"a", "b", "c"}},
		{"-name", []string{"c", "b", "a"}},
		{"status", []string{"c", "a", "b"}}, // Failed < Running < Sleeping
		{"-status", []string{"b", "a", "c"}},
		{"model", []string{"c", "b", "a"}}, // claude-1 < claude-2 < claude-3
		{"-model", []string{"a", "b", "c"}},
		{"cost", []string{"b", "a", "c"}}, // 0.25 < 1.50 < 3.00
		{"-cost", []string{"c", "a", "b"}},
		{"", []string{"b", "c", "a"}}, // default "-created"
	}

	for _, tc := range cases {
		t.Run(tc.sort, func(t *testing.T) {
			got, total, next, _, err := filterSortPaginateAgents(agents, AgentListQuery{Sort: tc.sort})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if total != 3 {
				t.Errorf("total = %d, want 3", total)
			}
			if next != nil {
				t.Errorf("nextOffset = %v, want nil (no limit)", *next)
			}
			if !reflect.DeepEqual(names(got), tc.want) {
				t.Errorf("sort %q: got %v, want %v", tc.sort, names(got), tc.want)
			}
		})
	}
}

func TestFilterSortPaginateAgents_InvalidSortKey(t *testing.T) {
	_, _, _, _, err := filterSortPaginateAgents(nil, AgentListQuery{Sort: "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid sort key, got nil")
	}
}

func TestFilterSortPaginateAgents_TiesBrokenByName(t *testing.T) {
	agents := []AgentResponse{
		{Name: "zebra", Status: "Running", CreatedAt: "2024-01-01T00:00:00Z"},
		{Name: "apple", Status: "Running", CreatedAt: "2024-01-01T00:00:00Z"},
		{Name: "mango", Status: "Running", CreatedAt: "2024-01-01T00:00:00Z"},
	}
	got, _, _, _, err := filterSortPaginateAgents(agents, AgentListQuery{Sort: "status"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"apple", "mango", "zebra"}
	if !reflect.DeepEqual(names(got), want) {
		t.Errorf("got %v, want %v (tie-break by name ascending)", names(got), want)
	}
}

func TestFilterSortPaginateAgents_LastActiveNeverSortsLastBothDirections(t *testing.T) {
	agents := []AgentResponse{
		{Name: "never1"},
		{Name: "active-old", LastActivityAt: "2024-01-01T00:00:00Z"},
		{Name: "never2"},
		{Name: "active-new", LastActivityAt: "2024-01-05T00:00:00Z"},
	}

	t.Run("ascending", func(t *testing.T) {
		got, _, _, _, err := filterSortPaginateAgents(agents, AgentListQuery{Sort: "lastActive"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"active-old", "active-new", "never1", "never2"}
		if !reflect.DeepEqual(names(got), want) {
			t.Errorf("got %v, want %v", names(got), want)
		}
	})

	t.Run("descending", func(t *testing.T) {
		got, _, _, _, err := filterSortPaginateAgents(agents, AgentListQuery{Sort: "-lastActive"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"active-new", "active-old", "never1", "never2"}
		if !reflect.DeepEqual(names(got), want) {
			t.Errorf("got %v, want %v (never-active must still sort last)", names(got), want)
		}
	})
}

func TestFilterSortPaginateAgents_CostNumericEmptyUnparseable(t *testing.T) {
	agents := []AgentResponse{
		{Name: "empty", TotalCostUSD: ""},
		{Name: "bad", TotalCostUSD: "not-a-number"},
		{Name: "paid", TotalCostUSD: "2.00"},
		{Name: "zero", TotalCostUSD: "0"},
	}
	got, _, _, _, err := filterSortPaginateAgents(agents, AgentListQuery{Sort: "cost"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// empty/unparseable/zero all treated as 0 -> tie, broken by name ascending; paid last.
	want := []string{"bad", "empty", "zero", "paid"}
	if !reflect.DeepEqual(names(got), want) {
		t.Errorf("got %v, want %v", names(got), want)
	}
}

func TestFilterSortPaginateAgents_QCaseInsensitive(t *testing.T) {
	agents := []AgentResponse{
		{Name: "Worker-One"},
		{Name: "worker-two"},
		{Name: "other"},
	}
	got, total, _, _, err := filterSortPaginateAgents(agents, AgentListQuery{Q: "WORKER", Sort: "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	want := []string{"Worker-One", "worker-two"}
	if !reflect.DeepEqual(names(got), want) {
		t.Errorf("got %v, want %v", names(got), want)
	}
}

func TestFilterSortPaginateAgents_LabelAND(t *testing.T) {
	agents := []AgentResponse{
		{Name: "a", Labels: map[string]string{"env": "prod", "team": "alpha"}},
		{Name: "b", Labels: map[string]string{"env": "prod", "team": "beta"}},
		{Name: "c", Labels: map[string]string{"env": "dev", "team": "alpha"}},
	}
	got, total, _, _, err := filterSortPaginateAgents(agents, AgentListQuery{
		Labels: map[string]string{"env": "prod", "team": "alpha"},
		Sort:   "name",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	want := []string{"a"}
	if !reflect.DeepEqual(names(got), want) {
		t.Errorf("got %v, want %v", names(got), want)
	}
}

func TestFilterSortPaginateAgents_StatusFilterCaseInsensitive(t *testing.T) {
	agents := []AgentResponse{
		{Name: "a", Status: "Running"},
		{Name: "b", Status: "Sleeping"},
	}
	got, total, _, _, err := filterSortPaginateAgents(agents, AgentListQuery{Status: "running"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].Name != "a" {
		t.Errorf("got %v (total %d), want [a] (total 1)", names(got), total)
	}
}

func TestFilterSortPaginateAgents_OffsetLimitSlicing(t *testing.T) {
	agents := make([]AgentResponse, 10)
	for i := range agents {
		agents[i] = AgentResponse{Name: string(rune('a' + i))}
	}

	cases := []struct {
		name       string
		offset     int
		limit      int
		wantNames  []string
		wantNext   *int
		wantTotal  int
	}{
		{"first page", 0, 3, []string{"a", "b", "c"}, intPtr(3), 10},
		{"second page", 3, 3, []string{"d", "e", "f"}, intPtr(6), 10},
		{"last partial page", 9, 3, []string{"j"}, nil, 10},
		{"offset exactly at end", 10, 3, []string{}, nil, 10},
		{"offset past end", 15, 3, []string{}, nil, 10},
		{"limit larger than remaining", 8, 100, []string{"i", "j"}, nil, 10},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, total, next, _, err := filterSortPaginateAgents(agents, AgentListQuery{
				Sort:   "name",
				Limit:  intPtr(tc.limit),
				Offset: tc.offset,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if total != tc.wantTotal {
				t.Errorf("total = %d, want %d", total, tc.wantTotal)
			}
			if !reflect.DeepEqual(names(got), tc.wantNames) {
				t.Errorf("names = %v, want %v", names(got), tc.wantNames)
			}
			if (next == nil) != (tc.wantNext == nil) || (next != nil && tc.wantNext != nil && *next != *tc.wantNext) {
				t.Errorf("nextOffset = %v, want %v", next, tc.wantNext)
			}
		})
	}
}

func TestFilterSortPaginateAgents_NoLimitReturnsEverything(t *testing.T) {
	agents := make([]AgentResponse, 500)
	for i := range agents {
		agents[i] = AgentResponse{Name: string(rune('a' + i%26))}
	}
	got, total, next, _, err := filterSortPaginateAgents(agents, AgentListQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 500 {
		t.Errorf("len(got) = %d, want 500 (no limit = everything)", len(got))
	}
	if total != 500 {
		t.Errorf("total = %d, want 500", total)
	}
	if next != nil {
		t.Errorf("nextOffset = %v, want nil", *next)
	}
}

func TestFilterSortPaginateAgents_FacetsIgnoreFilters(t *testing.T) {
	agents := []AgentResponse{
		{Name: "a", Namespace: "ns1", Status: "Running", Labels: map[string]string{"env": "prod"}},
		{Name: "b", Namespace: "ns2", Status: "Sleeping", Labels: map[string]string{"env": "dev"}},
		{Name: "c", Namespace: "ns1", Status: "Failed", Labels: map[string]string{"team": "alpha"}},
	}
	// Filter down to almost nothing...
	_, _, _, facets, err := filterSortPaginateAgents(agents, AgentListQuery{
		Status: "Running",
		Labels: map[string]string{"env": "prod"},
		Q:      "a",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ...but facets must still reflect ALL agents passed in, regardless of the filters.
	wantNS := []string{"ns1", "ns2"}
	if !reflect.DeepEqual(facets.Namespaces, wantNS) {
		t.Errorf("facets.Namespaces = %v, want %v", facets.Namespaces, wantNS)
	}
	wantLabels := []string{"env=dev", "env=prod", "team=alpha"}
	if !reflect.DeepEqual(facets.Labels, wantLabels) {
		t.Errorf("facets.Labels = %v, want %v", facets.Labels, wantLabels)
	}
}

func TestParseAgentListLimit(t *testing.T) {
	cases := []struct {
		raw     string
		wantNil bool
		wantErr bool
		wantVal int
	}{
		{"", true, false, 0},
		{"50", false, false, 50},
		{"0", false, true, 0},
		{"-5", false, true, 0},
		{"abc", false, true, 0},
		{"201", false, true, 0},
		{"200", false, false, 200},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := parseAgentListLimit(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.raw, err)
			}
			if tc.wantNil {
				if got != nil {
					t.Errorf("got %v, want nil", *got)
				}
				return
			}
			if got == nil || *got != tc.wantVal {
				t.Errorf("got %v, want %d", got, tc.wantVal)
			}
		})
	}
}

func TestParseAgentListOffset(t *testing.T) {
	cases := []struct {
		raw     string
		wantErr bool
		wantVal int
	}{
		{"", false, 0},
		{"0", false, 0},
		{"50", false, 50},
		{"-1", true, 0},
		{"abc", true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := parseAgentListOffset(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.raw, err)
			}
			if got != tc.wantVal {
				t.Errorf("got %d, want %d", got, tc.wantVal)
			}
		})
	}
}
