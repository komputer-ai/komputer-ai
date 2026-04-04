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
}
