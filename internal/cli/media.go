package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nathfavour/threader/internal/media"
	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/pkg/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mediaCmd)
	mediaCmd.AddCommand(mediaUploadCmd)
	mediaCmd.AddCommand(mediaListCmd)
	mediaCmd.AddCommand(mediaAddCmd)

	mediaUploadCmd.Flags().StringP("project", "p", "", "Project ID to associate with")
	mediaListCmd.Flags().StringP("project", "p", "", "Project ID to list media from")
	mediaAddCmd.Flags().StringP("project", "p", "", "Project ID to add media to")
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
			reg, _ := project.NewRegistry(config.ProjectsPath())
			projects := reg.List()
			if len(projects) > 0 {
				projectID = projects[0].ID
				fmt.Printf("Using default project: %s (%s)\n", projects[0].Name, projectID)
			} else {
				fmt.Println("Error: No projects found.")
				return
			}
		}

		projectMediaDir := filepath.Join(config.MediaDir(), projectID, "media")
		absPath, _ := filepath.Abs(projectMediaDir)
		fmt.Printf("Project Media Directory: %s\n", absPath)
		fmt.Println("--------------------------------------------------")

		files, err := os.ReadDir(projectMediaDir)
		if err != nil {
			fmt.Printf("No media indexed for project %s\n", projectID)
			return
		}

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
				}
			}
		}
	},
}

var mediaAddCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Add media from a file or directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			reg, _ := project.NewRegistry(config.ProjectsPath())
			projects := reg.List()
			if len(projects) > 0 {
				projectID = projects[0].ID
			} else {
				fmt.Println("Error: No projects found.")
				return
			}
		}

		inputPath := args[0]
		info, err := os.Stat(inputPath)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		engine := media.NewEngine(config.MediaDir())
		var filesToProcess []string

		if info.IsDir() {
			entries, _ := os.ReadDir(inputPath)
			for _, entry := range entries {
				if !entry.IsDir() && isSupportedMedia(entry.Name()) {
					filesToProcess = append(filesToProcess, filepath.Join(inputPath, entry.Name()))
				}
			}
		} else {
			if isSupportedMedia(info.Name()) {
				filesToProcess = append(filesToProcess, inputPath)
			} else {
				fmt.Printf("Error: Unsupported media type: %s\n", info.Name())
				return
			}
		}

		if len(filesToProcess) == 0 {
			fmt.Println("No supported media files found to add.")
			return
		}

		fmt.Printf("Adding %d files to project %s...\n", len(filesToProcess), projectID)
		for _, f := range filesToProcess {
			asset, err := engine.IndexMedia(projectID, f)
			if err != nil {
				fmt.Printf("  [FAILED] %s: %v\n", filepath.Base(f), err)
			} else {
				fmt.Printf("  [OK] %s (ID: %s)\n", filepath.Base(f), asset.ID)
			}
		}
	},
}

func isSupportedMedia(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".mp4", ".mov":
		return true
	}
	return false
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
