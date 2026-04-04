package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func registerOfficeCommands(root *cobra.Command) {
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
}
