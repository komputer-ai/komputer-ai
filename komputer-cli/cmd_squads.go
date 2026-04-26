package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func registerSquadCommands(root *cobra.Command) {
	// ── squad (parent) ──────────────────────────────────────────────────
	squadCmd := &cobra.Command{
		Use:   "squad",
		Short: "Manage agent squads (shared workspace groups)",
	}

	// ── squad list ──────────────────────────────────────────────────────
	squadCmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all squads",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			data, status, err := apiRequest("GET", ep+"/api/v1/squads"+nsQuery(cmd), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				return fmt.Errorf("request failed: %w", err)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				return fmt.Errorf("API error (%d): %s", status, string(data))
			}

			var resp SquadListResponse
			json.Unmarshal(data, &resp)

			if jsonMode {
				printJSON(resp)
				return nil
			}

			if len(resp.Squads) == 0 {
				fmt.Println(dimStyle.Render("No squads found."))
				return nil
			}

			fmt.Println(titleStyle.Render(fmt.Sprintf("  %d squad(s)  ", len(resp.Squads))))
			fmt.Println()

			// Compute dynamic column widths.
			nameW := len("NAME")
			for _, s := range resp.Squads {
				if len(s.Name) > nameW {
					nameW = len(s.Name)
				}
			}
			nameW += 2
			totalW := nameW + 10 + 22 + 16

			fmt.Printf("  %s  %s  %s  %s\n",
				labelStyle.Render(fmt.Sprintf("%-*s", nameW, "NAME")),
				labelStyle.Render(fmt.Sprintf("%-8s", "MEMBERS")),
				labelStyle.Render(fmt.Sprintf("%-20s", "CREATED")),
				labelStyle.Render(fmt.Sprintf("%-14s", "ORPHAN TTL")),
			)
			fmt.Println(dimStyle.Render("  " + strings.Repeat("─", totalW)))

			for _, s := range resp.Squads {
				orphanTTL := s.OrphanTTL
				if orphanTTL == "" {
					orphanTTL = "—"
				}
				fmt.Printf("  %s  %s  %s  %s\n",
					valueStyle.Render(fmt.Sprintf("%-*s", nameW, s.Name)),
					valueStyle.Render(fmt.Sprintf("%-8d", len(s.Members))),
					dimStyle.Render(fmt.Sprintf("%-20s", s.CreatedAt)),
					dimStyle.Render(fmt.Sprintf("%-14s", orphanTTL)),
				)
			}
			fmt.Println()
			return nil
		},
	})

	// ── squad get ───────────────────────────────────────────────────────
	squadGetCmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Get squad details and members",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			squadName := args[0]

			data, status, err := apiRequest("GET", fmt.Sprintf("%s/api/v1/squads/%s%s", ep, url.PathEscape(squadName), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				return fmt.Errorf("request failed: %w", err)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Squad %q not found", squadName), 404)
				}
				return fmt.Errorf("squad %q not found", squadName)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				return fmt.Errorf("API error (%d): %s", status, string(data))
			}

			var squad SquadResponse
			json.Unmarshal(data, &squad)

			if jsonMode {
				printJSON(squad)
				return nil
			}

			// Squad header
			fmt.Println(headerStyle.Render(fmt.Sprintf("  %s  ", squad.Name)))

			row := func(label, value string) {
				fmt.Printf("  %s %s\n", labelStyle.Render(fmt.Sprintf("%-16s", label)), valueStyle.Render(value))
			}

			row("Namespace:", squad.Namespace)
			row("Members:", fmt.Sprintf("%d", len(squad.Members)))
			if squad.OrphanTTL != "" {
				row("Orphan TTL:", squad.OrphanTTL)
			}
			row("Created:", squad.CreatedAt)
			fmt.Println()

			// Members table
			if len(squad.Members) > 0 {
				fmt.Println(labelStyle.Render("  Members"))

				memberNameW := len("NAME")
				memberNsW := len("NAMESPACE")
				for _, m := range squad.Members {
					if len(m.Name) > memberNameW {
						memberNameW = len(m.Name)
					}
					if len(m.Namespace) > memberNsW {
						memberNsW = len(m.Namespace)
					}
				}
				memberNameW += 2
				memberNsW += 2
				memberTotalW := memberNameW + memberNsW

				fmt.Println(dimStyle.Render("  " + strings.Repeat("─", memberTotalW)))
				fmt.Printf("  %s  %s\n",
					labelStyle.Render(fmt.Sprintf("%-*s", memberNameW, "NAME")),
					labelStyle.Render(fmt.Sprintf("%-*s", memberNsW, "NAMESPACE")),
				)

				for _, m := range squad.Members {
					fmt.Printf("  %s  %s\n",
						valueStyle.Render(fmt.Sprintf("%-*s", memberNameW, m.Name)),
						dimStyle.Render(fmt.Sprintf("%-*s", memberNsW, m.Namespace)),
					)
				}
				fmt.Println()
			}
			return nil
		},
	}
	squadCmd.AddCommand(squadGetCmd)

	// ── squad create ────────────────────────────────────────────────────
	squadCreateCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new squad",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			ns, _ := cmd.Flags().GetString("namespace")
			agentNames, _ := cmd.Flags().GetStringSlice("agents")
			orphanTTL, _ := cmd.Flags().GetString("orphan-ttl")

			body := map[string]interface{}{
				"name": args[0],
			}
			if ns != "" {
				body["namespace"] = ns
			}
			if orphanTTL != "" {
				body["orphanTTL"] = orphanTTL
			}

			// Build members list from agent name refs.
			if len(agentNames) > 0 {
				members := make([]map[string]interface{}, 0, len(agentNames))
				for _, agentName := range agentNames {
					agentName = strings.TrimSpace(agentName)
					if agentName == "" {
						continue
					}
					ref := map[string]interface{}{"name": agentName}
					if ns != "" {
						ref["namespace"] = ns
					}
					members = append(members, map[string]interface{}{"ref": ref})
				}
				if len(members) > 0 {
					body["members"] = members
				}
			}

			data, status, err := apiRequest("POST", ep+"/api/v1/squads", body)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				return fmt.Errorf("request failed: %w", err)
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
				return fmt.Errorf("API error (%d): %s", status, string(data))
			}

			var squad SquadResponse
			json.Unmarshal(data, &squad)

			if jsonMode {
				printJSON(squad)
				return nil
			}

			fmt.Println(successStyle.Render("✔ Squad created"))
			printSquad(squad)
			return nil
		},
	}
	squadCreateCmd.Flags().StringSlice("agents", nil, "Comma-separated list of existing agent names to include in the squad")
	squadCreateCmd.Flags().String("orphan-ttl", "", "TTL after which an empty squad is deleted (e.g. 10m)")
	squadCmd.AddCommand(squadCreateCmd)

	// ── squad add ───────────────────────────────────────────────────────
	squadAddCmd := &cobra.Command{
		Use:   "add <squad> <agent>",
		Short: "Add an agent to a squad",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			squadName := args[0]
			agentName := args[1]
			ns, _ := cmd.Flags().GetString("namespace")

			ref := map[string]interface{}{"name": agentName}
			if ns != "" {
				ref["namespace"] = ns
			}
			body := map[string]interface{}{"ref": ref}

			data, status, err := apiRequest("POST", fmt.Sprintf("%s/api/v1/squads/%s/members%s", ep, url.PathEscape(squadName), nsQuery(cmd)), body)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				return fmt.Errorf("request failed: %w", err)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Squad %q not found", squadName), 404)
				}
				return fmt.Errorf("squad %q not found", squadName)
			}
			if status != 200 && status != 201 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				return fmt.Errorf("API error (%d): %s", status, string(data))
			}

			var squad SquadResponse
			json.Unmarshal(data, &squad)

			if jsonMode {
				printJSON(squad)
				return nil
			}

			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Agent %q added to squad %q", agentName, squadName)))
			printSquad(squad)
			return nil
		},
	}
	squadCmd.AddCommand(squadAddCmd)

	// ── squad remove ────────────────────────────────────────────────────
	squadRemoveCmd := &cobra.Command{
		Use:   "remove <squad> <agent>",
		Short: "Remove an agent from a squad",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			squadName := args[0]
			agentName := args[1]

			data, status, err := apiRequest("DELETE", fmt.Sprintf("%s/api/v1/squads/%s/members/%s%s", ep, url.PathEscape(squadName), url.PathEscape(agentName), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				return fmt.Errorf("request failed: %w", err)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Squad %q or agent %q not found", squadName, agentName), 404)
				}
				return fmt.Errorf("squad %q or agent %q not found", squadName, agentName)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				return fmt.Errorf("API error (%d): %s", status, string(data))
			}

			var squad SquadResponse
			json.Unmarshal(data, &squad)

			if jsonMode {
				printJSON(squad)
				return nil
			}

			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Agent %q removed from squad %q", agentName, squadName)))
			printSquad(squad)
			return nil
		},
	}
	squadCmd.AddCommand(squadRemoveCmd)

	// ── squad delete ────────────────────────────────────────────────────
	squadCmd.AddCommand(&cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a squad",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			squadName := args[0]

			data, status, err := apiRequest("DELETE", fmt.Sprintf("%s/api/v1/squads/%s%s", ep, url.PathEscape(squadName), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				return fmt.Errorf("request failed: %w", err)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Squad %q not found", squadName), 404)
				}
				return fmt.Errorf("squad %q not found", squadName)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				return fmt.Errorf("API error (%d): %s", status, string(data))
			}

			if jsonMode {
				printJSON(map[string]any{"name": squadName, "deleted": true})
				return nil
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Squad %q deleted", squadName)))
			return nil
		},
	})

	// ── squad break-up ──────────────────────────────────────────────────
	squadCmd.AddCommand(&cobra.Command{
		Use:   "break-up <name>",
		Short: "Request squad break-up (dissolves squad once all members are sleeping)",
		Long: "Marks the squad for dissolution. Once every member is asleep, " +
			"the operator deletes the squad and members fall back to solo agents. " +
			"Workspaces (PVCs) are preserved. Sending tasks to sleeping members " +
			"in the meantime is allowed — they wake, run, return to Sleeping, and " +
			"the break-up eventually completes.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonMode, _ := cmd.Flags().GetBool("json")
			ep := resolveEndpoint(cmd)
			squadName := args[0]

			data, status, err := apiRequest("POST", fmt.Sprintf("%s/api/v1/squads/%s/break-up%s", ep, url.PathEscape(squadName), nsQuery(cmd)), nil)
			if err != nil {
				if jsonMode {
					dieJSON("Request failed: "+err.Error(), 0)
				}
				return fmt.Errorf("request failed: %w", err)
			}
			if status == 404 {
				if jsonMode {
					dieJSON(fmt.Sprintf("Squad %q not found", squadName), 404)
				}
				return fmt.Errorf("squad %q not found", squadName)
			}
			if status != 200 {
				if jsonMode {
					dieJSON(fmt.Sprintf("API error (%d): %s", status, string(data)), status)
				}
				return fmt.Errorf("API error (%d): %s", status, string(data))
			}

			var squad SquadResponse
			json.Unmarshal(data, &squad)

			if jsonMode {
				printJSON(squad)
				return nil
			}
			fmt.Println(successStyle.Render(fmt.Sprintf("✔ Break-up requested for squad %q — will dissolve once all members are sleeping", squadName)))
			printSquad(squad)
			return nil
		},
	})

	root.AddCommand(squadCmd)
}

// printSquad renders a single squad's key fields.
func printSquad(s SquadResponse) {
	fmt.Println(headerStyle.Render(fmt.Sprintf("  %s  ", s.Name)))

	row := func(label, value string) {
		fmt.Printf("  %s %s\n", labelStyle.Render(fmt.Sprintf("%-16s", label)), valueStyle.Render(value))
	}

	row("Namespace:", s.Namespace)
	row("Members:", fmt.Sprintf("%d", len(s.Members)))
	if s.OrphanTTL != "" {
		row("Orphan TTL:", s.OrphanTTL)
	}
	if s.CreatedAt != "" {
		row("Created:", s.CreatedAt)
	}

	if len(s.Members) > 0 {
		fmt.Println()
		fmt.Println(labelStyle.Render("  Members:"))
		for _, m := range s.Members {
			ns := m.Namespace
			if ns == "" {
				ns = "default"
			}
			fmt.Printf("    %s %s\n", valueStyle.Render(m.Name), dimStyle.Render("("+ns+")"))
		}
	}
	fmt.Println()
}
