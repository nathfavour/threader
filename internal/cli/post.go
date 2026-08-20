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
	"github.com/nathfavour/threader/internal/orchestrator"
	"github.com/nathfavour/threader/internal/platform"
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
	postCraftCmd.Flags().StringP("target", "t", "", "Target platform(s) comma-separated (e.g. nostr, threads). Defaults to all enabled")

	postPublishCmd.Flags().StringP("project", "p", "", "Project ID or Name (defaults to first project)")
	postPublishCmd.Flags().String("media", "", "Path to media file")
	postPublishCmd.Flags().StringP("target", "t", "", "Target platform(s) comma-separated (e.g. nostr, threads). Defaults to all enabled")
}

var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Manage multi-platform posts (Nostr, Threads, etc.)",
}

var postCraftCmd = &cobra.Command{
	Use:   "craft",
	Short: "Craft a new post using AI or manual input",
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		goal, _ := cmd.Flags().GetString("goal")
		manual, _ := cmd.Flags().GetString("manual")
		mediaPath, _ := cmd.Flags().GetString("media")
		targetFlag, _ := cmd.Flags().GetString("target")

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

		var targets []string
		if targetFlag != "" {
			for _, t := range strings.Split(targetFlag, ",") {
				trimmed := strings.TrimSpace(t)
				if trimmed != "" {
					targets = append(targets, trimmed)
				}
			}
		}

		fmt.Print("Publish now? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		ans, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(ans)) == "y" {
			publishToProject(p, postText, asset, targets)
		}
	},
}

var postPublishCmd = &cobra.Command{
	Use:   "publish [text]",
	Short: "Publish a post to configured targets (Nostr, Threads)",
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		mediaPath, _ := cmd.Flags().GetString("media")
		targetFlag, _ := cmd.Flags().GetString("target")

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

		var targets []string
		if targetFlag != "" {
			for _, t := range strings.Split(targetFlag, ",") {
				trimmed := strings.TrimSpace(t)
				if trimmed != "" {
					targets = append(targets, trimmed)
				}
			}
		}

		publishToProject(p, text, asset, targets)
	},
}

func publishToProject(p *project.Project, text string, asset *media.Asset, targets []string) {
	var mediaURLs []string
	var cleanup func()

	if asset != nil {
		fmt.Printf("🧵 Preparing media for project %q...\n", p.Name)
		mediaURL := asset.FilePath

		// If it's a local file, set up transient hosting
		if !strings.HasPrefix(asset.FilePath, "http") {
			fmt.Println("🧵 Starting transient hosting via localhost.run...")
			u, c, err := threads.HostLocalFile(asset.FilePath)
			if err != nil {
				fmt.Printf("Warning: Failed to set up transient hosting: %v\n", err)
			} else {
				mediaURL = u
				cleanup = c
				fmt.Printf("🧵 Media temporarily hosted at: %s\n", mediaURL)
			}
		}

		if cleanup != nil {
			defer cleanup()
		}

		mediaURLs = append(mediaURLs, mediaURL)
	}

	content := platform.PostContent{
		Text:      text,
		MediaURLs: mediaURLs,
	}

	dispatcher := orchestrator.NewDispatcher()
	results, errors := dispatcher.Dispatch(context.Background(), p, content, targets...)

	if len(results) > 0 {
		for _, res := range results {
			fmt.Printf("✅ Published successfully to [%s]!\n   Post ID: %s\n   URL: %s\n", strings.ToUpper(res.Platform), res.PostID, res.URL)
		}
	}

	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Printf("⚠️  Publish error: %v\n", err)
		}
	}
}
