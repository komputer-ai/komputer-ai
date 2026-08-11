package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func registerScheduleCommands(root *cobra.Command) {
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
			if sched.Instructions != "" {
				row("Instructions:", sched.Instructions)
			}

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
				// Create agent from template. Role and model are left empty when
				// unset — the API defaults them (role=worker) for schedules.
				role, _ := cmd.Flags().GetString("role")
				templateRef, _ := cmd.Flags().GetString("template")
				systemPrompt, _ := cmd.Flags().GetString("system-prompt")
				priority, _ := cmd.Flags().GetInt32("priority")
				cpu, _ := cmd.Flags().GetString("cpu")
				memLimit, _ := cmd.Flags().GetString("memory-limit")
				image, _ := cmd.Flags().GetString("image")
				storageSize, _ := cmd.Flags().GetString("storage")

				spec := &ScheduleAgentSpec{
					Model:        model,
					Lifecycle:    lifecycle,
					Role:         role,
					TemplateRef:  templateRef,
					SystemPrompt: systemPrompt,
					Priority:     priority,
				}
				spec.Secrets, _ = cmd.Flags().GetStringSlice("secret")
				spec.Skills, _ = cmd.Flags().GetStringSlice("skill")
				spec.Memories, _ = cmd.Flags().GetStringSlice("memory")
				spec.AllowedTools, _ = cmd.Flags().GetStringSlice("allow-tool")
				spec.DisallowedTools, _ = cmd.Flags().GetStringSlice("disallow-tool")
				if storageSize != "" {
					spec.Storage = map[string]string{"size": storageSize}
				}
				spec.PodSpec = buildPodSpecOverride(cpu, memLimit, image)

				if labelFlags, _ := cmd.Flags().GetStringArray("label"); len(labelFlags) > 0 {
					spec.Labels = parseLabelFlags(labelFlags)
				}

				body["agent"] = spec
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
	scheduleCreateCmd.Flags().String("template", "", "KomputerAgentTemplate name")
	scheduleCreateCmd.Flags().String("role", "", "Agent role: worker (default for schedules) or manager")
	scheduleCreateCmd.Flags().StringSlice("secret", nil, "Secret names to attach (repeatable)")
	scheduleCreateCmd.Flags().StringSlice("memory", nil, "Memory names to attach (repeatable, e.g. --memory k8s-debug)")
	scheduleCreateCmd.Flags().StringSlice("skill", nil, "Skill names to attach (repeatable, e.g. --skill python-expert)")
	scheduleCreateCmd.Flags().StringSlice("allow-tool", nil, "Restrict agent to these tools (repeatable). REPLACES the default tool set, so re-list the built-ins you still need, e.g. --allow-tool Read --allow-tool Grep --allow-tool 'mcp__figma__*'")
	scheduleCreateCmd.Flags().StringSlice("disallow-tool", nil, "Remove these tools, keeping all others (repeatable), e.g. --disallow-tool Bash --disallow-tool mcp__figma__use_figma")
	scheduleCreateCmd.Flags().String("system-prompt", "", "Custom system prompt for the agent")
	scheduleCreateCmd.Flags().Int32("priority", 0, "Queue priority (higher = admitted first when template cap is reached; default 0)")
	scheduleCreateCmd.Flags().String("cpu", "", "Override CPU (e.g. 2 or 500m). Sets both requests and limits.")
	scheduleCreateCmd.Flags().String("memory-limit", "", "Override memory (e.g. 4Gi). Sets both requests and limits.")
	scheduleCreateCmd.Flags().String("storage", "", "Override PVC storage size (e.g. 20Gi).")
	scheduleCreateCmd.Flags().String("image", "", "Override agent container image.")
	scheduleCreateCmd.Flags().StringArray("label", nil, "Label key=value (repeatable, e.g. --label team=core)")
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

	// ── schedule trigger ───────────────────────────────────────────────
	scheduleCmd.AddCommand(&cobra.Command{
		Use:   "trigger <name>",
		Short: "Trigger a schedule to run now (outside of its cron cadence)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			scheduleName := args[0]

			data, status, err := apiRequest("POST", fmt.Sprintf("%s/api/v1/schedules/%s/trigger%s", ep, url.PathEscape(scheduleName), nsQuery(cmd)), nil)
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
				printJSON(json.RawMessage(data))
				return
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Schedule %q triggered", scheduleName)))
		},
	})

	// ── schedule update ────────────────────────────────────────────────
	scheduleUpdateCmd := &cobra.Command{
		Use:     "update <name>",
		Aliases: []string{"edit", "patch"},
		Short:   "Update any editable field on a schedule",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			scheduleName := args[0]

			// PATCH replaces spec.agent wholesale, so read the schedule first and
			// overlay only what changed — otherwise updating one field wipes the rest.
			existingData, existingStatus, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/schedules/%s%s", ep, url.PathEscape(scheduleName), nsQuery(cmd)), nil)
			if err != nil || existingStatus != 200 {
				msg := fmt.Sprintf("failed to read schedule %q before update: %v", scheduleName, err)
				if err == nil {
					msg = fmt.Sprintf("failed to read schedule %q before update: API error (%d): %s", scheduleName, existingStatus, string(existingData))
				}
				if jsonMode {
					dieJSON(msg, existingStatus)
				}
				fmt.Println(errorStyle.Render(msg))
				os.Exit(1)
			}
			var existing struct {
				Agent map[string]interface{} `json:"agent"`
			}
			if jsonErr := json.Unmarshal(existingData, &existing); jsonErr != nil {
				existing.Agent = nil
			}

			body := buildScheduleUpdateBody(cmd, existing.Agent)

			if len(body) == 0 {
				msg := "no fields to update — pass at least one of --cron, --instructions, --timezone, --auto-delete, --keep-agents, --suspended, --agent, --model, --lifecycle, --role, --template, --secret, --skill, --memory, --allow-tool, --disallow-tool, --system-prompt, --priority, --cpu, --memory-limit, --storage, --image"
				if jsonMode {
					dieJSON(msg, 400)
				}
				fmt.Println(errorStyle.Render(msg))
				os.Exit(1)
			}

			data, status, err := apiRequest("PATCH", fmt.Sprintf("%s/api/v1/schedules/%s%s", ep, url.PathEscape(scheduleName), nsQuery(cmd)), body)
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
			var sched ScheduleResponse
			json.Unmarshal(data, &sched)
			if jsonMode {
				printJSON(sched)
				return
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Schedule %q updated", scheduleName)))
		},
	}
	scheduleUpdateCmd.Flags().String("cron", "", "New cron expression")
	scheduleUpdateCmd.Flags().String("instructions", "", "New instructions for the agent")
	scheduleUpdateCmd.Flags().String("timezone", "", "New IANA timezone")
	scheduleUpdateCmd.Flags().Bool("auto-delete", false, "Toggle auto-delete after first successful run")
	scheduleUpdateCmd.Flags().Bool("keep-agents", false, "Toggle keep-agents on auto-delete")
	scheduleUpdateCmd.Flags().Bool("suspended", false, "Toggle suspended state (paused without deletion)")
	scheduleUpdateCmd.Flags().String("agent", "", "Target an existing agent by name (clears inline template)")
	scheduleUpdateCmd.Flags().String("model", "", "Agent template: Claude model")
	scheduleUpdateCmd.Flags().String("lifecycle", "", "Agent template: lifecycle (Sleep, AutoDelete, or empty)")
	scheduleUpdateCmd.Flags().String("role", "", "Agent template: role")
	scheduleUpdateCmd.Flags().String("template-ref", "", "Agent template: KomputerAgentTemplate ref")
	// `schedule update` shipped with --template-ref while `agents create` uses
	// --template; keep the old spelling working while the two converge.
	scheduleUpdateCmd.Flags().String("template", "", "KomputerAgentTemplate name")
	_ = scheduleUpdateCmd.Flags().MarkDeprecated("template-ref", "use --template instead")
	scheduleUpdateCmd.Flags().StringSlice("secret", nil, "Secret names to attach (repeatable)")
	scheduleUpdateCmd.Flags().StringSlice("memory", nil, "Memory names to attach (repeatable, e.g. --memory k8s-debug)")
	scheduleUpdateCmd.Flags().StringSlice("skill", nil, "Skill names to attach (repeatable, e.g. --skill python-expert)")
	scheduleUpdateCmd.Flags().StringSlice("allow-tool", nil, "Restrict agent to these tools (repeatable). REPLACES the default tool set, so re-list the built-ins you still need, e.g. --allow-tool Read --allow-tool Grep --allow-tool 'mcp__figma__*'")
	scheduleUpdateCmd.Flags().StringSlice("disallow-tool", nil, "Remove these tools, keeping all others (repeatable), e.g. --disallow-tool Bash --disallow-tool mcp__figma__use_figma")
	scheduleUpdateCmd.Flags().String("system-prompt", "", "Custom system prompt for the agent")
	scheduleUpdateCmd.Flags().Int32("priority", 0, "Queue priority (higher = admitted first when template cap is reached; default 0)")
	scheduleUpdateCmd.Flags().String("cpu", "", "Override CPU (e.g. 2 or 500m). Sets both requests and limits.")
	scheduleUpdateCmd.Flags().String("memory-limit", "", "Override memory (e.g. 4Gi). Sets both requests and limits.")
	scheduleUpdateCmd.Flags().String("storage", "", "Override PVC storage size (e.g. 20Gi).")
	scheduleUpdateCmd.Flags().String("image", "", "Override agent container image.")
	scheduleCmd.AddCommand(scheduleUpdateCmd)

	root.AddCommand(scheduleCmd)
}

