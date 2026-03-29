package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

// ─── Spinner ────────────────────────────────────────────────────────────────

type Spinner struct {
	mu      sync.Mutex
	msg     string
	stop    chan struct{}
	stopped chan struct{}
}

func NewSpinner(msg string) *Spinner {
	s := &Spinner{
		msg:     msg,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *Spinner) run() {
	defer close(s.stopped)
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			// Clear the spinner line.
			fmt.Printf("\r\033[K")
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.msg
			s.mu.Unlock()
			frame := busyStyle.Render(frames[i%len(frames)])
			fmt.Printf("\r\033[K%s %s", frame, dimStyle.Render(msg))
			i++
		}
	}
}

func (s *Spinner) SetMessage(msg string) {
	s.mu.Lock()
	s.msg = msg
	s.mu.Unlock()
}

func (s *Spinner) Stop() {
	select {
	case <-s.stop:
		return // already stopped
	default:
		close(s.stop)
		<-s.stopped
	}
}

// ─── Styles ──────────────────────────────────────────────────────────────────

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED"))
	labelStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6B7280"))
	valueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB"))
	successStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981"))
	errorStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EF4444"))
	warnStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F59E0B"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	busyStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3B82F6"))
	idleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981"))

	// Event type styles
	eventTextStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9FAFB"))
	eventThinkStyle    = lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#A78BFA"))
	eventToolStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981"))
	eventResultStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6EE7B7"))
	eventErrorStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EF4444"))
	eventStartStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3B82F6"))
	eventCompleteStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#4C1D95")).
			PaddingBottom(1).
			MarginBottom(1)
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

// ─── API Types ───────────────────────────────────────────────────────────────

type AgentResponse struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	Model           string `json:"model"`
	Status          string `json:"status"`
	TaskStatus      string `json:"taskStatus"`
	LastTaskMessage string `json:"lastTaskMessage"`
	Lifecycle       string `json:"lifecycle"`
	LastTaskCostUSD string `json:"lastTaskCostUSD"`
	TotalCostUSD    string `json:"totalCostUSD"`
	CreatedAt       string `json:"createdAt"`
}

type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
}

