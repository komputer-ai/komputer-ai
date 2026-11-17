package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ─── Config ──────────────────────────────────────────────────────────────────

type Config struct {
	APIEndpoint string `json:"apiEndpoint"`
	// Token is the API bearer token, for an API with auth on.
	Token string `json:"token,omitempty"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".komputer-ai", "config.json")
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	// 0600: the file can hold the API token. Chmod too, since WriteFile keeps
	// the mode of a file that already exists.
	if err := os.WriteFile(configPath(), data, 0600); err != nil {
		return err
	}
	return os.Chmod(configPath(), 0600)
}

// apiToken returns the API bearer token: KOMPUTER_API_TOKEN, else the saved one.
func apiToken() string {
	if t := strings.TrimSpace(os.Getenv("KOMPUTER_API_TOKEN")); t != "" {
		return t
	}
	if cfg, err := loadConfig(); err == nil {
		return cfg.Token
	}
	return ""
}

// authHeader is the header set for requests and WebSocket dials; nil without a token.
func authHeader() http.Header {
	t := apiToken()
	if t == "" {
		return nil
	}
	return http.Header{"Authorization": []string{"Bearer " + t}}
}

// resolveEndpoint gets the API endpoint from flag or config.
func resolveEndpoint(cmd *cobra.Command) string {
	ep, _ := cmd.Flags().GetString("api")
	if ep != "" {
		return strings.TrimRight(ep, "/")
	}
	cfg, err := loadConfig()
	if err != nil || cfg.APIEndpoint == "" {
		fmt.Println(errorStyle.Render("No API endpoint configured."))
		fmt.Println(dimStyle.Render("Run: komputer login <endpoint>  or use --api flag"))
		os.Exit(1)
	}
	return strings.TrimRight(cfg.APIEndpoint, "/")
}
