package cli

import (
	"context"
	"fmt"

	"github.com/nathfavour/threader/internal/ai"
	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/internal/synthesis"
	"github.com/nathfavour/threader/internal/threads"
	"github.com/nathfavour/threader/pkg/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(postCmd)
	postCmd.AddCommand(postCraftCmd)
	postCmd.AddCommand(postPublishCmd)

	postCraftCmd.Flags().String("project", "", "Project ID")
	postCraftCmd.Flags().String("goal", "", "Goal for the post")
}

var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Manage Threads posts",
}

var postCraftCmd = &cobra.Command{
	Use:   "craft",
	Short: "Craft a new post using AI",
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		goal, _ := cmd.Flags().GetString("goal")

		if projectID == "" {
			fmt.Println("Error: --project flag is required")
			return
		}

		reg, _ := project.NewRegistry(config.ProjectsPath())
		p, ok := reg.Get(projectID)
		if !ok {
			fmt.Println("Project not found.")
			return
		}

		aiClient := ai.NewClient()
		synth := synthesis.NewSynthesizer(aiClient)

		// TODO: In a real scenario, we'd fetch media assets for this project
		resp, err := synth.CraftPost(context.Background(), p, nil, goal)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Println("--- Crafted Post ---")
		fmt.Println(resp)
	},
}

var postPublishCmd = &cobra.Command{
	Use:   "publish [text]",
	Short: "Publish a post to Threads",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		aiClient := ai.NewClient()
		token, err := aiClient.VaultGet("THREADS_ACCESS_TOKEN")
		if err != nil {
			fmt.Println("Error: THREADS_ACCESS_TOKEN not found in Vibe Vault.")
			fmt.Println("Run 'threader config set-token [token]' to configure.")
			return
		}

		client := threads.NewClient(token)
		id, err := client.CreateTextPost(args[0])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Published successfully! Post ID: %s\n", id)
	},
}
