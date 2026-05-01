package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

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
	case "Queued":
		phaseBadge = warnStyle.Render(fmt.Sprintf("Queued #%d", a.QueuePosition))
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
	if len(a.Labels) > 0 {
		keys := make([]string, 0, len(a.Labels))
		for k := range a.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+"="+a.Labels[k])
		}
		row("Labels:", strings.Join(parts, ", "))
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

// fetchEventHistory fetches up to limit events from the REST API and returns
// them along with a dedup set of "timestamp:normType" keys. normType maps
// "task_started" → "user_message" so WS duplicates are correctly suppressed.
func fetchEventHistory(ep, agentName string, limit int, nsQ string) ([]AgentEvent, map[string]struct{}) {
	seen := make(map[string]struct{})
	eventsURL := fmt.Sprintf("%s/api/v1/agents/%s/events?limit=%d%s",
		ep, url.PathEscape(agentName), limit, nsQ)
	data, status, err := apiRequest("GET", eventsURL, nil)
	if err != nil || status != 200 {
		return nil, seen
	}
	var resp struct {
		Agent  string       `json:"agent"`
		Events []AgentEvent `json:"events"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, seen
	}
	for _, e := range resp.Events {
		normType := e.Type
		if normType == "task_started" {
			normType = "user_message"
		}
		seen[e.Timestamp+":"+normType] = struct{}{}
	}
	return resp.Events, seen
}

// eventSeen returns true if the event is already in the seen set (dedup by timestamp+normType).
func eventSeen(seen map[string]struct{}, e AgentEvent) bool {
	normType := e.Type
	if normType == "task_started" {
		normType = "user_message"
	}
	_, ok := seen[e.Timestamp+":"+normType]
	return ok
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
