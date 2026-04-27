package main

import (
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "komputer",
		Short: titleStyle.Render("komputer-cli") + " — CLI for the komputer.ai platform",
	}

	root.PersistentFlags().String("api", "", "API endpoint URL (overrides login config)")
	root.PersistentFlags().StringP("namespace", "n", "", "Kubernetes namespace (default: server default)")
	root.PersistentFlags().Bool("json", false, "Output raw JSON instead of formatted text")

	registerAgentCommands(root)
	registerOfficeCommands(root)
	registerScheduleCommands(root)
	registerMemoryCommands(root)
	registerSkillCommands(root)
	registerConnectorCommands(root)
	registerSquadCommands(root)

	root.Execute()
}
