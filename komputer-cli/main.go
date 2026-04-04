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
	"github.com/chzyer/readline"
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

type ScheduleResponse struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	Schedule       string `json:"schedule"`
	Timezone       string `json:"timezone"`
	AutoDelete     bool   `json:"autoDelete"`
	KeepAgents     bool   `json:"keepAgents"`
	Phase          string `json:"phase"`
	AgentName      string `json:"agentName"`
	NextRunTime    string `json:"nextRunTime"`
	LastRunTime    string `json:"lastRunTime"`
	RunCount       int    `json:"runCount"`
	SuccessfulRuns int    `json:"successfulRuns"`
	FailedRuns     int    `json:"failedRuns"`
	TotalCostUSD   string `json:"totalCostUSD"`
	LastRunCostUSD string `json:"lastRunCostUSD"`
	LastRunStatus  string `json:"lastRunStatus"`
	CreatedAt      string `json:"createdAt"`
}

type ScheduleListResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
}

type MemoryResponse struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Agents      []string `json:"agents"`
	CreatedAt   string   `json:"createdAt"`
}

type MemoryListResponse struct {
	Memories []MemoryResponse `json:"memories"`
}

type SkillResponse struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Agents      []string `json:"agentNames"`
	CreatedAt   string   `json:"createdAt"`
}

type SkillListResponse struct {
	Skills []SkillResponse `json:"skills"`
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

// ─── JSON output ─────────────────────────────────────────────────────────────

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func dieJSON(msg string, httpStatus int) {
	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "  ")
	enc.Encode(map[string]any{"error": msg, "status": httpStatus})
	os.Exit(1)
}

// ─── Commands ────────────────────────────────────────────────────────────────

