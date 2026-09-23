package controller

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// getKomputerAPIURL returns the komputer-api base URL. Checks the KOMPUTER_API_URL env
// var first (for local dev), then falls back to KomputerConfig (for in-cluster).
//
// Extracted from the per-reconciler copies that previously lived on
// KomputerScheduleReconciler and KomputerSquadReconciler.
func getKomputerAPIURL(ctx context.Context, c client.Client) (string, error) {
	if envURL := os.Getenv("KOMPUTER_API_URL"); envURL != "" {
		return envURL, nil
	}
	configList := &komputerv1alpha1.KomputerConfigList{}
	if err := c.List(ctx, configList); err != nil {
		return "", err
	}
	if len(configList.Items) == 0 {
		return "", fmt.Errorf("no KomputerConfig found")
	}
	url := configList.Items[0].Spec.APIURL
	if url == "" {
		return "", fmt.Errorf("KomputerConfig has no apiURL")
	}
	return url, nil
}

// cancelAgentTaskViaAPI calls POST /api/v1/agents/<name>/cancel on the running
// komputer-api, which cancels the agent's in-flight task while leaving the agent alive.
//
// Routing through the API rather than the pod directly is deliberate: the API's
// CancelAgentTask has a kubectl-exec fallback that also works in LOCAL=true mode.
//
// namespace is accepted for call-site clarity but not sent: the API resolves the
// agent's namespace itself.
func cancelAgentTaskViaAPI(ctx context.Context, c client.Client, namespace, agentName string) error {
	_ = namespace
	apiURL, err := getKomputerAPIURL(ctx, c)
	if err != nil {
		return err
	}
	cancelURL := fmt.Sprintf("%s/api/v1/agents/%s/cancel", apiURL, agentName)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cancelURL, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cancel returned %d", resp.StatusCode)
	}
	return nil
}
