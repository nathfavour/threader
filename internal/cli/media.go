package cli

import (
	"fmt"

	"github.com/nathfavour/threader/internal/media"
	"github.com/nathfavour/threader/pkg/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mediaCmd)
	mediaCmd.AddCommand(mediaUploadCmd)

	mediaUploadCmd.Flags().String("project", "", "Project ID to associate with")
}

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Manage project media assets",
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
