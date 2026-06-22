package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/pkg/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectEditCmd)

	projectCreateCmd.Flags().String("desc", "", "Project description")
	projectCreateCmd.Flags().String("voice", "", "Brand voice")
	projectCreateCmd.Flags().String("site", "", "Website URL")
	projectCreateCmd.Flags().String("code", "", "Codebase URL (if open source)")
	projectCreateCmd.Flags().String("manifest", "", "Path to system architecture manifest file")
	projectCreateCmd.Flags().Int("interval", 4, "Post spacing interval in hours")
}

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage marketing projects",
}

var projectEditCmd = &cobra.Command{
	Use:   "edit [projectID]",
	Short: "Interactively edit a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		reg, _ := project.NewRegistry(config.ProjectsPath())
		var p *project.Project
		var ok bool

		if len(args) == 0 {
			projects := reg.List()
			if len(projects) == 0 {
				fmt.Println("No projects found.")
				return
			}
			p = projects[0]
		} else {
			p, ok = reg.Get(args[0])
			if !ok {
				fmt.Printf("Project %q not found.\n", args[0])
				return
			}
		}

		fmt.Printf("Editing Project: %s (%s)\n", p.Name, p.ID)
		reader := bufio.NewReader(os.Stdin)

		fmt.Printf("1) Name [%s]\n", p.Name)
		fmt.Printf("2) Description [%s]\n", p.Description)
		fmt.Printf("3) Brand Voice [%s]\n", p.BrandVoice)
		fmt.Printf("4) Website URL [%s]\n", p.WebsiteURL)
		fmt.Printf("5) Codebase URL [%s]\n", p.CodebaseURL)
		fmt.Printf("6) Manifest Path [%s]\n", p.ManifestPath)
		fmt.Printf("7) Post Interval (Hours) [%d]\n", p.PostIntervalHours)
		fmt.Printf("8) Cancel\n")
		fmt.Print("Select parameter to edit (1-8): ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		var name, desc, voice, site, code, manifestPath string
		var interval int

		switch choice {
		case "1":
			fmt.Print("Enter new name: ")
			name, _ = reader.ReadString('\n')
			name = strings.TrimSpace(name)
		case "2":
			fmt.Print("Enter new description: ")
			desc, _ = reader.ReadString('\n')
			desc = strings.TrimSpace(desc)
		case "3":
			fmt.Print("Enter new brand voice: ")
			voice, _ = reader.ReadString('\n')
			voice = strings.TrimSpace(voice)
		case "4":
			fmt.Print("Enter new website URL: ")
			site, _ = reader.ReadString('\n')
			site = strings.TrimSpace(site)
		case "5":
			fmt.Print("Enter new codebase URL: ")
			code, _ = reader.ReadString('\n')
			code = strings.TrimSpace(code)
		case "6":
			fmt.Print("Enter new manifest path: ")
			manifestPath, _ = reader.ReadString('\n')
			manifestPath = strings.TrimSpace(manifestPath)
		case "7":
			fmt.Print("Enter new post interval (hours): ")
			valStr, _ := reader.ReadString('\n')
			valStr = strings.TrimSpace(valStr)
			if val, err := strconv.Atoi(valStr); err == nil {
				interval = val
			}
		default:
			fmt.Println("Edit cancelled.")
			return
		}

		updated, err := reg.Update(p.ID, name, desc, voice, site, code, "", manifestPath, interval)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("✅ Project %q updated successfully.\n", updated.Name)
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new project namespace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		reg, _ := project.NewRegistry(config.ProjectsPath())
		desc, _ := cmd.Flags().GetString("desc")
		voice, _ := cmd.Flags().GetString("voice")
		site, _ := cmd.Flags().GetString("site")
		code, _ := cmd.Flags().GetString("code")
		manifest, _ := cmd.Flags().GetString("manifest")
		interval, _ := cmd.Flags().GetInt("interval")

		p, err := reg.Register(args[0], desc, voice, site, code)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		if manifest != "" || interval > 0 {
			_, _ = reg.Update(p.ID, "", "", "", "", "", "", manifest, interval)
		}

		fmt.Printf("Created project: %s (ID: %s)\n", p.Name, p.ID)
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Run: func(cmd *cobra.Command, args []string) {
		reg, _ := project.NewRegistry(config.ProjectsPath())
		projects := reg.List()
		if len(projects) == 0 {
			fmt.Println("No projects found.")
			return
		}
		fmt.Println("Projects:")
		for _, p := range projects {
			fmt.Printf("- %s (%s)\n", p.Name, p.ID)
		}
	},
}
