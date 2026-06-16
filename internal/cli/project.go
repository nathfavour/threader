package cli

import (
	"fmt"

	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/pkg/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)

	projectCreateCmd.Flags().String("desc", "", "Project description")
	projectCreateCmd.Flags().String("voice", "", "Brand voice")
}

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage marketing projects",
}

var projectCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new project namespace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		reg, _ := project.NewRegistry(config.ProjectsPath())
		desc, _ := cmd.Flags().GetString("desc")
		voice, _ := cmd.Flags().GetString("voice")

		p, err := reg.Register(args[0], desc, voice)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Created project: %s (ID: %s)\n", p.Name, p.ID)
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Run: func(cmd *cobra.Command, args []string) {
		reg, _ := project.NewRegistry(config.ProjectsPath())
		projects := reg.List()
		if len(projects) == 0 {
			fmt.Println("No projects found.")
			return
		}
		fmt.Println("Projects:")
		for _, p := range projects {
			fmt.Printf("- %s (%s)\n", p.Name, p.ID)
		}
	},
}
