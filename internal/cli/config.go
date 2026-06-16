package cli

import (
	"fmt"

	"github.com/nathfavour/threader/internal/ai"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetTokenCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure threader settings",
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
