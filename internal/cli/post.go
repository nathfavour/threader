package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nathfavour/threader/internal/ai"
	"github.com/nathfavour/threader/internal/media"
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
	postCraftCmd.Flags().String("media", "", "Path to media file (image/video)")

	postPublishCmd.Flags().StringP("project", "p", "", "Project ID or Name (defaults to first project)")
	postPublishCmd.Flags().String("media", "", "Path to media file")
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
		mediaPath, _ := cmd.Flags().GetString("media")

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

		var asset *media.Asset
		if mediaPath != "" {
			// Resolve absolute path
			absPath, err := filepath.Abs(mediaPath)
			if err != nil {
				fmt.Printf("Error resolving media path: %v\n", err)
				return
			}
			
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				fmt.Printf("Error: Media file %q does not exist.\n", absPath)
				return
			}

			fmt.Printf("🧵 Indexing media %q...\n", filepath.Base(absPath))
			engine := media.NewEngine(config.MediaDir())
			asset, err = engine.IndexMedia(p.ID, absPath)
			if err != nil {
				fmt.Printf("Warning: Media indexing failed: %v\n", err)
			}
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

			var assets []*media.Asset
			if asset != nil {
				assets = append(assets, asset)
			}

			fmt.Printf("🧵 Crafting AI post for project %q...\n", p.Name)
			cta, _ := reg.RotateCTA(p.ID)
			resp, err := synth.CraftPost(context.Background(), p, assets, goal, cta, nil)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			postText = resp
		}

		fmt.Println("\n--- Crafted Post ---")
		if asset != nil {
			fmt.Printf("[Media: %s]\n", asset.FilePath)
		}
		fmt.Println(postText)
		fmt.Println("--------------------")
		
		fmt.Print("Publish now? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		ans, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(ans)) == "y" {
			publishToProject(p, postText, asset)
		}
	},
}

var postPublishCmd = &cobra.Command{
	Use:   "publish [text]",
	Short: "Publish a post to Threads",
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		mediaPath, _ := cmd.Flags().GetString("media")
		
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

		var asset *media.Asset
		if mediaPath != "" {
			absPath, _ := filepath.Abs(mediaPath)
			engine := media.NewEngine(config.MediaDir())
			asset, _ = engine.IndexMedia(p.ID, absPath)
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

		publishToProject(p, text, asset)
	},
}

func publishToProject(p *project.Project, text string, asset *media.Asset) {
	if p.AccessToken == "" {
		fmt.Printf("Error: Threads token not found for project %q.\n", p.Name)
		return
	}

	client := threads.NewClient(p.AccessToken)
	
	var id string
	var err error

	if asset != nil {
		fmt.Printf("🧵 Preparing media for project %q...\n", p.Name)
		
		mediaURL := asset.FilePath
		var cleanup func()

		// If it's a local file, set up transient hosting
		if !strings.HasPrefix(asset.FilePath, "http") {
			fmt.Println("🧵 Starting transient hosting via localhost.run...")
			u, c, err := threads.HostLocalFile(asset.FilePath)
			if err != nil {
				fmt.Printf("Error setting up transient hosting: %v\n", err)
				return
			}
			mediaURL = u
			cleanup = c
			fmt.Printf("🧵 Media temporarily hosted at: %s\n", mediaURL)
		}

		if cleanup != nil {
			defer cleanup()
		}

		ext := strings.ToLower(filepath.Ext(asset.FilePath))
		if ext == ".mp4" || ext == ".mov" {
			// Video publishing logic
			fmt.Println("⚠️  Video publishing requires processing time on Meta servers.")
			// Note: For videos, Step 1 creates a container, but we might need to poll for status.
			// Simple implementation for now.
			containerID, err := client.CreateImageContainer(mediaURL, text) // Reuse container logic for now if possible or update Client
			if err != nil {
				fmt.Printf("Error creating media container: %v\n", err)
				return
			}
			id, err = client.PublishContainer(containerID)
		} else {
			// Image publishing logic
			containerID, err := client.CreateImageContainer(mediaURL, text)
			if err != nil {
				fmt.Printf("Error creating image container: %v\n", err)
				return
			}
			id, err = client.PublishContainer(containerID)
		}
	} else {
		id, err = client.CreateTextPost(text)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Published successfully to project %q! Post ID: %s\n", p.Name, id)
}
