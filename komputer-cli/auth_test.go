package main

import (
	"os"
	"testing"
)

func TestAPITokenEnvOverridesConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := saveConfig(&Config{APIEndpoint: "http://x", Token: "from-config"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KOMPUTER_API_TOKEN", "")
	if got := apiToken(); got != "from-config" {
		t.Errorf("apiToken() = %q, want from-config", got)
	}
	t.Setenv("KOMPUTER_API_TOKEN", "from-env")
	if got := apiToken(); got != "from-env" {
		t.Errorf("apiToken() = %q, want from-env", got)
	}
	if h := authHeader().Get("Authorization"); h != "Bearer from-env" {
		t.Errorf("authHeader = %q", h)
	}
}

func TestAuthHeaderNilWithoutToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("KOMPUTER_API_TOKEN", "")
	if h := authHeader(); h != nil {
		t.Errorf("authHeader() = %v, want nil", h)
	}
}

func TestConfigFileIsPrivate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := saveConfig(&Config{APIEndpoint: "http://x", Token: "t"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(configPath())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config perm = %o, want 600", perm)
	}
}
