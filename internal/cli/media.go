package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nathfavour/threader/internal/media"
	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/pkg/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mediaCmd)
	mediaCmd.AddCommand(mediaUploadCmd)
	mediaCmd.AddCommand(mediaListCmd)

	mediaUploadCmd.Flags().StringP("project", "p", "", "Project ID to associate with")
	mediaListCmd.Flags().StringP("project", "p", "", "Project ID to list media from")
}

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Manage project media assets",
}

var mediaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List media assets in a project",
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			// Try to find the first project as default
			reg, _ := project.NewRegistry(config.ProjectsPath())
			projects := reg.List()
			if len(projects) > 0 {
				projectID = projects[0].ID
				fmt.Printf("Using default project: %s (%s)\n", projects[0].Name, projectID)
			} else {
				fmt.Println("Error: No projects found. Create one with 'threader project create'.")
				return
			}
		}

		projectMediaDir := filepath.Join(config.MediaDir(), projectID, "media")
		files, err := os.ReadDir(projectMediaDir)
		if err != nil {
			fmt.Printf("No media found for project %s\n", projectID)
			return
		}

		fmt.Printf("Media Assets for Project %s:\n", projectID)
		for _, f := range files {
			if filepath.Ext(f.Name()) == ".json" {
				data, _ := os.ReadFile(filepath.Join(projectMediaDir, f.Name()))
				var asset media.Asset
				if json.Unmarshal(data, &asset) == nil {
					status := "Unposted"
					if asset.Posted {
						status = fmt.Sprintf("Posted (Thread: %s)", asset.ThreadID)
					}
					fmt.Printf("- [%s] %s\n", asset.ID, filepath.Base(asset.FilePath))
					fmt.Printf("  Status: %s\n", status)
					if asset.OCRText != "" {
						fmt.Printf("  OCR: %s...\n", asset.OCRText[:stringsMin(len(asset.OCRText), 30)])
					}
				}
			}
		}
	},
}

var mediaUploadCmd = &cobra.Command{
	Use:   "upload [file_path]",
	Short: "Upload and index a media file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			fmt.Println("Error: --project flag is required")
			return
		}

		engine := media.NewEngine(config.MediaDir())
		asset, err := engine.IndexMedia(projectID, args[0])
		if err != nil {
			fmt.Printf("Error indexing media: %v\n", err)
			return
		}

		fmt.Printf("Media indexed successfully!\n")
		fmt.Printf("ID: %s\n", asset.ID)
		if asset.OCRText != "" {
			fmt.Printf("OCR Snippet: %s...\n", asset.OCRText[:stringsMin(len(asset.OCRText), 50)])
		}
	},
}

func stringsMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
