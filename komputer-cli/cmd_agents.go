package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/chzyer/readline"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

func registerAgentCommands(root *cobra.Command) {
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
			if sp, _ := cmd.Flags().GetString("system-prompt"); sp != "" {
				body["systemPrompt"] = sp
			}
			cpu, _ := cmd.Flags().GetString("cpu")
			memLimit, _ := cmd.Flags().GetString("memory-limit")
			image, _ := cmd.Flags().GetString("image")
			storage, _ := cmd.Flags().GetString("storage")
			if ps := buildPodSpecOverride(cpu, memLimit, image); ps != nil {
				body["podSpec"] = ps
			}
			if storage != "" {
				body["storage"] = map[string]interface{}{"size": storage}
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
	createCmd.Flags().String("system-prompt", "", "Custom system prompt for the agent")
	createCmd.Flags().String("cpu", "", "Override CPU (e.g. 2 or 500m). Sets both requests and limits.")
	createCmd.Flags().String("memory-limit", "", "Override memory (e.g. 4Gi). Sets both requests and limits.")
	createCmd.Flags().String("storage", "", "Override PVC storage size (e.g. 20Gi).")
	createCmd.Flags().String("image", "", "Override agent container image.")
	root.AddCommand(createCmd)

	// ── update ───────────────────────────────────────────────────────────
	updateCmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update an agent's spec (model, instructions, cpu, memory, storage, image)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			body := map[string]interface{}{}

			if cmd.Flags().Changed("model") {
				v, _ := cmd.Flags().GetString("model")
				body["model"] = v
			}
			if cmd.Flags().Changed("instructions") {
				v, _ := cmd.Flags().GetString("instructions")
				body["instructions"] = v
			}
			if cmd.Flags().Changed("lifecycle") {
				v, _ := cmd.Flags().GetString("lifecycle")
				body["lifecycle"] = v
			}
			if cmd.Flags().Changed("system-prompt") {
				v, _ := cmd.Flags().GetString("system-prompt")
				body["systemPrompt"] = v
			}
			if cmd.Flags().Changed("storage") {
				v, _ := cmd.Flags().GetString("storage")
				body["storage"] = map[string]interface{}{"size": v}
			}
			cpu, _ := cmd.Flags().GetString("cpu")
			memLimit, _ := cmd.Flags().GetString("memory-limit")
			image, _ := cmd.Flags().GetString("image")
			if cmd.Flags().Changed("cpu") || cmd.Flags().Changed("memory-limit") || cmd.Flags().Changed("image") {
				if ps := buildPodSpecOverride(cpu, memLimit, image); ps != nil {
					body["podSpec"] = ps
				}
			}

			if len(body) == 0 {
				if jsonMode {
					dieJSON("No fields provided. Use --model, --instructions, --cpu, --memory-limit, --storage, or --image.", 400)
				}
				fmt.Println(errorStyle.Render("No fields provided. Use --model, --instructions, --cpu, --memory-limit, --storage, or --image."))
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
			fmt.Println(successStyle.Render("✔ Agent updated"))
			printAgent(agent)
		},
	}
	updateCmd.Flags().String("model", "", "Claude model (e.g. claude-opus-4-6)")
	updateCmd.Flags().String("instructions", "", "New task instructions")
	updateCmd.Flags().String("lifecycle", "", "Lifecycle: Sleep or AutoDelete (empty for default)")
	updateCmd.Flags().String("system-prompt", "", "Custom system prompt (use empty string to clear)")
	updateCmd.Flags().String("cpu", "", "Override CPU (e.g. 2 or 500m). Sets both requests and limits.")
	updateCmd.Flags().String("memory-limit", "", "Override memory (e.g. 4Gi). Sets both requests and limits.")
	updateCmd.Flags().String("storage", "", "Override PVC storage size (e.g. 20Gi).")
	updateCmd.Flags().String("image", "", "Override agent container image.")
	root.AddCommand(updateCmd)

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
			if sp, _ := cmd.Flags().GetString("system-prompt"); cmd.Flags().Changed("system-prompt") {
				body["systemPrompt"] = sp
			}

			if len(body) == 0 {
				if jsonMode {
					dieJSON("No settings provided. Use --model, --lifecycle, --secret, --memory, --skill, or --system-prompt flags.", 400)
				}
				fmt.Println(errorStyle.Render("No settings provided. Use --model, --lifecycle, --secret, --memory, --skill, or --system-prompt flags."))
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
	configCmd.Flags().String("system-prompt", "", "Custom system prompt (use empty string to clear)")
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

			// Prefetch event history via REST before opening the WebSocket.
			historyEvents, seen := fetchEventHistory(ep, agentName, 200, nsQueryAmp(cmd))
			for _, e := range historyEvents {
				if formatted := formatEvent(e); formatted != "" {
					fmt.Println(formatted)
					fmt.Println()
				}
			}

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

				// Skip events already shown from history.
				if eventSeen(seen, event) {
					continue
				}
				seen[event.Timestamp+":"+event.Type] = struct{}{}

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
			if sp, _ := cmd.Flags().GetString("system-prompt"); sp != "" {
				body["systemPrompt"] = sp
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

			// Prefetch event history via REST now that the pod is up.
			historyEvents, seen := fetchEventHistory(ep, agentName, 200, nsQueryAmp(cmd))
			for _, e := range historyEvents {
				if formatted := formatEvent(e); formatted != "" {
					fmt.Println(formatted)
					fmt.Println()
				}
			}

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

				// Skip events already shown from history.
				if eventSeen(seen, event) {
					continue
				}
				seen[event.Timestamp+":"+event.Type] = struct{}{}

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
	runCmd.Flags().String("system-prompt", "", "Custom system prompt to inject into the agent")
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
}

// buildPodSpecOverride constructs a minimal podSpec map for the API from cpu/memory/image overrides.
// CPU and memory are set on both requests and limits. Returns nil if no overrides are provided.
func buildPodSpecOverride(cpu, memory, image string) map[string]interface{} {
	if cpu == "" && memory == "" && image == "" {
		return nil
	}
	container := map[string]interface{}{"name": "agent"}
	if image != "" {
		container["image"] = image
	}
	if cpu != "" || memory != "" {
		rl := map[string]interface{}{}
		if cpu != "" {
			rl["cpu"] = cpu
		}
		if memory != "" {
			rl["memory"] = memory
		}
		container["resources"] = map[string]interface{}{
			"requests": rl,
			"limits":   rl,
		}
	}
	return map[string]interface{}{
		"containers": []interface{}{container},
	}
}
