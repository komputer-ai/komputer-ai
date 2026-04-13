package client

import (
	"context"
	"net/http"
	"os"
	"testing"
)

// skipIfNoAPI skips the test if the komputer API is not reachable.
// Set KOMPUTER_API_URL to override the default localhost:8080.
func skipIfNoAPI(t *testing.T) string {
	t.Helper()
	url := os.Getenv("KOMPUTER_API_URL")
	if url == "" {
		url = "http://localhost:8080"
		resp, err := http.Get(url + "/api/v1/agents")
		if err != nil {
			t.Skip("Skipping integration tests: API not available at " + url)
		}
		resp.Body.Close()
	}
	return url
}

func newTestClient(t *testing.T) (*Client, context.Context) {
	t.Helper()
	url := skipIfNoAPI(t)
	return New(url), context.Background()
}

// --- Memories ---

func TestIntegration_MemoryCRUD(t *testing.T) {
	c, ctx := newTestClient(t)
	name := "sdk-test-go-mem-crud"

	// cleanup before and after
	c.DeleteMemory(ctx, name) //nolint:errcheck
	t.Cleanup(func() { c.DeleteMemory(ctx, name) }) //nolint:errcheck

	// Create
	created, _, err := c.CreateMemory(ctx, name, "initial content",
		CreateMemoryOpts{Description: PtrString("test memory description")})
	if err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}
	if created.GetName() != name {
		t.Errorf("expected name %q, got %q", name, created.GetName())
	}
	if created.GetContent() != "initial content" {
		t.Errorf("expected content %q, got %q", "initial content", created.GetContent())
	}
	if created.GetDescription() != "test memory description" {
		t.Errorf("expected description %q, got %q", "test memory description", created.GetDescription())
	}

	// Get
	got, _, err := c.GetMemory(ctx, name)
	if err != nil {
		t.Fatalf("GetMemory: %v", err)
	}
	if got.GetName() != name {
		t.Errorf("Get: expected name %q, got %q", name, got.GetName())
	}
	if got.GetContent() != "initial content" {
		t.Errorf("Get: expected content %q, got %q", "initial content", got.GetContent())
	}

	// List — verify the created memory appears
	list, _, err := c.ListMemories(ctx)
	if err != nil {
		t.Fatalf("ListMemories: %v", err)
	}
	if !memoryListContains(list, name) {
		t.Errorf("ListMemories: expected %q to appear in list", name)
	}

	// Patch
	patched, _, err := c.PatchMemory(ctx, name, PatchMemoryOpts{Content: PtrString("patched content")})
	if err != nil {
		t.Fatalf("PatchMemory: %v", err)
	}
	if patched.GetContent() != "patched content" {
		t.Errorf("Patch: expected content %q, got %q", "patched content", patched.GetContent())
	}

	// Delete
	_, _, err = c.DeleteMemory(ctx, name)
	if err != nil {
		t.Fatalf("DeleteMemory: %v", err)
	}

	// Verify deleted — GetMemory should now fail
	_, httpResp, err := c.GetMemory(ctx, name)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
	if httpResp != nil && httpResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", httpResp.StatusCode)
	}
}

func TestIntegration_MemoryIdempotentCreate(t *testing.T) {
	c, ctx := newTestClient(t)
	name := "sdk-test-go-mem-idem"

	c.DeleteMemory(ctx, name) //nolint:errcheck
	t.Cleanup(func() { c.DeleteMemory(ctx, name) }) //nolint:errcheck

	// First create
	_, _, err := c.CreateMemory(ctx, name, "original content")
	if err != nil {
		t.Fatalf("first CreateMemory: %v", err)
	}

	// Second create with different content — must not error (idempotent upsert)
	_, _, err = c.CreateMemory(ctx, name, "updated content")
	if err != nil {
		t.Fatalf("idempotent CreateMemory: %v", err)
	}

	// Verify content was updated
	got, _, err := c.GetMemory(ctx, name)
	if err != nil {
		t.Fatalf("GetMemory after idempotent create: %v", err)
	}
	if got.GetContent() != "updated content" {
		t.Errorf("expected content %q after idempotent create, got %q", "updated content", got.GetContent())
	}
}

// --- Skills ---

func TestIntegration_SkillCRUD(t *testing.T) {
	c, ctx := newTestClient(t)
	name := "sdk-test-go-skill-crud"

	c.DeleteSkill(ctx, name) //nolint:errcheck
	t.Cleanup(func() { c.DeleteSkill(ctx, name) }) //nolint:errcheck

	// Create
	created, _, err := c.CreateSkill(ctx, name, "echo hello", "prints hello")
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	if created.GetName() != name {
		t.Errorf("expected name %q, got %q", name, created.GetName())
	}
	if created.GetContent() != "echo hello" {
		t.Errorf("expected content %q, got %q", "echo hello", created.GetContent())
	}
	if created.GetDescription() != "prints hello" {
		t.Errorf("expected description %q, got %q", "prints hello", created.GetDescription())
	}

	// Get
	got, _, err := c.GetSkill(ctx, name)
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if got.GetName() != name {
		t.Errorf("Get: expected name %q, got %q", name, got.GetName())
	}
	if got.GetContent() != "echo hello" {
		t.Errorf("Get: expected content %q, got %q", "echo hello", got.GetContent())
	}

	// List — verify skill appears
	list, _, err := c.ListSkills(ctx)
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if !skillListContains(list, name) {
		t.Errorf("ListSkills: expected %q to appear in list", name)
	}

	// Patch
	patched, _, err := c.PatchSkill(ctx, name, PatchSkillOpts{
		Content:     PtrString("echo world"),
		Description: PtrString("prints world"),
	})
	if err != nil {
		t.Fatalf("PatchSkill: %v", err)
	}
	if patched.GetContent() != "echo world" {
		t.Errorf("Patch: expected content %q, got %q", "echo world", patched.GetContent())
	}
	if patched.GetDescription() != "prints world" {
		t.Errorf("Patch: expected description %q, got %q", "prints world", patched.GetDescription())
	}

	// Delete
	_, _, err = c.DeleteSkill(ctx, name)
	if err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}

	// Verify deleted
	_, httpResp, err := c.GetSkill(ctx, name)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
	if httpResp != nil && httpResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", httpResp.StatusCode)
	}
}