// buildScheduleUpdateBody assembles the PATCH body for `schedule update` from
// the flags the caller changed.
//
// existingAgent is the schedule's current spec.agent as decoded from the API,
// or nil when the schedule targets an agent by name instead. PATCH replaces
// spec.agent wholesale rather than merging field by field, so the "agent" value
// has to be the complete object: the changed flags are overlaid on top of
// existingAgent, and podSpec/storage are merged into rather than replaced so
// sibling overrides (image, memory, storageClassName) survive.
//
// The "agent" key is omitted entirely unless an agent flag actually changed.
// Re-sending it on, say, a --cron-only update would be more than noise: the API
// treats a non-nil agent as a switch to an inline template and clears
// spec.agentName.
func buildScheduleUpdateBody(cmd *cobra.Command, existingAgent map[string]interface{}) map[string]interface{} {
	body := map[string]interface{}{}
	if cmd.Flags().Changed("cron") {
		cron, _ := cmd.Flags().GetString("cron")
		body["schedule"] = cron
	}
	if cmd.Flags().Changed("instructions") {
		instructions, _ := cmd.Flags().GetString("instructions")
		body["instructions"] = instructions
	}
	if cmd.Flags().Changed("timezone") {
		tz, _ := cmd.Flags().GetString("timezone")
		body["timezone"] = tz
	}
	if cmd.Flags().Changed("auto-delete") {
		v, _ := cmd.Flags().GetBool("auto-delete")
		body["autoDelete"] = v
	}
	if cmd.Flags().Changed("keep-agents") {
		v, _ := cmd.Flags().GetBool("keep-agents")
		body["keepAgents"] = v
	}
	if cmd.Flags().Changed("suspended") {
		v, _ := cmd.Flags().GetBool("suspended")
		body["suspended"] = v
	}
	if cmd.Flags().Changed("agent") {
		agent, _ := cmd.Flags().GetString("agent")
		body["agentName"] = agent
	}

	agentSpec := existingAgent
	if agentSpec == nil {
		agentSpec = map[string]interface{}{}
	}
	// agentSpec is pre-populated from the server, so its length says nothing
	// about whether the caller asked for an agent change.
	agentChanged := false
	for _, f := range []struct{ flag, key string }{
		{"model", "model"},
		{"lifecycle", "lifecycle"},
		{"role", "role"},
		{"template-ref", "templateRef"},
		{"template", "templateRef"},
		{"system-prompt", "systemPrompt"},
	} {
		if cmd.Flags().Changed(f.flag) {
			v, _ := cmd.Flags().GetString(f.flag)
			agentSpec[f.key] = v
			agentChanged = true
		}
	}
	for _, f := range []struct{ flag, key string }{
		{"secret", "secrets"},
		{"skill", "skills"},
		{"memory", "memories"},
		{"allow-tool", "allowedTools"},
		{"disallow-tool", "disallowedTools"},
	} {
		if cmd.Flags().Changed(f.flag) {
			v, _ := cmd.Flags().GetStringSlice(f.flag)
			agentSpec[f.key] = v
			agentChanged = true
		}
	}
	if cmd.Flags().Changed("priority") {
		v, _ := cmd.Flags().GetInt32("priority")
		agentSpec["priority"] = v
		agentChanged = true
	}
	if cmd.Flags().Changed("storage") {
		v, _ := cmd.Flags().GetString("storage")
		// Set only the size so storageClassName is not dropped.
		if existing, ok := agentSpec["storage"].(map[string]interface{}); ok {
			existing["size"] = v
		} else {
			agentSpec["storage"] = map[string]interface{}{"size": v}
		}
		agentChanged = true
	}
	if cmd.Flags().Changed("cpu") || cmd.Flags().Changed("memory-limit") || cmd.Flags().Changed("image") {
		cpu, _ := cmd.Flags().GetString("cpu")
		memLimit, _ := cmd.Flags().GetString("memory-limit")
		image, _ := cmd.Flags().GetString("image")
		if ps := mergePodSpecOverride(agentSpec["podSpec"], cpu, memLimit, image); ps != nil {
			agentSpec["podSpec"] = ps
			agentChanged = true
		}
	}
	if agentChanged {
		body["agent"] = agentSpec
	}
	return body
}

