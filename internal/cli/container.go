package cli

import (
	"fmt"

	"github.com/nathfavour/threader/internal/container"
	"github.com/nathfavour/threader/pkg/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(containerCmd)
	containerCmd.AddCommand(containerListCmd)
	containerCmd.AddCommand(containerCreateCmd)
	containerCmd.AddCommand(containerUseCmd)
}

var containerCmd = &cobra.Command{
	Use:   "container",
	Short: "Manage Threads personality containers",
}

var containerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all containers",
	Run: func(cmd *cobra.Command, args []string) {
		m := container.NewManager(config.DataDir())
		list, _ := m.List()
		if len(list) == 0 {
			fmt.Println("No containers found.")
			return
		}
		for _, c := range list {
			def := ""
			if c.IsDefault {
				def = " (default)"
			}
			fmt.Printf("- %s: %s%s\n", c.Name, c.Description, def)
		}
	},
}

var containerCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new container",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		m := container.NewManager(config.DataDir())
		desc, _ := cmd.Flags().GetString("desc")
		c, err := m.Create(args[0], desc)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Created container: %s\n", c.Name)
	},
}

var containerUseCmd = &cobra.Command{
	Use:   "use [name]",
	Short: "Set a container as default",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		m := container.NewManager(config.DataDir())
		err := m.SetDefault(args[0])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Set %q as default container.\n", args[0])
	},
}