func TestIntegration_SkillIdempotentCreate(t *testing.T) {
	c, ctx := newTestClient(t)
	name := "sdk-test-go-skill-idem"

	c.DeleteSkill(ctx, name) //nolint:errcheck
	t.Cleanup(func() { c.DeleteSkill(ctx, name) }) //nolint:errcheck

	// First create
	_, _, err := c.CreateSkill(ctx, name, "original content", "original description")
	if err != nil {
		t.Fatalf("first CreateSkill: %v", err)
	}

	// Second create with different content — must not error
	_, _, err = c.CreateSkill(ctx, name, "updated content", "updated description")
	if err != nil {
		t.Fatalf("idempotent CreateSkill: %v", err)
	}

	// Verify content was updated
	got, _, err := c.GetSkill(ctx, name)
	if err != nil {
		t.Fatalf("GetSkill after idempotent create: %v", err)
	}
	if got.GetContent() != "updated content" {
		t.Errorf("expected content %q after idempotent create, got %q", "updated content", got.GetContent())
	}
	if got.GetDescription() != "updated description" {
		t.Errorf("expected description %q after idempotent create, got %q", "updated description", got.GetDescription())
	}
}

// --- Secrets ---

func TestIntegration_SecretCRUD(t *testing.T) {
	c, ctx := newTestClient(t)
	name := "sdk-test-go-secret-crud"

	c.DeleteSecret(ctx, name) //nolint:errcheck
	t.Cleanup(func() { c.DeleteSecret(ctx, name) }) //nolint:errcheck

	// Create
	created, _, err := c.CreateSecret(ctx, name, map[string]string{"key1": "value1", "key2": "value2"})
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if created.GetName() != name {
		t.Errorf("expected name %q, got %q", name, created.GetName())
	}
	// The API returns key names (not values) — verify both keys are present
	keys := created.GetKeys()
	if !containsString(keys, "key1") || !containsString(keys, "key2") {
		t.Errorf("expected keys [key1 key2], got %v", keys)
	}

	// List — verify the secret appears
	list, _, err := c.ListSecrets(ctx)
	if err != nil {
		t.Fatalf("ListSecrets: %v", err)
	}
	found := false
	for _, s := range list.GetSecrets() {
		if s.GetName() == name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListSecrets: expected %q to appear in list", name)
	}

	// Update
	_, _, err = c.UpdateSecret(ctx, name, map[string]string{"key3": "value3"})
	if err != nil {
		t.Fatalf("UpdateSecret: %v", err)
	}

	// Delete
	_, _, err = c.DeleteSecret(ctx, name)
	if err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
}

func TestIntegration_SecretIdempotentCreate(t *testing.T) {
	c, ctx := newTestClient(t)
	name := "sdk-test-go-secret-idem"

	c.DeleteSecret(ctx, name) //nolint:errcheck
	t.Cleanup(func() { c.DeleteSecret(ctx, name) }) //nolint:errcheck

	// First create
	_, _, err := c.CreateSecret(ctx, name, map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("first CreateSecret: %v", err)
	}

	// Second create with different data — must not error (client upserts on 409)
	_, _, err = c.CreateSecret(ctx, name, map[string]string{"baz": "qux"})
	if err != nil {
		t.Fatalf("idempotent CreateSecret: %v", err)
	}
}

// --- helpers ---

// memoryListContains checks whether a memory named `name` exists in the raw
// list response (map[string]interface{} keyed by "memories").
func memoryListContains(list map[string]interface{}, name string) bool {
	return namedItemExistsInList(list, "memories", name)
}

// skillListContains checks whether a skill named `name` exists in the raw
// list response (map[string]interface{} keyed by "skills").
func skillListContains(list map[string]interface{}, name string) bool {
	return namedItemExistsInList(list, "skills", name)
}

// namedItemExistsInList searches a list response map for an item with the given name.
// The list response is expected to have a top-level key (e.g. "memories" or "skills")
// whose value is []interface{}, each element being a map[string]interface{} with a
// "name" key.
func namedItemExistsInList(list map[string]interface{}, key string, name string) bool {
	raw, ok := list[key]
	if !ok {
		return false
	}
	items, ok := raw.([]interface{})
	if !ok {
		return false
	}
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if n, _ := m["name"].(string); n == name {
			return true
		}
	}
	return false
}

// containsString reports whether slice contains s.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
