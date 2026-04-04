package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

func registerSkillCommands(root *cobra.Command) {
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
}