// mergePodSpecOverride overlays the cpu/memory/image overrides onto an existing
// podSpec, setting only the fields the caller supplied so the others survive —
// `--cpu 8` must not drop an image or memory override set earlier.
//
// existing is the raw podSpec decoded from the API, so its nested values are
// map[string]interface{} / []interface{}. Anything that is not the shape we
// know how to merge into falls back to building a fresh podSpec, which is the
// replace-everything behavior. Returns nil when there is nothing to set.
func mergePodSpecOverride(existing interface{}, cpu, memory, image string) map[string]interface{} {
	if cpu == "" && memory == "" && image == "" {
		return nil
	}
	podSpec, ok := existing.(map[string]interface{})
	if !ok {
		return buildPodSpecOverride(cpu, memory, image)
	}
	containers, ok := podSpec["containers"].([]interface{})
	if !ok {
		return buildPodSpecOverride(cpu, memory, image)
	}
	var agentContainer map[string]interface{}
	for _, c := range containers {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if name, _ := container["name"].(string); name == "agent" {
			agentContainer = container
			break
		}
	}
	if agentContainer == nil {
		return buildPodSpecOverride(cpu, memory, image)
	}

	if image != "" {
		agentContainer["image"] = image
	}
	if cpu != "" || memory != "" {
		resources, ok := agentContainer["resources"].(map[string]interface{})
		if !ok {
			resources = map[string]interface{}{}
			agentContainer["resources"] = resources
		}
		for _, key := range []string{"requests", "limits"} {
			quantities, ok := resources[key].(map[string]interface{})
			if !ok {
				quantities = map[string]interface{}{}
				resources[key] = quantities
			}
			if cpu != "" {
				quantities["cpu"] = cpu
			}
			if memory != "" {
				quantities["memory"] = memory
			}
		}
	}
	return podSpec
}
