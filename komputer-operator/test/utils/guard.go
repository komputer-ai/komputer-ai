package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AllowContextEnv names the one kube context, other than the suite's kind
// cluster, that the e2e suite may run against. Exact match only.
const AllowContextEnv = "KOMPUTER_E2E_ALLOW_CONTEXT"

// KindClusterName is the kind cluster the suite loads its image into.
func KindClusterName() string {
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		return v
	}
	return "kind"
}

// CheckE2EContext refuses any kube context but the kind cluster the suite
// loads its image into (kind-<cluster>), or the one named in AllowContextEnv.
// The suite creates and deletes cluster-wide resources (cert-manager, CRDs,
// namespaces), so it must never reach a real cluster.
func CheckE2EContext(current, kindCluster, allowed string) error {
	if current != "" && (current == "kind-"+kindCluster || (allowed != "" && current == allowed)) {
		return nil
	}
	return fmt.Errorf("refusing to run the e2e suite against kube context %q: it installs and deletes "+
		"cluster-wide resources. Use the kind cluster %q (context %q), or set %s=<exact context name> "+
		"for a disposable cluster", current, kindCluster, "kind-"+kindCluster, AllowContextEnv)
}

// CurrentKubeContext reads the current context from the local kubeconfig
// (no cluster is contacted).
func CurrentKubeContext() (string, error) {
	out, err := exec.Command("kubectl", "config", "current-context").Output()
	return strings.TrimSpace(string(out)), err
}

// PinKubeContext writes a kubeconfig holding only kubeContext into dir and
// returns its path. Pointing KUBECONFIG at it pins every kubectl/make call
// the suite makes to that one cluster.
func PinKubeContext(kubeContext, dir string) (string, error) {
	out, err := exec.Command("kubectl", "config", "view", "--minify", "--flatten",
		"--context", kubeContext).Output()
	if err != nil {
		return "", fmt.Errorf("kubectl config view --context %s: %w", kubeContext, err)
	}
	path := filepath.Join(dir, "kubeconfig-e2e")
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return "", err
	}
	return path, nil
}
