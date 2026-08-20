package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/pkg/config"
	"github.com/nathfavour/threader/pkg/nostrutil"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectEditCmd)
	projectCmd.AddCommand(projectTargetCmd)

	projectCreateCmd.Flags().String("desc", "", "Project description")
	projectCreateCmd.Flags().String("voice", "", "Brand voice")
	projectCreateCmd.Flags().String("site", "", "Website URL")
	projectCreateCmd.Flags().String("code", "", "Codebase URL (if open source)")
	projectCreateCmd.Flags().String("manifest", "", "Path to system architecture manifest file")
	projectCreateCmd.Flags().Int("interval", 4, "Post spacing interval in hours")
	projectCreateCmd.Flags().String("nsec", "", "Nostr private key (nsec or hex)")
	projectCreateCmd.Flags().String("nsec-env", "", "Nostr private key environment variable name")
	projectCreateCmd.Flags().String("threads-token", "", "Threads Access Token")
	projectCreateCmd.Flags().String("relays", "", "Custom Nostr relays comma-separated")
}

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage marketing projects and publishing targets",
}

var projectTargetCmd = &cobra.Command{
	Use:   "target [projectID] [enable|disable] [platform]",
	Short: "Quickly enable or disable a publishing target (e.g. nostr, threads)",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		projectID := args[0]
		action := strings.ToLower(args[1])
		platID := strings.ToLower(args[2])

		reg, _ := project.NewRegistry(config.ProjectsPath())
		p, ok := reg.Get(projectID)
		if !ok {
			fmt.Printf("Project %q not found.\n", projectID)
			return
		}

		cfg := p.GetPlatformConfig(platID)
		if cfg == nil {
			cfg = &project.PlatformConfig{}
		}

		if action == "enable" || action == "on" || action == "true" {
			cfg.Enabled = true
			fmt.Printf("✅ Enabled %s target for project %q\n", platID, p.Name)
		} else {
			cfg.Enabled = false
			fmt.Printf("❌ Disabled %s target for project %q\n", platID, p.Name)
		}

		_, err := reg.UpdatePlatform(p.ID, platID, cfg)
		if err != nil {
			fmt.Printf("Error updating target: %v\n", err)
		}
	},
}

