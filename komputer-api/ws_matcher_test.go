package main

import "testing"

func TestCompileMatcher_Empty(t *testing.T) {
	if _, err := compileMatcher(nil, nil, ""); err == nil {
		t.Fatal("expected error for empty subscription, got nil")
	}
}

func TestCompileMatcher_WildcardMatchesAll(t *testing.T) {
	m, err := compileMatcher([]string{"*"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !m.matches("anything", "default") {
		t.Fatal("* should match all agents")
	}
	if !m.matches("alice", "other-ns") {
		t.Fatal("no namespace filter should match all namespaces")
	}
}

func TestCompileMatcher_PrefixGlob(t *testing.T) {
	m, _ := compileMatcher([]string{"worker-*"}, nil, "")
	if !m.matches("worker-1", "default") {
		t.Fatal("worker-1 should match worker-*")
	}
	if m.matches("manager-1", "default") {
		t.Fatal("manager-1 should not match worker-*")
	}
}

func TestCompileMatcher_ExplicitList(t *testing.T) {
	m, _ := compileMatcher(nil, []string{"alice", "bob"}, "")
	if !m.matches("alice", "default") {
		t.Fatal("alice in list")
	}
	if !m.matches("bob", "default") {
		t.Fatal("bob in list")
	}
	if m.matches("carol", "default") {
		t.Fatal("carol not in list")
	}
}

func TestCompileMatcher_Combined(t *testing.T) {
	m, _ := compileMatcher([]string{"worker-*"}, []string{"manager-1"}, "")
	if !m.matches("worker-3", "default") {
		t.Fatal("worker-3 via glob")
	}
	if !m.matches("manager-1", "default") {
		t.Fatal("manager-1 via explicit list")
	}
	if m.matches("other", "default") {
		t.Fatal("other should not match")
	}
}

func TestCompileMatcher_NamespaceFilter(t *testing.T) {
	m, _ := compileMatcher([]string{"*"}, nil, "team-a")
	if !m.matches("alice", "team-a") {
		t.Fatal("namespace match")
	}
	if m.matches("alice", "team-b") {
		t.Fatal("wrong namespace should not match")
	}
}

func TestCompileMatcher_RejectsBadPattern(t *testing.T) {
	if _, err := compileMatcher([]string{"["}, nil, ""); err == nil {
		t.Fatal("expected error for malformed pattern")
	}
}

func TestCompileMatcher_RejectsTooLargeFanin(t *testing.T) {
	// Build a list larger than the cap.
	big := make([]string, 0, maxAgentsPerSubscription+1)
	for i := 0; i < cap(big); i++ {
		big = append(big, "a")
	}
	if _, err := compileMatcher(nil, big, ""); err == nil {
		t.Fatal("expected error for oversized agent list")
	}
}
