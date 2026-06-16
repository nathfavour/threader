package cli

import (
	"fmt"

	"github.com/nathfavour/threader/internal/ai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetTokenCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure threader settings",
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Run: func(cmd *cobra.Command, args []string) {
		settings := viper.AllSettings()
		if len(settings) == 0 {
			fmt.Println("No configuration values found.")
			return
		}
		for k, v := range settings {
			fmt.Printf("%s: %v\n", k, v)
		}
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		val := viper.Get(args[0])
		if val == nil {
			fmt.Printf("Key %q not found\n", args[0])
			return
		}
		fmt.Printf("%s: %v\n", args[0], val)
	},
}

var configSetTokenCmd = &cobra.Command{
	Use:   "set-token [token]",
	Short: "Set Threads Access Token in Vibe Vault",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		aiClient := ai.NewClient()
		err := aiClient.VaultSet("THREADS_ACCESS_TOKEN", args[0])
		if err != nil {
			fmt.Printf("Error setting token in vault: %v\n", err)
			return
		}
		fmt.Println("Threads Access Token saved to Vibe Vault.")
	},
}
