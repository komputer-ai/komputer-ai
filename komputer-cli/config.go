package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ─── Config ──────────────────────────────────────────────────────────────────

type Config struct {
	APIEndpoint string `json:"apiEndpoint"`
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
	return os.WriteFile(configPath(), data, 0644)
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
