package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

func registerMemoryCommands(root *cobra.Command) {
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
}