type AgentEvent struct {
	AgentName string                 `json:"agentName"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type OfficeResponse struct {
	Name            string                 `json:"name"`
	Namespace       string                 `json:"namespace"`
	Manager         string                 `json:"manager"`
	Phase           string                 `json:"phase"`
	TotalAgents     int                    `json:"totalAgents"`
	ActiveAgents    int                    `json:"activeAgents"`
	CompletedAgents int                    `json:"completedAgents"`
	TotalCostUSD    string                 `json:"totalCostUSD"`
	Members         []OfficeMemberResponse `json:"members"`
	CreatedAt       string                 `json:"createdAt"`
}

type OfficeMemberResponse struct {
	Name            string `json:"name"`
	Role            string `json:"role"`
	TaskStatus      string `json:"taskStatus"`
	LastTaskCostUSD string `json:"lastTaskCostUSD"`
}

type OfficeListResponse struct {
	Offices []OfficeResponse `json:"offices"`
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func printAgent(a AgentResponse) {
	fmt.Println(headerStyle.Render(fmt.Sprintf("  %s  ", a.Name)))

	taskBadge := dimStyle.Render("—")
	switch a.TaskStatus {
	case "InProgress":
		taskBadge = busyStyle.Render("● In Progress")
	case "Complete":
		taskBadge = idleStyle.Render("✔ Complete")
	case "Error":
		taskBadge = errorStyle.Render("✗ Error")
	}

	phaseBadge := dimStyle.Render(a.Status)
	switch a.Status {
	case "Running":
		phaseBadge = successStyle.Render("Running")
	case "Pending":
		phaseBadge = warnStyle.Render("Pending")
	case "Failed":
		phaseBadge = errorStyle.Render("Failed")
	case "Sleeping":
		phaseBadge = dimStyle.Render("💤 Sleeping")
	}

	row := func(label, value string) {
		fmt.Printf("  %s %s\n", labelStyle.Render(fmt.Sprintf("%-16s", label)), valueStyle.Render(value))
	}

	row("Phase:", phaseBadge)
	row("Task:", taskBadge)
	row("Model:", a.Model)
	row("Namespace:", a.Namespace)
	if a.Lifecycle != "" {
		row("Lifecycle:", a.Lifecycle)
	}
	if a.LastTaskCostUSD != "" {
		row("Last Task Cost:", "$"+a.LastTaskCostUSD)
	}
	if a.TotalCostUSD != "" {
		row("Total Cost:", "$"+a.TotalCostUSD)
	}
	row("Created:", a.CreatedAt)
	if a.LastTaskMessage != "" {
		row("Last Message:", a.LastTaskMessage)
	}
	fmt.Println()
}

func apiRequest(method, url string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

func formatEvent(event AgentEvent) string {
	ts := dimStyle.Render(event.Timestamp)
	payload := event.Payload

	switch event.Type {
	case "task_started":
		instr, _ := payload["instructions"].(string)
		return fmt.Sprintf("%s %s\n%s", ts, eventStartStyle.Render("▶ Task Started"), eventTextStyle.Render("  "+instr))

	case "text":
		content, _ := payload["content"].(string)
		// Print content without lipgloss Render — it pads every line to terminal width,
		// causing excessive whitespace on multi-line markdown output.
		return fmt.Sprintf("%s %s\n  %s", ts, labelStyle.Render("💬 Text"), content)

	case "thinking":
		content, _ := payload["content"].(string)
		return fmt.Sprintf("%s %s\n%s", ts, dimStyle.Render("🧠 Thinking"), eventThinkStyle.Render("  "+content))

	case "tool_call":
		// tool_call events arrive batched at turn end, after tool_result events
		// already showed the tool execution in real-time. Skip to avoid duplicates.
		return ""

	case "tool_result":
		tool, _ := payload["tool"].(string)
		output, _ := payload["output"].(string)
		isError := strings.Contains(output, "Error") || strings.Contains(output, "error") || strings.Contains(output, "failed") || strings.Contains(output, "Stream closed")
		if isError {
			return fmt.Sprintf("%s %s %s\n%s", ts,
				eventErrorStyle.Render("✗ Tool Error:"),
				eventErrorStyle.Render(tool),
				eventErrorStyle.Render("  "+truncate(output, 500)))
		}
		// For Bash tools, show the command that was run.
		cmdLine := ""
		if tool == "Bash" {
			if inputMap, ok := payload["input"].(map[string]interface{}); ok {
				if cmd, ok := inputMap["command"].(string); ok {
					cmdLine = dimStyle.Render("  $ " + truncate(cmd, 150) + "\n")
				}
			}
		}
		return fmt.Sprintf("%s %s %s\n%s%s", ts,
			eventResultStyle.Render("✓"),
			eventResultStyle.Render(tool),
			cmdLine,
			dimStyle.Render("  "+truncate(output, 200)))

	case "task_completed":
		cost, _ := payload["cost_usd"].(float64)
		duration, _ := payload["duration_ms"].(float64)
		turns, _ := payload["turns"].(float64)

		return fmt.Sprintf("%s %s\n%s", ts,
			eventCompleteStyle.Render("✔ Task Completed"),
			dimStyle.Render(fmt.Sprintf("  Cost: $%.4f  Duration: %.1fs  Turns: %.0f", cost, duration/1000, turns)))

	case "task_cancelled":
		reason, _ := payload["reason"].(string)
		return fmt.Sprintf("%s %s\n%s", ts, warnStyle.Render("⚠ Task Cancelled"), dimStyle.Render("  "+reason))

	case "error":
		errMsg, _ := payload["error"].(string)
		return fmt.Sprintf("%s %s\n%s", ts, eventErrorStyle.Render("✗ Error"), eventErrorStyle.Render("  "+errMsg))

	default:
		raw, _ := json.Marshal(event)
		return fmt.Sprintf("%s %s %s", ts, dimStyle.Render("["+event.Type+"]"), dimStyle.Render(truncate(string(raw), 200)))
	}
}

// nsQuery returns "?namespace=X" or "" based on the --namespace flag.
func nsQuery(cmd *cobra.Command) string {
	ns, _ := cmd.Flags().GetString("namespace")
	if ns != "" {
		return "?namespace=" + url.QueryEscape(ns)
	}
	return ""
}

// nsQueryAmp returns "&namespace=X" or "" for appending to existing query params.
func nsQueryAmp(cmd *cobra.Command) string {
	ns, _ := cmd.Flags().GetString("namespace")
	if ns != "" {
		return "&namespace=" + url.QueryEscape(ns)
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// ─── Commands ────────────────────────────────────────────────────────────────

func main() {
	root := &cobra.Command{
		Use:   "komputer",
		Short: titleStyle.Render("komputer-cli") + " — CLI for the komputer.ai platform",
	}

	root.PersistentFlags().String("api", "", "API endpoint URL (overrides login config)")
	root.PersistentFlags().StringP("namespace", "n", "", "Kubernetes namespace (default: server default)")

	// ── login ────────────────────────────────────────────────────────────
	root.AddCommand(&cobra.Command{
		Use:   "login <endpoint>",
		Short: "Save API endpoint to ~/.komputer-ai/config.json",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			endpoint := strings.TrimRight(args[0], "/")
			if err := saveConfig(&Config{APIEndpoint: endpoint}); err != nil {
				fmt.Println(errorStyle.Render("Failed to save config: " + err.Error()))
				os.Exit(1)
			}
			fmt.Println(successStyle.Render("✔ Logged in"))
			fmt.Printf("  %s %s\n", labelStyle.Render("Endpoint:"), valueStyle.Render(endpoint))
			fmt.Printf("  %s %s\n", labelStyle.Render("Config:"), dimStyle.Render(configPath()))
		},
	})

	// ── list ─────────────────────────────────────────────────────────────
	root.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all agents",
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("GET", ep+"/api/v1/agents"+nsQuery(cmd), nil)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 200 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var resp AgentListResponse
			json.Unmarshal(data, &resp)

			if len(resp.Agents) == 0 {
				fmt.Println(dimStyle.Render("No agents found."))
				return
			}

			fmt.Println(titleStyle.Render(fmt.Sprintf("  %d agent(s)  ", len(resp.Agents))))
			fmt.Println()

			// Compute dynamic column widths.
			nameW := len("NAME")
			modelW := len("MODEL")
			for _, a := range resp.Agents {
				if len(a.Name) > nameW {
					nameW = len(a.Name)
				}
				if len(a.Model) > modelW {
					modelW = len(a.Model)
				}
			}
			nameW += 2  // padding
			modelW += 2 // padding
			totalW := nameW + 12 + 16 + modelW + 22

			// Table header
			fmt.Printf("  %s  %s  %s  %s  %s\n",
				labelStyle.Render(fmt.Sprintf("%-*s", nameW, "NAME")),
				labelStyle.Render(fmt.Sprintf("%-10s", "PHASE")),
				labelStyle.Render(fmt.Sprintf("%-14s", "TASK")),
				labelStyle.Render(fmt.Sprintf("%-*s", modelW, "MODEL")),
				labelStyle.Render(fmt.Sprintf("%-20s", "CREATED")),
			)
			fmt.Println(dimStyle.Render("  " + strings.Repeat("─", totalW)))

			for _, a := range resp.Agents {
				phase := a.Status
				switch phase {
				case "Running":
					phase = successStyle.Render(fmt.Sprintf("%-10s", phase))
				case "Pending":
					phase = warnStyle.Render(fmt.Sprintf("%-10s", phase))
				case "Failed":
					phase = errorStyle.Render(fmt.Sprintf("%-10s", phase))
				case "Sleeping":
					phase = dimStyle.Render(fmt.Sprintf("%-10s", "💤 Sleep"))
				default:
					phase = dimStyle.Render(fmt.Sprintf("%-10s", phase))
				}

				task := a.TaskStatus
				switch task {
				case "InProgress":
					task = busyStyle.Render(fmt.Sprintf("%-14s", "● In Progress"))
				case "Complete":
					task = idleStyle.Render(fmt.Sprintf("%-14s", "✔ Complete"))
				case "Error":
					task = errorStyle.Render(fmt.Sprintf("%-14s", "✗ Error"))
				default:
					task = dimStyle.Render(fmt.Sprintf("%-14s", "—"))
				}

				fmt.Printf("  %s  %s  %s  %s  %s\n",
					valueStyle.Render(fmt.Sprintf("%-*s", nameW, a.Name)),
					phase,
					task,
					dimStyle.Render(fmt.Sprintf("%-*s", modelW, a.Model)),
					dimStyle.Render(fmt.Sprintf("%-20s", a.CreatedAt)),
				)
			}
			fmt.Println()
		},
	})

	// ── create ───────────────────────────────────────────────────────────
	createCmd := &cobra.Command{
		Use:   "create <name> <instructions>",
		Short: "Create a new agent or send a task to an existing one",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			model, _ := cmd.Flags().GetString("model")
			templateRef, _ := cmd.Flags().GetString("template")

			ns, _ := cmd.Flags().GetString("namespace")
			secretFlags, _ := cmd.Flags().GetStringSlice("secret")

			body := map[string]interface{}{
				"name":         args[0],
				"instructions": args[1],
			}
			if model != "" {
				body["model"] = model
			}
			if templateRef != "" {
				body["templateRef"] = templateRef
			}
			if ns != "" {
				body["namespace"] = ns
			}
			if len(secretFlags) > 0 {
				secrets := make(map[string]string)
				for _, s := range secretFlags {
					parts := strings.SplitN(s, "=", 2)
					if len(parts) == 2 {
						secrets[parts[0]] = parts[1]
					}
				}
				body["secrets"] = secrets
			}
			if lc, _ := cmd.Flags().GetString("lifecycle"); lc != "" {
				body["lifecycle"] = lc
			}

			data, status, err := apiRequest("POST", ep+"/api/v1/agents", body)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}

			if status == 409 {
				var errResp ErrorResponse
				json.Unmarshal(data, &errResp)
				fmt.Println(warnStyle.Render("⚠ " + errResp.Error))
				os.Exit(1)
			}

			if status != 200 && status != 201 && status != 202 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var agent AgentResponse
			json.Unmarshal(data, &agent)

			if status == 201 {
				fmt.Println(successStyle.Render("✔ Agent created"))
			} else if status == 202 {
				fmt.Println(successStyle.Render("✔ Waking sleeping agent"))
			} else {
				fmt.Println(successStyle.Render("✔ Task sent to existing agent"))
			}
			printAgent(agent)
		},
	}
	createCmd.Flags().String("model", "", "Claude model to use")
	createCmd.Flags().String("template", "", "KomputerAgentTemplate name")
	createCmd.Flags().StringSlice("secret", nil, "Secrets as KEY=VALUE (repeatable, e.g. --secret GITHUB=ghp_xxx)")
	createCmd.Flags().String("lifecycle", "", "Agent lifecycle: Sleep (delete pod after task) or AutoDelete (delete agent after task)")
	root.AddCommand(createCmd)

	// ── get ──────────────────────────────────────────────────────────────
	getCmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get details of a specific agent and its recent events",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			agentName := args[0]
			limit, _ := cmd.Flags().GetInt("events")

			// Fetch agent details
			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/agents/%s%s", ep, url.PathEscape(agentName), nsQuery(cmd)), nil)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("Agent %q not found", agentName)))
				os.Exit(1)
			}
			if status != 200 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var agent AgentResponse
			json.Unmarshal(data, &agent)
			printAgent(agent)

			// Fetch recent events
			eventsData, eventsStatus, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/agents/%s/events?limit=%d%s", ep, url.PathEscape(agentName), limit, nsQueryAmp(cmd)), nil)
			if err != nil || eventsStatus != 200 {
				return // silently skip events if unavailable
			}

			var eventsResp struct {
				Events []AgentEvent `json:"events"`
			}
			json.Unmarshal(eventsData, &eventsResp)

			if len(eventsResp.Events) > 0 {
				fmt.Println(labelStyle.Render(fmt.Sprintf("  Recent Events (%d)", len(eventsResp.Events))))
				fmt.Println(dimStyle.Render("  " + strings.Repeat("─", 60)))
				for _, e := range eventsResp.Events {
					if formatted := formatEvent(e); formatted != "" {
						fmt.Println(formatted)
						fmt.Println()
					}
				}
			}
		},
	}
	getCmd.Flags().Int("events", 5, "Number of recent events to show")
	root.AddCommand(getCmd)

	// ── delete ───────────────────────────────────────────────────────────
	root.AddCommand(&cobra.Command{
		Use:     "delete <name> [name...]",
		Aliases: []string{"rm"},
		Short:   "Delete one or more agents and all their resources",
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			hasError := false
			for _, name := range args {
				data, status, err := apiRequest("DELETE", fmt.Sprintf("%s/api/v1/agents/%s%s", ep, url.PathEscape(name), nsQuery(cmd)), nil)
				if err != nil {
					fmt.Println(errorStyle.Render(fmt.Sprintf("Failed to delete %q: %s", name, err.Error())))
					hasError = true
					continue
				}
				if status == 404 {
					fmt.Println(errorStyle.Render(fmt.Sprintf("Agent %q not found", name)))
					hasError = true
					continue
				}
				if status != 200 {
					fmt.Println(errorStyle.Render(fmt.Sprintf("Failed to delete %q (%d): %s", name, status, string(data))))
					hasError = true
					continue
				}
				fmt.Println(successStyle.Render(fmt.Sprintf("✔ Agent %q deleted", name)))
			}
			if hasError {
				os.Exit(1)
			}
		},
	})

	// ── cancel ───────────────────────────────────────────────────────────
	root.AddCommand(&cobra.Command{
		Use:   "cancel <name>",
		Short: "Cancel the running task on an agent",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("POST", fmt.Sprintf("%s/api/v1/agents/%s/cancel%s", ep, url.PathEscape(args[0]), nsQuery(cmd)), nil)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("Agent %q not found", args[0])))
				os.Exit(1)
			}
			if status == 409 {
				var errResp ErrorResponse
				json.Unmarshal(data, &errResp)
				fmt.Println(warnStyle.Render("⚠ " + errResp.Error))
				os.Exit(1)
			}
			if status != 200 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			fmt.Println(warnStyle.Render(fmt.Sprintf("⚠ Cancelling task on %q", args[0])))
		},
	})

	// ── watch ────────────────────────────────────────────────────────────
	root.AddCommand(&cobra.Command{
		Use:   "watch <name>",
		Short: "Stream live events from an agent via WebSocket",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			agentName := args[0]

			// Verify agent exists before connecting WebSocket
			_, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/agents/%s%s", ep, url.PathEscape(agentName), nsQuery(cmd)), nil)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("Agent %q not found", agentName)))
				os.Exit(1)
			}

			// Convert http(s) to ws(s)
			wsURL := strings.Replace(ep, "https://", "wss://", 1)
			wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
			wsURL = fmt.Sprintf("%s/api/v1/agents/%s/ws", wsURL, url.PathEscape(agentName))

			fmt.Println(titleStyle.Render(fmt.Sprintf("  Watching %s  ", agentName)))
			fmt.Println(dimStyle.Render(fmt.Sprintf("  Connected to %s", wsURL)))
			fmt.Println(dimStyle.Render("  Press Ctrl+C to stop"))
			fmt.Println()

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				fmt.Println(errorStyle.Render("WebSocket connect failed: " + err.Error()))
				os.Exit(1)
			}
			defer conn.Close()

			// Handle Ctrl+C
			interrupted := make(chan struct{})
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)
			go func() {
				<-sigCh
				close(interrupted)
				conn.Close()
			}()

			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					select {
					case <-interrupted:
						fmt.Println()
						fmt.Println(dimStyle.Render("Disconnected."))
					default:
						if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
							fmt.Println(errorStyle.Render("WebSocket error: " + err.Error()))
						}
					}
					return
				}

				var event AgentEvent
				if err := json.Unmarshal(msg, &event); err != nil {
					fmt.Println(dimStyle.Render(string(msg)))
					continue
				}

				if formatted := formatEvent(event); formatted != "" {
					fmt.Println(formatted)
					fmt.Println()
				}

				// Exit on terminal events
				if event.Type == "task_completed" || event.Type == "task_cancelled" || event.Type == "error" {
					return
				}
			}
		},
	})

	// ── run (create + watch) ─────────────────────────────────────────────
	runCmd := &cobra.Command{
		Use:   "run <name> <instructions>",
		Short: "Create an agent and stream its output live",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			agentName := args[0]
			instructions := args[1]
			model, _ := cmd.Flags().GetString("model")

			// Create agent
			ns, _ := cmd.Flags().GetString("namespace")
			secretFlags, _ := cmd.Flags().GetStringSlice("secret")

			body := map[string]interface{}{
				"name":         agentName,
				"instructions": instructions,
			}
			if model != "" {
				body["model"] = model
			}
			if ns != "" {
				body["namespace"] = ns
			}
			if len(secretFlags) > 0 {
				secrets := make(map[string]string)
				for _, s := range secretFlags {
					parts := strings.SplitN(s, "=", 2)
					if len(parts) == 2 {
						secrets[parts[0]] = parts[1]
					}
				}
				body["secrets"] = secrets
			}
			if lc, _ := cmd.Flags().GetString("lifecycle"); lc != "" {
				body["lifecycle"] = lc
			}

			data, status, err := apiRequest("POST", ep+"/api/v1/agents", body)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 409 {
				var errResp ErrorResponse
				json.Unmarshal(data, &errResp)
				fmt.Println(warnStyle.Render("⚠ " + errResp.Error))
				os.Exit(1)
			}
			if status != 200 && status != 201 && status != 202 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var agent AgentResponse
			json.Unmarshal(data, &agent)

			if status == 201 {
				fmt.Println(successStyle.Render("✔ Agent created"))
			} else if status == 202 {
				fmt.Println(successStyle.Render("✔ Waking sleeping agent"))
			} else {
				fmt.Println(successStyle.Render("✔ Task sent"))
			}
			fmt.Println(dimStyle.Render(fmt.Sprintf("  Streaming events for %s...", agentName)))
			fmt.Println()

			// Connect WebSocket
			wsURL := strings.Replace(ep, "https://", "wss://", 1)
			wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
			wsURL = fmt.Sprintf("%s/api/v1/agents/%s/ws", wsURL, url.PathEscape(agentName))

			// Retry connection with spinner (pod may not be ready yet)
			spinner := NewSpinner("Waiting for agent pod to start...")
			var conn *websocket.Conn
			for i := 0; i < 30; i++ {
				conn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
				if err == nil {
					break
				}
				spinner.SetMessage(fmt.Sprintf("Waiting for agent pod to start... (%ds)", i+1))
				time.Sleep(1 * time.Second)
			}
			spinner.Stop()
			if conn == nil {
				fmt.Println(errorStyle.Render("Could not connect to WebSocket: " + err.Error()))
				os.Exit(1)
			}
			defer conn.Close()

			interrupted := make(chan struct{})
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)
			go func() {
				<-sigCh
				close(interrupted)
				conn.Close()
			}()

			// Idle spinner: shows when no events are received for a while.
			idleSpinner := (*Spinner)(nil)
			idleMu := sync.Mutex{}
			lastEvent := time.Now()

			// Background goroutine to start/update idle spinner.
			idleDone := make(chan struct{})
			go func() {
				defer close(idleDone)
				ticker := time.NewTicker(500 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-interrupted:
						return
					case <-ticker.C:
						idleMu.Lock()
						idle := time.Since(lastEvent)
						s := idleSpinner
						idleMu.Unlock()

						if idle > 3*time.Second && s == nil {
							ns := NewSpinner("Agent working...")
							idleMu.Lock()
							idleSpinner = ns
							idleMu.Unlock()
						} else if s != nil {
							elapsed := int(idle.Seconds())
							s.SetMessage(fmt.Sprintf("Agent working... (%ds)", elapsed))
						}
					}
				}
			}()

			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					idleMu.Lock()
					if idleSpinner != nil {
						idleSpinner.Stop()
					}
					idleMu.Unlock()
					select {
					case <-interrupted:
						fmt.Println()
						fmt.Println(dimStyle.Render("Disconnected."))
					default:
					}
					return
				}

				// Stop idle spinner when we get an event.
				idleMu.Lock()
				lastEvent = time.Now()
				if idleSpinner != nil {
					idleSpinner.Stop()
					idleSpinner = nil
				}
				idleMu.Unlock()

				var event AgentEvent
				if err := json.Unmarshal(msg, &event); err != nil {
					continue
				}

				if formatted := formatEvent(event); formatted != "" {
					fmt.Println(formatted)
					fmt.Println()
				}

				// Exit after task completion or cancellation
				if event.Type == "task_completed" || event.Type == "error" || event.Type == "task_cancelled" {
					return
				}
			}
		},
	}
	runCmd.Flags().String("model", "", "Claude model to use")
	runCmd.Flags().StringSlice("secret", nil, "Secrets as KEY=VALUE (repeatable, e.g. --secret GITHUB=ghp_xxx)")
	runCmd.Flags().String("lifecycle", "", "Agent lifecycle: Sleep (delete pod after task) or AutoDelete (delete agent after task)")
	root.AddCommand(runCmd)

	// ── office (parent) ─────────────────────────────────────────────────
	officeCmd := &cobra.Command{
		Use:   "office",
		Short: "Manage offices (groups of agents)",
	}

	// ── office list ─────────────────────────────────────────────────────
	officeCmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all offices",
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("GET", ep+"/api/v1/offices"+nsQuery(cmd), nil)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 200 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var resp OfficeListResponse
			json.Unmarshal(data, &resp)

			if len(resp.Offices) == 0 {
				fmt.Println(dimStyle.Render("No offices found."))
				return
			}

			fmt.Println(titleStyle.Render(fmt.Sprintf("  %d office(s)  ", len(resp.Offices))))
			fmt.Println()

			// Compute dynamic column widths.
			nameW := len("NAME")
			managerW := len("MANAGER")
			for _, o := range resp.Offices {
				if len(o.Name) > nameW {
					nameW = len(o.Name)
				}
				if len(o.Manager) > managerW {
					managerW = len(o.Manager)
				}
			}
			nameW += 2
			managerW += 2
			totalW := nameW + 16 + 8 + 8 + 10 + managerW + 22

			// Table header
			fmt.Printf("  %s  %s  %s  %s  %s  %s  %s\n",
				labelStyle.Render(fmt.Sprintf("%-*s", nameW, "NAME")),
				labelStyle.Render(fmt.Sprintf("%-14s", "PHASE")),
				labelStyle.Render(fmt.Sprintf("%-6s", "AGENTS")),
				labelStyle.Render(fmt.Sprintf("%-6s", "ACTIVE")),
				labelStyle.Render(fmt.Sprintf("%-8s", "COST")),
				labelStyle.Render(fmt.Sprintf("%-*s", managerW, "MANAGER")),
				labelStyle.Render(fmt.Sprintf("%-20s", "CREATED")),
			)
			fmt.Println(dimStyle.Render("  " + strings.Repeat("─", totalW)))

			for _, o := range resp.Offices {
				phase := o.Phase
				switch phase {
				case "InProgress":
					phase = busyStyle.Render(fmt.Sprintf("%-14s", "● In Progress"))
				case "Complete":
					phase = idleStyle.Render(fmt.Sprintf("%-14s", "✔ Complete"))
				case "Error":
					phase = errorStyle.Render(fmt.Sprintf("%-14s", "✗ Error"))
				default:
					phase = dimStyle.Render(fmt.Sprintf("%-14s", phase))
				}

				cost := "—"
				if o.TotalCostUSD != "" {
					cost = "$" + o.TotalCostUSD
				}

				fmt.Printf("  %s  %s  %s  %s  %s  %s  %s\n",
					valueStyle.Render(fmt.Sprintf("%-*s", nameW, o.Name)),
					phase,
					valueStyle.Render(fmt.Sprintf("%-6d", o.TotalAgents)),
					valueStyle.Render(fmt.Sprintf("%-6d", o.ActiveAgents)),
					valueStyle.Render(fmt.Sprintf("%-8s", cost)),
					dimStyle.Render(fmt.Sprintf("%-*s", managerW, o.Manager)),
					dimStyle.Render(fmt.Sprintf("%-20s", o.CreatedAt)),
				)
			}
			fmt.Println()
		},
	})

	// ── office get ──────────────────────────────────────────────────────
	officeGetCmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get office details and members",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			officeName := args[0]

			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/offices/%s%s", ep, url.PathEscape(officeName), nsQuery(cmd)), nil)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("Office %q not found", officeName)))
				os.Exit(1)
			}
			if status != 200 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var office OfficeResponse
			json.Unmarshal(data, &office)

			// Office header
			fmt.Println(headerStyle.Render(fmt.Sprintf("  %s  ", office.Name)))

			phaseBadge := dimStyle.Render(office.Phase)
			switch office.Phase {
			case "InProgress":
				phaseBadge = busyStyle.Render("● In Progress")
			case "Complete":
				phaseBadge = idleStyle.Render("✔ Complete")
			case "Error":
				phaseBadge = errorStyle.Render("✗ Error")
			}

			cost := "—"
			if office.TotalCostUSD != "" {
				cost = "$" + office.TotalCostUSD
			}

			row := func(label, value string) {
				fmt.Printf("  %s %s\n", labelStyle.Render(fmt.Sprintf("%-16s", label)), valueStyle.Render(value))
			}

			row("Phase:", phaseBadge)
			row("Manager:", office.Manager)
			row("Total Cost:", cost)
			row("Agents:", fmt.Sprintf("%d total, %d active, %d complete", office.TotalAgents, office.ActiveAgents, office.CompletedAgents))
			row("Created:", office.CreatedAt)
			fmt.Println()

			// Members table
			if len(office.Members) > 0 {
				fmt.Println(labelStyle.Render("  Members"))

				memberNameW := len("NAME")
				memberRoleW := len("ROLE")
				for _, m := range office.Members {
					if len(m.Name) > memberNameW {
						memberNameW = len(m.Name)
					}
					if len(m.Role) > memberRoleW {
						memberRoleW = len(m.Role)
					}
				}
				memberNameW += 2
				memberRoleW += 2
				memberTotalW := memberNameW + memberRoleW + 16 + 10

				fmt.Println(dimStyle.Render("  " + strings.Repeat("─", memberTotalW)))
				fmt.Printf("  %s  %s  %s  %s\n",
					labelStyle.Render(fmt.Sprintf("%-*s", memberNameW, "NAME")),
					labelStyle.Render(fmt.Sprintf("%-*s", memberRoleW, "ROLE")),
					labelStyle.Render(fmt.Sprintf("%-14s", "STATUS")),
					labelStyle.Render(fmt.Sprintf("%-8s", "COST")),
				)

				for _, m := range office.Members {
					taskBadge := dimStyle.Render(fmt.Sprintf("%-14s", "—"))
					switch m.TaskStatus {
					case "InProgress":
						taskBadge = busyStyle.Render(fmt.Sprintf("%-14s", "● In Progress"))
					case "Complete":
						taskBadge = idleStyle.Render(fmt.Sprintf("%-14s", "✔ Complete"))
					case "Error":
						taskBadge = errorStyle.Render(fmt.Sprintf("%-14s", "✗ Error"))
					}

					memberCost := "—"
					if m.LastTaskCostUSD != "" {
						memberCost = "$" + m.LastTaskCostUSD
					}

					fmt.Printf("  %s  %s  %s  %s\n",
						valueStyle.Render(fmt.Sprintf("%-*s", memberNameW, m.Name)),
						dimStyle.Render(fmt.Sprintf("%-*s", memberRoleW, m.Role)),
						taskBadge,
						valueStyle.Render(fmt.Sprintf("%-8s", memberCost)),
					)
				}
				fmt.Println()
			}

			// Recent events
			eventsLimit, _ := cmd.Flags().GetInt("events")
			eventsData, eventsStatus, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/offices/%s/events?limit=%d%s", ep, url.PathEscape(officeName), eventsLimit, nsQueryAmp(cmd)), nil)
			if err != nil || eventsStatus != 200 {
				return
			}

			var eventsResp struct {
				Events []AgentEvent `json:"events"`
			}
			json.Unmarshal(eventsData, &eventsResp)

			if len(eventsResp.Events) > 0 {
				fmt.Println(labelStyle.Render(fmt.Sprintf("  Recent Events (%d)", len(eventsResp.Events))))
				fmt.Println(dimStyle.Render("  " + strings.Repeat("─", 60)))
				for _, e := range eventsResp.Events {
					if formatted := formatEvent(e); formatted != "" {
						prefix := titleStyle.Render(fmt.Sprintf("[%s]", e.AgentName))
						fmt.Printf("%s %s\n\n", prefix, formatted)
					}
				}
			}
		},
	}
	officeGetCmd.Flags().Int("events", 10, "Number of recent events to show")
	officeCmd.AddCommand(officeGetCmd)

	// ── office watch ────────────────────────────────────────────────────
	officeCmd.AddCommand(&cobra.Command{
		Use:   "watch <name>",
		Short: "Stream live events from all agents in an office",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			officeName := args[0]

			// Verify office exists
			_, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/offices/%s%s", ep, url.PathEscape(officeName), nsQuery(cmd)), nil)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("Office %q not found", officeName)))
				os.Exit(1)
			}
			if status != 200 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d)", status)))
				os.Exit(1)
			}

			fmt.Println(titleStyle.Render(fmt.Sprintf("  Watching office %s  ", officeName)))
			fmt.Println(dimStyle.Render("  Polling events every 2 seconds. Press Ctrl+C to stop."))
			fmt.Println()

			// Handle Ctrl+C
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)

			lastTimestamp := ""
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			// Fetch once immediately, then on ticker
			fetchEvents := func() {
				eventsData, eventsStatus, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/offices/%s/events?limit=5%s", ep, url.PathEscape(officeName), nsQueryAmp(cmd)), nil)
				if err != nil || eventsStatus != 200 {
					return
				}

				var eventsResp struct {
					Events []AgentEvent `json:"events"`
				}
				json.Unmarshal(eventsData, &eventsResp)

				for _, e := range eventsResp.Events {
					if e.Timestamp <= lastTimestamp {
						continue
					}
					if formatted := formatEvent(e); formatted != "" {
						prefix := titleStyle.Render(fmt.Sprintf("[%s]", e.AgentName))
						fmt.Printf("%s %s\n\n", prefix, formatted)
					}
					lastTimestamp = e.Timestamp
				}
			}

			fetchEvents()
			for {
				select {
				case <-sigCh:
					fmt.Println()
					fmt.Println(dimStyle.Render("Stopped watching."))
					return
				case <-ticker.C:
					fetchEvents()
				}
			}
		},
	})

	// ── office delete ───────────────────────────────────────────────────
	officeCmd.AddCommand(&cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete an office and all its agents",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			officeName := args[0]

			data, status, err := apiRequest("DELETE", fmt.Sprintf("%s/api/v1/offices/%s%s", ep, url.PathEscape(officeName), nsQuery(cmd)), nil)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("Office %q not found", officeName)))
				os.Exit(1)
			}
			if status != 200 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Office %q deleted", officeName)))
		},
	})

	root.AddCommand(officeCmd)

	root.Execute()
}