func main() {
	root := &cobra.Command{
		Use:   "komputer",
		Short: titleStyle.Render("komputer-cli") + " — CLI for the komputer.ai platform",
	}

	root.PersistentFlags().String("api", "", "API endpoint URL (overrides login config)")
	root.PersistentFlags().StringP("namespace", "n", "", "Kubernetes namespace (default: server default)")
	root.PersistentFlags().Bool("json", false, "Output raw JSON instead of formatted text")

	// ── login ────────────────────────────────────────────────────────────
	root.AddCommand(&cobra.Command{
		Use:   "login <endpoint>",
		Short: "Save API endpoint to ~/.komputer-ai/config.json",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			endpoint := strings.TrimRight(args[0], "/")
			if err := saveConfig(&Config{APIEndpoint: endpoint}); err != nil {
				if jsonMode {
					dieJSON("Failed to save config: "+err.Error(), 500)
				}
				fmt.Println(errorStyle.Render("Failed to save config: " + err.Error()))
				os.Exit(1)
			}
			if jsonMode {
				printJSON(map[string]string{"endpoint": endpoint, "config": configPath()})
				return
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
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("GET", ep+"/api/v1/agents"+nsQuery(cmd), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var resp AgentListResponse
			json.Unmarshal(data, &resp)

			if jsonMode {
				printJSON(resp)
				return
			}

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
			jsonMode, _ := cmd.Flags().GetBool("json")
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
			if memFlags, _ := cmd.Flags().GetStringSlice("memory"); len(memFlags) > 0 {
				body["memories"] = memFlags
			}
			if skillFlags, _ := cmd.Flags().GetStringSlice("skill"); len(skillFlags) > 0 {
				body["skills"] = skillFlags
			}

			data, status, err := apiRequest("POST", ep+"/api/v1/agents", body)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}

			if status == 409 {
				var errResp ErrorResponse
				json.Unmarshal(data, &errResp)
				if jsonMode {
					dieJSON(errResp.Error, 409)
				}
				fmt.Println(warnStyle.Render("⚠ " + errResp.Error))
				os.Exit(1)
			}

			if status != 200 && status != 201 && status != 202 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var agent AgentResponse
			json.Unmarshal(data, &agent)

			if jsonMode {
				printJSON(agent)
				return
			}

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
	createCmd.Flags().StringSlice("memory", nil, "Memory names to attach (repeatable, e.g. --memory k8s-debug)")
	createCmd.Flags().StringSlice("skill", nil, "Skill names to attach (repeatable, e.g. --skill python-expert)")
	root.AddCommand(createCmd)

	// ── get ──────────────────────────────────────────────────────────────
	getCmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get details of a specific agent and its recent events",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			agentName := args[0]
			limit, _ := cmd.Flags().GetInt("events")

			// Fetch agent details
			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/agents/%s%s", ep, url.PathEscape(agentName), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Agent %q not found", agentName), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Agent %q not found", agentName)))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var agent AgentResponse
			json.Unmarshal(data, &agent)

			// Fetch recent events
			eventsData, eventsStatus, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/agents/%s/events?limit=%d%s", ep, url.PathEscape(agentName), limit, nsQueryAmp(cmd)), nil)
			var events []AgentEvent
			if err == nil && eventsStatus == 200 {
				var eventsResp struct {
					Events []AgentEvent `json:"events"`
				}
				json.Unmarshal(eventsData, &eventsResp)
				events = eventsResp.Events
			}

			if jsonMode {
				printJSON(map[string]any{"agent": agent, "events": events})
				return
			}

			printAgent(agent)

			if len(events) > 0 {
				fmt.Println(labelStyle.Render(fmt.Sprintf("  Recent Events (%d)", len(events))))
				fmt.Println(dimStyle.Render("  " + strings.Repeat("─", 60)))
				for _, e := range events {
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
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			hasError := false
			type deleteResult struct {
				Name    string `json:"name"`
				Deleted bool   `json:"deleted"`
				Error   string `json:"error,omitempty"`
			}
			var results []deleteResult
			for _, name := range args {
				data, status, err := apiRequest("DELETE", fmt.Sprintf("%s/api/v1/agents/%s%s", ep, url.PathEscape(name), nsQuery(cmd)), nil)
				if err != nil {
					if jsonMode {
						results = append(results, deleteResult{Name: name, Deleted: false, Error: err.Error()})
						hasError = true
						continue
					}
					fmt.Println(errorStyle.Render(fmt.Sprintf("Failed to delete %q: %s", name, err.Error())))
					hasError = true
					continue
				}
				if status == 404 {
					if jsonMode {
						results = append(results, deleteResult{Name: name, Deleted: false, Error: "not found"})
						hasError = true
						continue
					}
					fmt.Println(errorStyle.Render(fmt.Sprintf("Agent %q not found", name)))
					hasError = true
					continue
				}
				if status != 200 {
					if jsonMode {
						results = append(results, deleteResult{Name: name, Deleted: false, Error: fmt.Sprintf("API error (%d): %s", status, string(data))})
						hasError = true
						continue
					}
					fmt.Println(errorStyle.Render(fmt.Sprintf("Failed to delete %q (%d): %s", name, status, string(data))))
					hasError = true
					continue
				}
				if jsonMode {
					results = append(results, deleteResult{Name: name, Deleted: true})
					continue
				}
				fmt.Println(successStyle.Render(fmt.Sprintf("✔ Agent %q deleted", name)))
			}
			if jsonMode {
				printJSON(results)
				if hasError {
					os.Exit(1)
				}
				return
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
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("POST", fmt.Sprintf("%s/api/v1/agents/%s/cancel%s", ep, url.PathEscape(args[0]), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Agent %q not found", args[0]), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Agent %q not found", args[0])))
				os.Exit(1)
			}
			if status == 409 {
				var errResp ErrorResponse
				json.Unmarshal(data, &errResp)
				if jsonMode {
					dieJSON(errResp.Error, 409)
				}
				fmt.Println(warnStyle.Render("⚠ " + errResp.Error))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			if jsonMode {
				printJSON(map[string]any{"name": args[0], "cancelled": true})
				return
			}
			fmt.Println(warnStyle.Render(fmt.Sprintf("⚠ Cancelling task on %q", args[0])))
		},
	})

	// ── config ───────────────────────────────────────────────────────────
	configCmd := &cobra.Command{
		Use:   "config <name>",
		Short: "Update settings on an existing agent (model, lifecycle, secrets)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			body := map[string]interface{}{}

			if model, _ := cmd.Flags().GetString("model"); model != "" {
				body["model"] = model
			}
			if lc, _ := cmd.Flags().GetString("lifecycle"); lc != "" {
				body["lifecycle"] = lc
			}
			if secretFlags, _ := cmd.Flags().GetStringSlice("secret"); len(secretFlags) > 0 {
				secrets := make(map[string]string)
				for _, s := range secretFlags {
					parts := strings.SplitN(s, "=", 2)
					if len(parts) == 2 {
						secrets[parts[0]] = parts[1]
					}
				}
				body["secrets"] = secrets
			}

			if memFlags, _ := cmd.Flags().GetStringSlice("memory"); len(memFlags) > 0 {
				body["memories"] = memFlags
			}
			if skillFlags, _ := cmd.Flags().GetStringSlice("skill"); len(skillFlags) > 0 {
				body["skills"] = skillFlags
			}

			if len(body) == 0 {
				if jsonMode {
					dieJSON("No settings provided. Use --model, --lifecycle, --secret, --memory, or --skill flags.", 400)
				}
				fmt.Println(errorStyle.Render("No settings provided. Use --model, --lifecycle, --secret, --memory, or --skill flags."))
				os.Exit(1)
			}

			data, status, err := apiRequest("PATCH", fmt.Sprintf("%s/api/v1/agents/%s%s", ep, url.PathEscape(args[0]), nsQuery(cmd)), body)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Agent %q not found", args[0]), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Agent %q not found", args[0])))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var agent AgentResponse
			json.Unmarshal(data, &agent)
			if jsonMode {
				printJSON(agent)
				return
			}
			fmt.Println(successStyle.Render("✔ Settings updated"))
			printAgent(agent)
		},
	}
	configCmd.Flags().String("model", "", "Claude model (e.g. claude-opus-4-6)")
	configCmd.Flags().String("lifecycle", "", "Lifecycle: Sleep or AutoDelete (empty for default)")
	configCmd.Flags().StringSlice("secret", nil, "Secrets as KEY=VALUE (repeatable, e.g. --secret GITHUB=ghp_xxx)")
	configCmd.Flags().StringSlice("memory", nil, "Memory names to attach (repeatable, e.g. --memory k8s-debug)")
	configCmd.Flags().StringSlice("skill", nil, "Skill names to attach (repeatable, e.g. --skill python-expert)")
	root.AddCommand(configCmd)

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

	// ── chat (interactive) ──────────────────────────────────────────────
	chatCmd := &cobra.Command{
		Use:   "chat <name>",
		Short: "Start an interactive chat session with an agent",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ep := resolveEndpoint(cmd)
			agentName := args[0]

			// 1. Check if agent exists — don't fail on 404, we'll create on first message
			agentExists := false
			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/agents/%s%s", ep, url.PathEscape(agentName), nsQuery(cmd)), nil)
			if err != nil {
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 200 {
				var agent AgentResponse
				json.Unmarshal(data, &agent)
				if agent.TaskStatus == "InProgress" {
					fmt.Println(errorStyle.Render(fmt.Sprintf("Agent %q is busy with a task. Wait for it to complete or cancel it.", agentName)))
					os.Exit(1)
				}
				agentExists = true
			} else if status != 404 {
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			// 2. Print header
			fmt.Println(titleStyle.Render(fmt.Sprintf("  Chat with %s  ", agentName)))
			fmt.Println(dimStyle.Render("  Type a message and press Enter. Ctrl+C to interrupt or exit."))
			fmt.Println()

			// 3. Signal handling: Ctrl+C during response cancels the turn,
			//    Ctrl+C at the prompt exits the chat.
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)
			streaming := false
			var streamingMu sync.Mutex
			cancelTurn := make(chan struct{}, 1)

			go func() {
				for range sigCh {
					streamingMu.Lock()
					isStreaming := streaming
					streamingMu.Unlock()
					if isStreaming {
						// Cancel the running task
						cancelURL := fmt.Sprintf("%s/api/v1/agents/%s/cancel%s", ep, url.PathEscape(agentName), nsQuery(cmd))
						apiRequest("POST", cancelURL, nil)
						select {
						case cancelTurn <- struct{}{}:
						default:
						}
					} else {
						// At the prompt — exit chat
						fmt.Println()
						fmt.Println(dimStyle.Render(fmt.Sprintf("  Chat ended. Agent %s is still running.", agentName)))
						os.Exit(0)
					}
				}
			}()

			// 4. WebSocket connection — established on first message send, persists across turns
			var conn *websocket.Conn
			defer func() {
				if conn != nil {
					conn.Close()
				}
			}()

			connectWS := func() error {
				if conn != nil {
					return nil // already connected
				}
				wsURL := strings.Replace(ep, "https://", "wss://", 1)
				wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
				wsURL = fmt.Sprintf("%s/api/v1/agents/%s/ws", wsURL, url.PathEscape(agentName))

				// Retry with spinner (pod may not be ready yet for new agents)
				spinner := NewSpinner("Waiting for agent to start...")
				var wsErr error
				for i := 0; i < 30; i++ {
					conn, _, wsErr = websocket.DefaultDialer.Dial(wsURL, nil)
					if wsErr == nil {
						break
					}
					spinner.SetMessage(fmt.Sprintf("Waiting for agent to start... (%ds)", i+1))
					time.Sleep(1 * time.Second)
				}
				spinner.Stop()
				if conn == nil {
					return fmt.Errorf("could not connect: %v", wsErr)
				}
				return nil
			}

			// If agent already exists, connect WebSocket immediately
			if agentExists {
				if err := connectWS(); err != nil {
					fmt.Println(errorStyle.Render("WebSocket connect failed: " + err.Error()))
					os.Exit(1)
				}
			}

			// 5. Input loop
			rl, rlErr := readline.NewEx(&readline.Config{
				Prompt:          "\033[1;34m> \033[0m", // bold blue prompt via ANSI (readline doesn't support lipgloss)
				InterruptPrompt: "^C",
				EOFPrompt:       "exit",
			})
			if rlErr != nil {
				fmt.Println(errorStyle.Render("Failed to initialize input: " + rlErr.Error()))
				os.Exit(1)
			}
			defer rl.Close()
			chatPromptStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3B82F6")).Inline(true)
			_ = chatPromptStyle // prompt handled by readline
			agentLabelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981")).Inline(true)
			chatDimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Inline(true)
			chatWarnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F59E0B")).Inline(true)
			chatErrStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EF4444")).Inline(true)
			toolNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Inline(true)        // light gray
			toolDetailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true).Inline(true) // dim italic
			toolIconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Inline(true)               // purple

			for {
				line, rlReadErr := rl.Readline()
				if rlReadErr != nil {
					// EOF (Ctrl+D) or interrupt
					fmt.Println()
					fmt.Println(dimStyle.Render(fmt.Sprintf("  Chat ended. Agent %s is still running.", agentName)))
					return
				}

				input := strings.TrimSpace(line)
				if input == "" {
					continue
				}

				// Show thinking spinner immediately
				fmt.Println()
				thinkSpinner := NewSpinner("Thinking...")
				thinkingActive := true

				// Send task to agent (creates agent if it doesn't exist).
				// No lifecycle set — agent stays running between turns.
				ns, _ := cmd.Flags().GetString("namespace")
				model, _ := cmd.Flags().GetString("model")
				body := map[string]interface{}{
					"name":         agentName,
					"instructions": input,
				}
				if ns != "" {
					body["namespace"] = ns
				}
				if model != "" {
					body["model"] = model
				}
				if lc, _ := cmd.Flags().GetString("lifecycle"); lc != "" {
					body["lifecycle"] = lc
				}

				// Retry on 409 (pod may still be starting)
				for attempt := 0; attempt < 10; attempt++ {
					data, status, err = apiRequest("POST", ep+"/api/v1/agents", body)
					if err != nil {
						thinkSpinner.Stop()
						fmt.Println(errorStyle.Render("  Failed to send message: " + err.Error()))
						break
					}
					if status == 409 {
						var errResp ErrorResponse
						json.Unmarshal(data, &errResp)
						// If agent is busy with a real task, don't retry
						if strings.Contains(errResp.Error, "busy") {
							thinkSpinner.Stop()
							fmt.Println(warnStyle.Render("  ⚠ " + errResp.Error))
							break
						}
						// Pod not ready yet — wait and retry
						thinkSpinner.SetMessage(fmt.Sprintf("Waiting for agent pod... (%ds)", attempt+1))
						time.Sleep(1 * time.Second)
						continue
					}
					break
				}
				if err != nil {
					continue
				}
				if status == 409 {
					var errResp ErrorResponse
					json.Unmarshal(data, &errResp)
					if strings.Contains(errResp.Error, "busy") {
						// Already printed above
					} else {
						thinkSpinner.Stop()
						fmt.Println(warnStyle.Render("  ⚠ Agent is not ready. Try again in a few seconds."))
					}
					continue
				}
				if status != 200 && status != 201 && status != 202 {
					thinkSpinner.Stop()
					fmt.Println(errorStyle.Render(fmt.Sprintf("  API error (%d): %s", status, string(data))))
					continue
				}

				// Agent was just created or woken — connect WebSocket with retry
				if status == 201 || status == 202 {
					thinkSpinner.Stop()
					if status == 201 {
						fmt.Println(dimStyle.Render("  Creating agent..."))
					} else {
						fmt.Println(dimStyle.Render("  Waking agent..."))
					}
					if err := connectWS(); err != nil {
						fmt.Println(errorStyle.Render("  " + err.Error()))
						return
					}
					thinkSpinner = NewSpinner("Thinking...")
					thinkingActive = true
				}

				// Stream response — read events until task_completed/error/cancelled
				firstText := true
				turnDone := false

				streamingMu.Lock()
				streaming = true
				streamingMu.Unlock()

				// Drain any pending cancel from previous turns
				select {
				case <-cancelTurn:
				default:
				}

				stopThinking := func() {
					if thinkingActive {
						thinkSpinner.Stop()
						thinkingActive = false
						fmt.Println()
					}
				}

				for !turnDone {
					// Check if Ctrl+C was pressed to cancel this turn
					select {
					case <-cancelTurn:
						stopThinking()
						fmt.Println(chatWarnStyle.Render("  ⚠ Cancelling..."))
						// Drain remaining events until terminal event
						for {
							_, msg, readErr := conn.ReadMessage()
							if readErr != nil {
								break
							}
							var ev AgentEvent
							if json.Unmarshal(msg, &ev) == nil {
								if ev.Type == "task_completed" || ev.Type == "task_cancelled" || ev.Type == "error" {
									break
								}
							}
						}
						fmt.Println()
						turnDone = true
						continue
					default:
					}

					_, msg, err := conn.ReadMessage()
					if err != nil {
						stopThinking()
						if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
							fmt.Println(errorStyle.Render("  Connection lost: " + err.Error()))
						}
						return
					}

					var event AgentEvent
					if err := json.Unmarshal(msg, &event); err != nil {
						continue
					}

					// Stop thinking spinner on first visible event
					if event.Type == "text" || event.Type == "tool_result" || event.Type == "task_completed" || event.Type == "task_cancelled" || event.Type == "error" {
						stopThinking()
					}

					switch event.Type {
					case "text":
						content, _ := event.Payload["content"].(string)
						if firstText {
							fmt.Println()
							fmt.Println(agentLabelStyle.Render(agentName + ":"))
							fmt.Println(content)
							firstText = false
						} else {
							fmt.Println(content)
						}

					case "tool_result":
						tool, _ := event.Payload["tool"].(string)
						displayName := tool
						// Clean up MCP tool names: mcp__komputer__create_agent → create_agent
						if strings.HasPrefix(tool, "mcp__komputer__") {
							displayName = strings.TrimPrefix(tool, "mcp__komputer__")
						}
						detail := ""
						if tool == "Bash" {
							if inputMap, ok := event.Payload["input"].(map[string]interface{}); ok {
								if cmdStr, ok := inputMap["command"].(string); ok {
									collapsed := strings.Join(strings.Fields(cmdStr), " ")
									detail = "$ " + truncate(collapsed, 200)
								}
							}
						} else if tool == "WebSearch" {
							if inputMap, ok := event.Payload["input"].(map[string]interface{}); ok {
								if q, ok := inputMap["query"].(string); ok {
									detail = q
								}
							}
						} else if tool == "Skill" {
							if inputMap, ok := event.Payload["input"].(map[string]interface{}); ok {
								if s, ok := inputMap["skill"].(string); ok {
									detail = s
									if a, ok := inputMap["args"].(string); ok && a != "" {
										detail += " " + a
									}
								}
							}
						} else if inputMap, ok := event.Payload["input"].(map[string]interface{}); ok {
							// For MCP and other tools, show key params
							var parts []string
							for _, key := range []string{"name", "instructions", "schedule", "query"} {
								if v, ok := inputMap[key].(string); ok && v != "" {
									parts = append(parts, key+"="+truncate(v, 60))
								}
							}
							if len(parts) > 0 {
								detail = strings.Join(parts, ", ")
							}
						}
						// Also show output snippet for MCP tools
						output, _ := event.Payload["output"].(string)
						icon := "⚙"
						if tool == "Skill" {
							icon = "✦"
						}
						if detail != "" {
							fmt.Printf("  %s %s %s\n", toolIconStyle.Render(icon), toolNameStyle.Render(displayName), toolDetailStyle.Render(detail))
						} else {
							fmt.Printf("  %s %s\n", toolIconStyle.Render(icon), toolNameStyle.Render(displayName))
						}
						if output != "" && strings.HasPrefix(tool, "mcp__") {
							fmt.Printf("    %s\n", toolDetailStyle.Render(truncate(output, 200)))
						}
						// Show working spinner while waiting for next event
						thinkSpinner = NewSpinner("Working...")
						thinkingActive = true

					case "task_completed":
						cost, _ := event.Payload["cost_usd"].(float64)
						duration, _ := event.Payload["duration_ms"].(float64)
						fmt.Println()
						fmt.Println(chatDimStyle.Render(fmt.Sprintf("✔ Cost: $%.4f  Duration: %.1fs", cost, duration/1000)))
						fmt.Println()
						turnDone = true

					case "task_cancelled":
						reason, _ := event.Payload["reason"].(string)
						fmt.Println()
						fmt.Println(chatWarnStyle.Render("  ⚠ Task cancelled: " + reason))
						fmt.Println()
						turnDone = true

					case "error":
						errMsg, _ := event.Payload["error"].(string)
						fmt.Println()
						fmt.Println(chatErrStyle.Render("  ✗ " + errMsg))
						fmt.Println()
						turnDone = true

						// Skip: task_started, thinking, tool_call
					}
				}

				streamingMu.Lock()
				streaming = false
				streamingMu.Unlock()
			}
		},
	}
	chatCmd.Flags().String("model", "", "Claude model to use")
	chatCmd.Flags().String("lifecycle", "", "Agent lifecycle: Sleep or AutoDelete (default: empty, pod stays running)")
	root.AddCommand(chatCmd)

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
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("GET", ep+"/api/v1/offices"+nsQuery(cmd), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var resp OfficeListResponse
			json.Unmarshal(data, &resp)

			if jsonMode {
				printJSON(resp)
				return
			}

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
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			officeName := args[0]

			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/offices/%s%s", ep, url.PathEscape(officeName), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Office %q not found", officeName), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Office %q not found", officeName)))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
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
			var events []AgentEvent
			if err == nil && eventsStatus == 200 {
				var eventsResp struct {
					Events []AgentEvent `json:"events"`
				}
				json.Unmarshal(eventsData, &eventsResp)
				events = eventsResp.Events
			}

			if jsonMode {
				printJSON(map[string]any{"office": office, "events": events})
				return
			}

			if len(events) > 0 {
				fmt.Println(labelStyle.Render(fmt.Sprintf("  Recent Events (%d)", len(events))))
				fmt.Println(dimStyle.Render("  " + strings.Repeat("─", 60)))
				for _, e := range events {
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
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			officeName := args[0]

			data, status, err := apiRequest("DELETE", fmt.Sprintf("%s/api/v1/offices/%s%s", ep, url.PathEscape(officeName), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Office %q not found", officeName), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Office %q not found", officeName)))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			if jsonMode {
				printJSON(map[string]any{"name": officeName, "deleted": true})
				return
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Office %q deleted", officeName)))
		},
	})

	root.AddCommand(officeCmd)

	// ── schedule (parent) ──────────────────────────────────────────────
	scheduleCmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage scheduled agent runs",
	}

	// ── schedule list ──────────────────────────────────────────────────
	scheduleCmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all schedules",
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("GET", ep+"/api/v1/schedules"+nsQuery(cmd), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var resp ScheduleListResponse
			json.Unmarshal(data, &resp)

			if jsonMode {
				printJSON(resp)
				return
			}

			if len(resp.Schedules) == 0 {
				fmt.Println(dimStyle.Render("No schedules found."))
				return
			}

			fmt.Println(titleStyle.Render(fmt.Sprintf("  %d schedule(s)  ", len(resp.Schedules))))
			fmt.Println()

			// Compute dynamic column widths.
			nameW := len("NAME")
			schedW := len("SCHEDULE")
			agentW := len("AGENT")
			for _, s := range resp.Schedules {
				if len(s.Name) > nameW {
					nameW = len(s.Name)
				}
				if len(s.Schedule) > schedW {
					schedW = len(s.Schedule)
				}
				if len(s.AgentName) > agentW {
					agentW = len(s.AgentName)
				}
			}
			nameW += 2
			schedW += 2
			agentW += 2
			totalW := nameW + schedW + 14 + agentW + 8 + 10 + 22

			// Table header
			fmt.Printf("  %s  %s  %s  %s  %s  %s  %s\n",
				labelStyle.Render(fmt.Sprintf("%-*s", nameW, "NAME")),
				labelStyle.Render(fmt.Sprintf("%-*s", schedW, "SCHEDULE")),
				labelStyle.Render(fmt.Sprintf("%-12s", "PHASE")),
				labelStyle.Render(fmt.Sprintf("%-*s", agentW, "AGENT")),
				labelStyle.Render(fmt.Sprintf("%-6s", "RUNS")),
				labelStyle.Render(fmt.Sprintf("%-8s", "COST")),
				labelStyle.Render(fmt.Sprintf("%-20s", "NEXT RUN")),
			)
			fmt.Println(dimStyle.Render("  " + strings.Repeat("─", totalW)))

			for _, s := range resp.Schedules {
				phase := s.Phase
				switch phase {
				case "Active":
					phase = successStyle.Render(fmt.Sprintf("%-12s", "● Active"))
				case "Suspended":
					phase = warnStyle.Render(fmt.Sprintf("%-12s", "● Suspended"))
				case "Error":
					phase = errorStyle.Render(fmt.Sprintf("%-12s", "● Error"))
				default:
					phase = dimStyle.Render(fmt.Sprintf("%-12s", phase))
				}

				cost := "—"
				if s.TotalCostUSD != "" {
					cost = "$" + s.TotalCostUSD
				}

				nextRun := s.NextRunTime
				if nextRun == "" {
					nextRun = "—"
				}
				if s.AutoDelete {
					nextRun = nextRun + " (one-time)"
				}

				fmt.Printf("  %s  %s  %s  %s  %s  %s  %s\n",
					valueStyle.Render(fmt.Sprintf("%-*s", nameW, s.Name)),
					dimStyle.Render(fmt.Sprintf("%-*s", schedW, s.Schedule)),
					phase,
					dimStyle.Render(fmt.Sprintf("%-*s", agentW, s.AgentName)),
					valueStyle.Render(fmt.Sprintf("%-6d", s.RunCount)),
					valueStyle.Render(fmt.Sprintf("%-8s", cost)),
					dimStyle.Render(fmt.Sprintf("%-20s", nextRun)),
				)
			}
			fmt.Println()
		},
	})

	// ── schedule get ───────────────────────────────────────────────────
	scheduleCmd.AddCommand(&cobra.Command{
		Use:   "get <name>",
		Short: "Get schedule details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			scheduleName := args[0]

			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/schedules/%s%s", ep, url.PathEscape(scheduleName), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Schedule %q not found", scheduleName), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Schedule %q not found", scheduleName)))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var sched ScheduleResponse
			json.Unmarshal(data, &sched)

			if jsonMode {
				printJSON(sched)
				return
			}

			// Schedule header
			fmt.Println(headerStyle.Render(fmt.Sprintf("  %s  ", sched.Name)))

			phaseBadge := dimStyle.Render(sched.Phase)
			switch sched.Phase {
			case "Active":
				phaseBadge = successStyle.Render("● Active")
			case "Suspended":
				phaseBadge = warnStyle.Render("● Suspended")
			case "Error":
				phaseBadge = errorStyle.Render("● Error")
			}

			row := func(label, value string) {
				fmt.Printf("  %s %s\n", labelStyle.Render(fmt.Sprintf("%-16s", label)), valueStyle.Render(value))
			}

			row("Schedule:", sched.Schedule)
			row("Timezone:", sched.Timezone)
			row("Phase:", phaseBadge)
			row("Agent:", sched.AgentName)

			if sched.NextRunTime != "" {
				row("Next Run:", sched.NextRunTime)
			}

			if sched.LastRunTime != "" {
				lastRunDisplay := sched.LastRunTime
				if sched.LastRunStatus != "" {
					lastRunDisplay = fmt.Sprintf("%s (%s)", sched.LastRunTime, sched.LastRunStatus)
				}
				row("Last Run:", lastRunDisplay)
			}

			row("Runs:", fmt.Sprintf("%d total, %d successful, %d failed", sched.RunCount, sched.SuccessfulRuns, sched.FailedRuns))

			if sched.TotalCostUSD != "" {
				row("Total Cost:", "$"+sched.TotalCostUSD)
			}
			if sched.LastRunCostUSD != "" {
				row("Last Cost:", "$"+sched.LastRunCostUSD)
			}

			if sched.AutoDelete {
				row("One-time:", "yes")
			}
			if sched.KeepAgents {
				row("Keep Agents:", "yes")
			}

			row("Created:", sched.CreatedAt)
			fmt.Println()
		},
	})

	// ── schedule create ────────────────────────────────────────────────
	scheduleCreateCmd := &cobra.Command{
		Use:   "create <name> <instructions>",
		Short: "Create a new schedule",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			cron, _ := cmd.Flags().GetString("cron")
			if cron == "" {
				if jsonMode {
					dieJSON("--cron flag is required", 400)
				}
				fmt.Println(errorStyle.Render("--cron flag is required"))
				os.Exit(1)
			}

			timezone, _ := cmd.Flags().GetString("timezone")
			autoDelete, _ := cmd.Flags().GetBool("auto-delete")
			keepAgents, _ := cmd.Flags().GetBool("keep-agents")
			agent, _ := cmd.Flags().GetString("agent")
			model, _ := cmd.Flags().GetString("model")
			lifecycle, _ := cmd.Flags().GetString("lifecycle")
			ns, _ := cmd.Flags().GetString("namespace")

			body := map[string]interface{}{
				"name":         args[0],
				"instructions": args[1],
				"schedule":     cron,
			}
			if timezone != "" {
				body["timezone"] = timezone
			}
			if autoDelete {
				body["autoDelete"] = true
			}
			if keepAgents {
				body["keepAgents"] = true
			}
			if agent != "" {
				// Reference existing agent
				body["agentName"] = agent
			} else {
				// Create agent from template
				agentSpec := map[string]interface{}{
					"lifecycle": lifecycle,
				}
				if model != "" {
					agentSpec["model"] = model
				}
				body["agent"] = agentSpec
			}
			if ns != "" {
				body["namespace"] = ns
			}

			data, status, err := apiRequest("POST", ep+"/api/v1/schedules", body)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 409 {
				var errResp ErrorResponse
				json.Unmarshal(data, &errResp)
				if jsonMode {
					dieJSON(errResp.Error, 409)
				}
				fmt.Println(warnStyle.Render("⚠ " + errResp.Error))
				os.Exit(1)
			}
			if status != 200 && status != 201 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}

			var sched ScheduleResponse
			json.Unmarshal(data, &sched)

			if jsonMode {
				printJSON(sched)
				return
			}

			fmt.Println(successStyle.Render("✔ Schedule created"))

			row := func(label, value string) {
				fmt.Printf("  %s %s\n", labelStyle.Render(fmt.Sprintf("%-16s", label)), valueStyle.Render(value))
			}

			row("Name:", sched.Name)
			row("Schedule:", sched.Schedule)
			row("Timezone:", sched.Timezone)
			row("Agent:", sched.AgentName)
			if sched.NextRunTime != "" {
				row("Next Run:", sched.NextRunTime)
			}
			if sched.AutoDelete {
				row("One-time:", "yes")
			}
			fmt.Println()
		},
	}
	scheduleCreateCmd.Flags().String("cron", "", "Cron expression (required, e.g. '0 9 * * MON-FRI')")
	scheduleCreateCmd.Flags().String("timezone", "UTC", "IANA timezone")
	scheduleCreateCmd.Flags().Bool("auto-delete", false, "Delete schedule after first successful run")
	scheduleCreateCmd.Flags().Bool("keep-agents", false, "Keep agents alive when schedule auto-deletes")
	scheduleCreateCmd.Flags().String("agent", "", "Reference existing agent instead of creating one")
	scheduleCreateCmd.Flags().String("model", "", "Claude model")
	scheduleCreateCmd.Flags().String("lifecycle", "Sleep", "Agent lifecycle (default: Sleep)")
	scheduleCmd.AddCommand(scheduleCreateCmd)

	// ── schedule delete ────────────────────────────────────────────────
	scheduleCmd.AddCommand(&cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a schedule and its managed agents",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			scheduleName := args[0]

			data, status, err := apiRequest("DELETE", fmt.Sprintf("%s/api/v1/schedules/%s%s", ep, url.PathEscape(scheduleName), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Schedule %q not found", scheduleName), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Schedule %q not found", scheduleName)))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			if jsonMode {
				printJSON(map[string]any{"name": scheduleName, "deleted": true})
				return
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Schedule %q deleted", scheduleName)))
		},
	})

	root.AddCommand(scheduleCmd)

	// ── memory ──────────────────────────────────────────────────────────
	memoryCmd := &cobra.Command{
		Use:   "memory",
		Short: "Manage agent memories",
	}

	// ── memory list ──────────────────────────────────────────────────
	memoryCmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all memories",
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/memories%s", ep, nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			var resp MemoryListResponse
			json.Unmarshal(data, &resp)
			if jsonMode {
				printJSON(resp)
				return
			}
			if len(resp.Memories) == 0 {
				fmt.Println(dimStyle.Render("No memories found."))
				return
			}
			fmt.Println(titleStyle.Render(fmt.Sprintf("  %d memory(s)  ", len(resp.Memories))))
			fmt.Println()
			nameW := len("NAME")
			descW := len("DESCRIPTION")
			for _, m := range resp.Memories {
				if len(m.Name) > nameW { nameW = len(m.Name) }
				d := m.Description
				if len(d) > 40 { d = d[:40] }
				if len(d) > descW { descW = len(d) }
			}
			header := fmt.Sprintf("  %-*s  %-*s  %s", nameW, "NAME", descW, "DESCRIPTION", "AGENTS")
			fmt.Println(dimStyle.Render(header))
			for _, m := range resp.Memories {
				desc := m.Description
				if len(desc) > 40 { desc = desc[:40] + "..." }
				fmt.Printf("  %-*s  %-*s  %d\n", nameW, m.Name, descW, desc, len(m.Agents))
			}
		},
	})

	// ── memory get ───────────────────────────────────────────────────
	memoryCmd.AddCommand(&cobra.Command{
		Use:   "get <name>",
		Short: "Get memory details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/memories/%s%s", ep, url.PathEscape(args[0]), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Memory %q not found", args[0]), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Memory %q not found", args[0])))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			var m MemoryResponse
			json.Unmarshal(data, &m)
			if jsonMode {
				printJSON(m)
				return
			}
			fmt.Println(headerStyle.Render(fmt.Sprintf("  %s  ", m.Name)))
			fmt.Println()
			if m.Description != "" {
				fmt.Printf("  Description: %s\n", m.Description)
			}
			fmt.Printf("  Namespace:   %s\n", m.Namespace)
			fmt.Printf("  Agents:      %d\n", len(m.Agents))
			fmt.Printf("  Created:     %s\n", m.CreatedAt)
			fmt.Println()
			fmt.Println(dimStyle.Render("  ── Content ──"))
			fmt.Println()
			fmt.Println(m.Content)
		},
	})

	// ── memory create ────────────────────────────────────────────────
	memoryCreateCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new memory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			content, _ := cmd.Flags().GetString("content")
			description, _ := cmd.Flags().GetString("description")
			if content == "" {
				if jsonMode {
					dieJSON("--content is required", 400)
				}
				fmt.Println(errorStyle.Render("--content is required"))
				os.Exit(1)
			}
			body := map[string]interface{}{
				"name":    args[0],
				"content": content,
			}
			if description != "" {
				body["description"] = description
			}
			if ns, _ := cmd.Flags().GetString("namespace"); ns != "" {
				body["namespace"] = ns
			}
			data, status, err := apiRequest("POST", ep+"/api/v1/memories", body)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 201 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			var m MemoryResponse
			json.Unmarshal(data, &m)
			if jsonMode {
				printJSON(m)
				return
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Memory %q created", args[0])))
		},
	}
	memoryCreateCmd.Flags().String("content", "", "Memory content (required)")
	memoryCreateCmd.Flags().String("description", "", "Short description")
	memoryCmd.AddCommand(memoryCreateCmd)

	// ── memory edit ──────────────────────────────────────────────────
	memoryEditCmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a memory's content or description",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			body := map[string]interface{}{}
			if content, _ := cmd.Flags().GetString("content"); content != "" {
				body["content"] = content
			}
			if description, _ := cmd.Flags().GetString("description"); description != "" {
				body["description"] = description
			}
			if len(body) == 0 {
				if jsonMode {
					dieJSON("No changes provided. Use --content or --description flags.", 400)
				}
				fmt.Println(errorStyle.Render("No changes provided. Use --content or --description flags."))
				os.Exit(1)
			}
			data, status, err := apiRequest("PATCH", fmt.Sprintf("%s/api/v1/memories/%s%s", ep, url.PathEscape(args[0]), nsQuery(cmd)), body)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Memory %q not found", args[0]), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Memory %q not found", args[0])))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			var m MemoryResponse
			json.Unmarshal(data, &m)
			if jsonMode {
				printJSON(m)
				return
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Memory %q updated", args[0])))
		},
	}
	memoryEditCmd.Flags().String("content", "", "New memory content")
	memoryEditCmd.Flags().String("description", "", "New description")
	memoryCmd.AddCommand(memoryEditCmd)

	// ── memory delete ────────────────────────────────────────────────
	memoryCmd.AddCommand(&cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a memory",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("DELETE", fmt.Sprintf("%s/api/v1/memories/%s%s", ep, url.PathEscape(args[0]), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			if jsonMode {
				printJSON(map[string]any{"name": args[0], "deleted": true})
				return
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Memory %q deleted", args[0])))
		},
	})

	root.AddCommand(memoryCmd)

	// ── skill ──────────────────────────────────────────────────────────────
	skillCmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage agent skills",
	}

	// ── skill list ──────────────────────────────────────────────────────
	skillCmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all skills",
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/skills%s", ep, nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			var resp SkillListResponse
			json.Unmarshal(data, &resp)
			if jsonMode {
				printJSON(resp)
				return
			}
			if len(resp.Skills) == 0 {
				fmt.Println(dimStyle.Render("No skills found."))
				return
			}
			fmt.Println(titleStyle.Render(fmt.Sprintf("  %d skill(s)  ", len(resp.Skills))))
			fmt.Println()
			nameW := len("NAME")
			descW := len("DESCRIPTION")
			for _, s := range resp.Skills {
				if len(s.Name) > nameW { nameW = len(s.Name) }
				d := s.Description
				if len(d) > 40 { d = d[:40] }
				if len(d) > descW { descW = len(d) }
			}
			header := fmt.Sprintf("  %-*s  %-*s  %s", nameW, "NAME", descW, "DESCRIPTION", "AGENTS")
			fmt.Println(dimStyle.Render(header))
			for _, s := range resp.Skills {
				desc := s.Description
				if len(desc) > 40 { desc = desc[:40] + "..." }
				fmt.Printf("  %-*s  %-*s  %d\n", nameW, s.Name, descW, desc, len(s.Agents))
			}
		},
	})

	// ── skill get ───────────────────────────────────────────────────────
	skillCmd.AddCommand(&cobra.Command{
		Use:   "get <name>",
		Short: "Get skill details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/skills/%s%s", ep, url.PathEscape(args[0]), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Skill %q not found", args[0]), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Skill %q not found", args[0])))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			var s SkillResponse
			json.Unmarshal(data, &s)
			if jsonMode {
				printJSON(s)
				return
			}
			fmt.Println(headerStyle.Render(fmt.Sprintf("  %s  ", s.Name)))
			fmt.Println()
			if s.Description != "" {
				fmt.Printf("  Description: %s\n", s.Description)
			}
			fmt.Printf("  Namespace:   %s\n", s.Namespace)
			fmt.Printf("  Agents:      %d\n", len(s.Agents))
			fmt.Printf("  Created:     %s\n", s.CreatedAt)
			fmt.Println()
			fmt.Println(dimStyle.Render("  ── Content ──"))
			fmt.Println()
			fmt.Println(s.Content)
		},
	})

	// ── skill create ────────────────────────────────────────────────────
	skillCreateCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new skill",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			content, _ := cmd.Flags().GetString("content")
			description, _ := cmd.Flags().GetString("description")
			if content == "" {
				if jsonMode {
					dieJSON("--content is required", 400)
				}
				fmt.Println(errorStyle.Render("--content is required"))
				os.Exit(1)
			}
			body := map[string]interface{}{
				"name":    args[0],
				"content": content,
			}
			if description != "" {
				body["description"] = description
			}
			if ns, _ := cmd.Flags().GetString("namespace"); ns != "" {
				body["namespace"] = ns
			}
			data, status, err := apiRequest("POST", ep+"/api/v1/skills", body)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 201 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			var s SkillResponse
			json.Unmarshal(data, &s)
			if jsonMode {
				printJSON(s)
				return
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Skill %q created", args[0])))
		},
	}
	skillCreateCmd.Flags().String("content", "", "Skill content (required)")
	skillCreateCmd.Flags().String("description", "", "Short description")
	skillCmd.AddCommand(skillCreateCmd)

	// ── skill edit ──────────────────────────────────────────────────────
	skillEditCmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a skill's content or description",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			body := map[string]interface{}{}
			if content, _ := cmd.Flags().GetString("content"); content != "" {
				body["content"] = content
			}
			if description, _ := cmd.Flags().GetString("description"); description != "" {
				body["description"] = description
			}
			if len(body) == 0 {
				if jsonMode {
					dieJSON("No changes provided. Use --content or --description flags.", 400)
				}
				fmt.Println(errorStyle.Render("No changes provided. Use --content or --description flags."))
				os.Exit(1)
			}
			data, status, err := apiRequest("PATCH", fmt.Sprintf("%s/api/v1/skills/%s%s", ep, url.PathEscape(args[0]), nsQuery(cmd)), body)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Skill %q not found", args[0]), 404)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("Skill %q not found", args[0])))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			var s SkillResponse
			json.Unmarshal(data, &s)
			if jsonMode {
				printJSON(s)
				return
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Skill %q updated", args[0])))
		},
	}
	skillEditCmd.Flags().String("content", "", "New skill content")
	skillEditCmd.Flags().String("description", "", "New description")
	skillCmd.AddCommand(skillEditCmd)

	// ── skill delete ────────────────────────────────────────────────────
	skillCmd.AddCommand(&cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a skill",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("DELETE", fmt.Sprintf("%s/api/v1/skills/%s%s", ep, url.PathEscape(args[0]), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				fmt.Println(errorStyle.Render("Request failed: " + err.Error()))
				os.Exit(1)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				fmt.Println(errorStyle.Render(fmt.Sprintf("API error (%d): %s", status, string(data))))
				os.Exit(1)
			}
			if jsonMode {
				printJSON(map[string]any{"name": args[0], "deleted": true})
				return
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Skill %q deleted", args[0])))
		},
	})

	root.AddCommand(skillCmd)

	root.Execute()
}
