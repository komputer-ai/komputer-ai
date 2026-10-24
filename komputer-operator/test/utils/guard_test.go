package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckE2EContext(t *testing.T) {
	cases := []struct {
		name, current, cluster, allowed string
		ok                              bool
	}{
		{"default kind cluster", "kind-kind", "kind", "", true},
		{"named kind cluster", "kind-komputer-operator-test-e2e", "komputer-operator-test-e2e", "", true},
		{"EKS", "arn:aws:eks:eu-central-1:123456789012:cluster/prod", "kind", "", false},
		{"docker-desktop", "docker-desktop", "kind", "", false},
		{"another kind cluster than the image is loaded into", "kind-other", "kind", "", false},
		{"no current context", "", "kind", "", false},
		{"explicitly allowed", "docker-desktop", "kind", "docker-desktop", true},
		{"allow names a different context", "arn:aws:eks:eu-central-1:123456789012:cluster/prod", "kind", "docker-desktop", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := CheckE2EContext(c.current, c.cluster, c.allowed)
			if c.ok && err != nil {
				t.Fatalf("want allowed, got %v", err)
			}
			if !c.ok {
				if err == nil {
					t.Fatal("want refused, got allowed")
				}
				if !strings.Contains(err.Error(), AllowContextEnv) {
					t.Errorf("error should name the override %s: %v", AllowContextEnv, err)
				}
			}
		})
	}
}

const twoContexts = `apiVersion: v1
kind: Config
clusters:
- name: local
  cluster: {server: "https://127.0.0.1:1"}
- name: remote
  cluster: {server: "https://203.0.113.1:443"}
users:
- name: u
  user: {token: t}
contexts:
- name: kind-kind
  context: {cluster: local, user: u}
- name: prod
  context: {cluster: remote, user: u}
current-context: kind-kind
`

// The pinned kubeconfig holds only the validated context, so nothing the
// suite runs can switch to another cluster.
func TestPinKubeContext_KeepsOnlyThatContext(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "config")
	if err := os.WriteFile(src, []byte(twoContexts), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", src)

	pinned, err := PinKubeContext("kind-kind", dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(pinned)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "127.0.0.1:1") || strings.Contains(got, "203.0.113.1") || strings.Contains(got, "prod") {
		t.Errorf("pinned kubeconfig should hold only kind-kind:\n%s", got)
	}
}
