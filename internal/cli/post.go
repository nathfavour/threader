package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

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

	postCraftCmd.Flags().StringP("project", "p", "", "Project ID or Name (defaults to first project)")
	postCraftCmd.Flags().String("goal", "", "Goal for the post")
	postCraftCmd.Flags().String("manual", "", "Manually provide post content")
	postCraftCmd.Flags().Lookup("manual").NoOptDefVal = "PROMPT"

	postPublishCmd.Flags().StringP("project", "p", "", "Project ID or Name (defaults to first project)")
}

var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Manage Threads posts",
}

var postCraftCmd = &cobra.Command{
	Use:   "craft",
	Short: "Craft a new post using AI or manual input",
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		goal, _ := cmd.Flags().GetString("goal")
		manual, _ := cmd.Flags().GetString("manual")

		reg, _ := project.NewRegistry(config.ProjectsPath())
		projects := reg.List()
		if len(projects) == 0 {
			fmt.Println("Error: No projects found. Run setup first.")
			return
		}

		var p *project.Project
		if projectID == "" {
			p = projects[0]
		} else {
			for _, proj := range projects {
				if proj.ID == projectID || proj.Name == projectID {
					p = proj
					break
				}
			}
		}

		if p == nil {
			fmt.Printf("Error: Project %q not found.\n", projectID)
			return
		}

		var postText string
		if cmd.Flags().Changed("manual") {
			if manual == "PROMPT" {
				fmt.Print("Enter post content: ")
				reader := bufio.NewReader(os.Stdin)
				postText, _ = reader.ReadString('\n')
				postText = strings.TrimSpace(postText)
			} else {
				postText = manual
			}
		} else {
			aiClient := ai.NewClient()
			synth := synthesis.NewSynthesizer(aiClient)

			if goal == "" {
				goal = "Create an engaging post about this project."
			}

			fmt.Printf("🧵 Crafting AI post for project %q...\n", p.Name)
			resp, err := synth.CraftPost(context.Background(), p, nil, goal)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			postText = resp
		}

		fmt.Println("\n--- Crafted Post ---")
		fmt.Println(postText)
		fmt.Println("--------------------")
		
		fmt.Print("Publish now? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		ans, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(ans)) == "y" {
			publishToProject(p, postText)
		}
	},
}

var postPublishCmd = &cobra.Command{
	Use:   "publish [text]",
	Short: "Publish a post to Threads",
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		
		reg, _ := project.NewRegistry(config.ProjectsPath())
		projects := reg.List()
		if len(projects) == 0 {
			fmt.Println("Error: No projects found.")
			return
		}

		var p *project.Project
		if projectID == "" {
			p = projects[0]
		} else {
			for _, proj := range projects {
				if proj.ID == projectID || proj.Name == projectID {
					p = proj
					break
				}
			}
		}

		if p == nil {
			fmt.Printf("Error: Project %q not found.\n", projectID)
			return
		}

		var text string
		if len(args) > 0 {
			text = args[0]
		} else {
			fmt.Print("Enter post content: ")
			reader := bufio.NewReader(os.Stdin)
			text, _ = reader.ReadString('\n')
			text = strings.TrimSpace(text)
		}

		if text == "" {
			fmt.Println("Error: Cannot publish empty post.")
			return
		}

		publishToProject(p, text)
	},
}

func publishToProject(p *project.Project, text string) {
	aiClient := ai.NewClient()
	token, err := aiClient.VaultGet(fmt.Sprintf("THREADS_TOKEN_%s", p.ID))
	if err != nil {
		fmt.Printf("Error: Threads token not found for project %q in Vibe Vault.\n", p.Name)
		return
	}

	client := threads.NewClient(token)
	id, err := client.CreateTextPost(text)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Published successfully to project %q! Post ID: %s\n", p.Name, id)
}