var projectEditCmd = &cobra.Command{
	Use:   "edit [projectID]",
	Short: "Interactively edit a project and its platform targets",
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

		p.EnsurePlatforms()

		nostrCfg := p.GetPlatformConfig("nostr")
		threadsCfg := p.GetPlatformConfig("threads")

		nostrStatus := "disabled"
		nostrKeyDisplay := "none"
		if nostrCfg != nil {
			if nostrCfg.Enabled {
				nostrStatus = "ENABLED"
			}
			if nostrCfg.Nsec != "" {
				nostrKeyDisplay = nostrutil.MaskNsec(nostrCfg.Nsec)
			} else if nostrCfg.NsecEnv != "" {
				nostrKeyDisplay = "$" + nostrCfg.NsecEnv
			}
		}

		threadsStatus := "disabled"
		threadsTokenDisplay := "none"
		if threadsCfg != nil {
			if threadsCfg.Enabled {
				threadsStatus = "ENABLED"
			}
			if threadsCfg.AccessToken != "" {
				threadsTokenDisplay = maskToken(threadsCfg.AccessToken)
			} else if threadsCfg.AccessTokenEnv != "" {
				threadsTokenDisplay = "$" + threadsCfg.AccessTokenEnv
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
		fmt.Printf("8) Generation Mode [%s]\n", p.GenerationMode)
		fmt.Printf("9) Nostr Target [%s | Key: %s | Npub: %s]\n", nostrStatus, nostrKeyDisplay, nostrCfg.Npub)
		fmt.Printf("10) Threads Target [%s | Token: %s]\n", threadsStatus, threadsTokenDisplay)
		fmt.Printf("11) Edit README/Manifest File directly\n")
		fmt.Printf("12) Cancel\n")
		fmt.Print("Select parameter to edit (1-12): ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		var name, desc, voice, site, code, manifestPath, generationMode string
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
		case "8":
			fmt.Print("Enter generation mode ('completion' or 'vibe'): ")
			generationMode, _ = reader.ReadString('\n')
			generationMode = strings.TrimSpace(generationMode)
		case "9":
			fmt.Println("\n--- Configure Nostr Target ---")
			fmt.Printf("Currently: %s\n", nostrStatus)
			fmt.Print("Enable Nostr target? (y/n, enter to keep current): ")
			en, _ := reader.ReadString('\n')
			en = strings.TrimSpace(strings.ToLower(en))
			if en == "y" || en == "yes" {
				nostrCfg.Enabled = true
			} else if en == "n" || en == "no" {
				nostrCfg.Enabled = false
			}

			fmt.Print("Enter Nostr nsec private key (nsec1... or hex, leave empty to keep): ")
			nsecInput, _ := reader.ReadString('\n')
			nsecInput = strings.TrimSpace(nsecInput)
			if nsecInput != "" {
				nostrCfg.Nsec = nsecInput
				if pubHex, err := nostrutil.GetPublicKeyHex(nsecInput); err == nil {
					if npub, err := nostrutil.EncodeNpub(pubHex); err == nil {
						nostrCfg.Npub = npub
						fmt.Printf("✅ Derived Npub: %s\n", npub)
					}
				}
			}

			fmt.Print("Or specify env variable name storing nsec (e.g. NOSTR_NSEC, leave empty to keep): ")
			nsecEnv, _ := reader.ReadString('\n')
			nsecEnv = strings.TrimSpace(nsecEnv)
			if nsecEnv != "" {
				nostrCfg.NsecEnv = nsecEnv
			}

			fmt.Print("Enter custom Nostr relays (comma-separated, leave empty to use defaults): ")
			relaysInput, _ := reader.ReadString('\n')
			relaysInput = strings.TrimSpace(relaysInput)
			if relaysInput != "" {
				var relays []string
				for _, r := range strings.Split(relaysInput, ",") {
					tr := strings.TrimSpace(r)
					if tr != "" {
						relays = append(relays, tr)
					}
				}
				nostrCfg.Relays = relays
			}

			_, _ = reg.UpdatePlatform(p.ID, "nostr", nostrCfg)
			fmt.Println("✅ Nostr target updated successfully.")
			return

		case "10":
			fmt.Println("\n--- Configure Threads Target ---")
			fmt.Printf("Currently: %s\n", threadsStatus)
			fmt.Print("Enable Threads target? (y/n, enter to keep current): ")
			en, _ := reader.ReadString('\n')
			en = strings.TrimSpace(strings.ToLower(en))
			if en == "y" || en == "yes" {
				threadsCfg.Enabled = true
			} else if en == "n" || en == "no" {
				threadsCfg.Enabled = false
			}

			fmt.Print("Enter new Threads Access Token (leave empty to keep current): ")
			tokenInput, _ := reader.ReadString('\n')
			tokenInput = strings.TrimSpace(tokenInput)
			if tokenInput != "" {
				threadsCfg.AccessToken = tokenInput
				p.AccessToken = tokenInput
			}

			fmt.Print("Or specify env variable name storing Threads token (e.g. THREADS_TOKEN): ")
			tokenEnv, _ := reader.ReadString('\n')
			tokenEnv = strings.TrimSpace(tokenEnv)
			if tokenEnv != "" {
				threadsCfg.AccessTokenEnv = tokenEnv
			}

			_, _ = reg.UpdatePlatform(p.ID, "threads", threadsCfg)
			fmt.Println("✅ Threads target updated successfully.")
			return

		case "11":
			if p.ManifestPath == "" {
				p.ManifestPath = filepath.Join(config.ProjectDir(p.ID), "README.md")
				_, _ = reg.Update(p.ID, "", "", "", "", "", "", p.ManifestPath, 0, "")
			}
			fmt.Println("\n--- Edit README/Manifest Content ---")
			fmt.Println("1) Edit directly with default terminal editor")
			fmt.Println("2) Import/Copy from an existing file path")
			fmt.Print("Select option (1-2): ")
			opt, _ := reader.ReadString('\n')
			opt = strings.TrimSpace(opt)

			if opt == "1" {
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "nano"
				}
				fmt.Printf("Opening %s with %s...\n", p.ManifestPath, editor)
				cmdExec := exec.Command(editor, p.ManifestPath)
				cmdExec.Stdin = os.Stdin
				cmdExec.Stdout = os.Stdout
				cmdExec.Stderr = os.Stderr
				_ = cmdExec.Run()
				fmt.Println("README/Manifest file updated directly.")
			} else if opt == "2" {
				fmt.Print("Enter source file path: ")
				srcPath, _ := reader.ReadString('\n')
				srcPath = strings.TrimSpace(srcPath)

				absPath, err := filepath.Abs(srcPath)
				if err != nil {
					fmt.Printf("Error resolving path: %v\n", err)
					return
				}
				content, err := os.ReadFile(absPath)
				if err != nil {
					fmt.Printf("Error reading source file: %v\n", err)
					return
				}
				err = os.WriteFile(p.ManifestPath, content, 0644)
				if err != nil {
					fmt.Printf("Error writing to destination: %v\n", err)
					return
				}
				fmt.Printf("Successfully copied contents of %s to project README (%s)\n", absPath, p.ManifestPath)
			} else {
				fmt.Println("Invalid option chosen.")
			}
			return
		default:
			fmt.Println("Edit cancelled.")
			return
		}

		updated, err := reg.Update(p.ID, name, desc, voice, site, code, "", manifestPath, interval, generationMode)
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
		nsec, _ := cmd.Flags().GetString("nsec")
		nsecEnv, _ := cmd.Flags().GetString("nsec-env")
		threadsToken, _ := cmd.Flags().GetString("threads-token")
		relays, _ := cmd.Flags().GetString("relays")

		p, err := reg.Register(args[0], desc, voice, site, code)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		if manifest != "" || interval > 0 {
			_, _ = reg.Update(p.ID, "", "", "", "", "", "", manifest, interval, "")
		}

		// Configure Nostr
		if nsec != "" || nsecEnv != "" || relays != "" {
			nostrCfg := p.GetPlatformConfig("nostr")
			if nostrCfg == nil {
				nostrCfg = &project.PlatformConfig{Enabled: true}
			}
			nostrCfg.Enabled = true
			if nsec != "" {
				nostrCfg.Nsec = nsec
				if pubHex, err := nostrutil.GetPublicKeyHex(nsec); err == nil {
					if npub, err := nostrutil.EncodeNpub(pubHex); err == nil {
						nostrCfg.Npub = npub
					}
				}
			}
			if nsecEnv != "" {
				nostrCfg.NsecEnv = nsecEnv
			}
			if relays != "" {
				var relayList []string
				for _, r := range strings.Split(relays, ",") {
					tr := strings.TrimSpace(r)
					if tr != "" {
						relayList = append(relayList, tr)
					}
				}
				nostrCfg.Relays = relayList
			}
			_, _ = reg.UpdatePlatform(p.ID, "nostr", nostrCfg)
		}

		// Configure Threads
		if threadsToken != "" {
			threadsCfg := &project.PlatformConfig{
				Enabled:     true,
				AccessToken: threadsToken,
			}
			_, _ = reg.UpdatePlatform(p.ID, "threads", threadsCfg)
		}

		fmt.Printf("Created project: %s (ID: %s)\n", p.Name, p.ID)
		fmt.Printf("Targets configured: Nostr [%v], Threads [%v]\n", p.GetPlatformConfig("nostr").Enabled, p.GetPlatformConfig("threads").Enabled)
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects and their active targets",
	Run: func(cmd *cobra.Command, args []string) {
		reg, _ := project.NewRegistry(config.ProjectsPath())
		projects := reg.List()
		if len(projects) == 0 {
			fmt.Println("No projects found.")
			return
		}
		fmt.Println("Projects:")
		for _, p := range projects {
			p.EnsurePlatforms()
			var targets []string
			for k, v := range p.Platforms {
				if v != nil && v.Enabled {
					info := k
					if k == "nostr" && v.Npub != "" {
						info += fmt.Sprintf("(%s)", v.Npub[:10]+"...")
					}
					targets = append(targets, info)
				}
			}
			targetStr := strings.Join(targets, ", ")
			if targetStr == "" {
				targetStr = "none"
			}
			fmt.Printf("- %s (%s) [Active Targets: %s]\n", p.Name, p.ID, targetStr)
		}
	},
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "*****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
