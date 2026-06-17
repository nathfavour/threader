package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nathfavour/threader/internal/ai"
	"github.com/nathfavour/threader/internal/container"
	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/internal/threads"
	"github.com/nathfavour/threader/pkg/config"
	"github.com/nathfavour/threader/pkg/spine"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile       string
	isDaemon      bool
	verbose       bool
	kill          bool
	containerName string
	targetProject string
)

var rootCmd = &cobra.Command{
	Use:   "threader",
	Short: "Threader is an agentic marketing system for Threads",
	Long:  `A specialized agent that handles product marketing on Meta's Threads platform using AI and OCR.`,
	Run: func(cmd *cobra.Command, args []string) {
		pidFile := config.PIDPath()

		// 1. Handle Kill Flag
		if kill {
			handleKill(pidFile)
			return
		}

		// 2. Validate Configs and Setup
		m := container.NewManager(config.DataDir())
		reg, _ := project.NewRegistry(config.ProjectsPath())
		
		selectedProject := validateAndSelectProject(m, reg, targetProject)
		if selectedProject == nil {
			list, _ := m.List()
			projects := reg.List()

			if len(list) == 0 && len(projects) == 0 {
				runInitialSetup(m)
				selectedProject = validateAndSelectProject(m, reg, targetProject)
			} else {
				fmt.Println("⚠️  No project with a valid Threads token found.")
				fmt.Println("--- Project Configuration ---")
				if len(projects) > 0 {
					fmt.Println("Existing projects found:")
					for i, p := range projects {
						fmt.Printf("%d) %s (%s)\n", i+1, p.Name, p.ID)
					}
					fmt.Printf("%d) Create new persona/project\n", len(projects)+1)
					fmt.Print("Select an option (index): ")
					
					reader := bufio.NewReader(os.Stdin)
					choice, _ := reader.ReadString('\n')
					choice = strings.TrimSpace(choice)
					idx, err := strconv.Atoi(choice)
					
					if err == nil && idx >= 1 && idx <= len(projects) {
						configureProject(projects[idx-1])
						selectedProject = projects[idx-1]
					} else if choice == "" || idx == len(projects)+1 {
						runInitialSetup(m)
						selectedProject = validateAndSelectProject(m, reg, targetProject)
					}
				} else {
					runInitialSetup(m)
					selectedProject = validateAndSelectProject(m, reg, targetProject)
				}
			}
			
			if selectedProject == nil {
				fmt.Println("❌ Could not resolve a valid project configuration. Exiting.")
				os.Exit(1)
			}
		}

		fmt.Printf("🧵 Using project: %s\n", selectedProject.Name)

		// 3. Check if already running
		if pidData, err := os.ReadFile(pidFile); err == nil {
			pid, _ := strconv.Atoi(string(pidData))
			if isProcessRunning(pid) {
				fmt.Printf("🧵 Threader is already running (PID: %d)\n", pid)
				return
			}
		}

		// 4. Handle Daemonization
		if !isDaemon && !verbose {
			daemonize()
			return
		}

		// 5. Persistence Setup
		if isDaemon {
			if err := EnsurePersistence(); err != nil {
				fmt.Printf("Warning: Could not configure persistence: %v\n", err)
			}
		}

		// 6. Dependency Check (Tesseract)
		if err := CheckAndInstallDependencies(); err != nil {
			fmt.Printf("Warning: Dependency check failed: %v\n", err)
		}

		// 7. Main process logic
		startAgent(selectedProject)
	},
}

func handleKill(pidFile string) {
	if pidData, err := os.ReadFile(pidFile); err == nil {
		pid, _ := strconv.Atoi(string(pidData))
		if isProcessRunning(pid) {
			process, _ := os.FindProcess(pid)
			if err := process.Signal(syscall.SIGTERM); err != nil {
				fmt.Printf("Error killing process %d: %v\n", pid, err)
			} else {
				fmt.Printf("🧵 Threader (PID: %d) terminated.\n", pid)
			}
		} else {
			fmt.Println("🧵 Threader is not running.")
		}
		_ = os.Remove(pidFile)
	} else {
		fmt.Println("🧵 No PID file found.")
	}
}

func daemonize() {
	daemonCmd := exec.Command(os.Args[0], "--daemon")
	logFile, _ := os.OpenFile(filepath.Join(config.DataDir(), "threader.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	daemonCmd.Stdout = logFile
	daemonCmd.Stderr = logFile
	
	if err := daemonCmd.Start(); err != nil {
		fmt.Printf("Error starting daemon: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("🧵 Threader started in background (PID: %d)\n", daemonCmd.Process.Pid)
	os.Exit(0)
}

func configureProject(p *project.Project) {
	fmt.Printf("\n--- Configuring Project: %s ---\n", p.Name)
	reader := bufio.NewReader(os.Stdin)
	aiClient := ai.NewClient()

	fmt.Print("Enter Threads Access Token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if token != "" {
		_ = aiClient.VaultSet(fmt.Sprintf("THREADS_TOKEN_%s", p.ID), token)
		fmt.Printf("✅ Token saved to vault for project %s\n", p.Name)
	}
}

func runInitialSetup(m *container.Manager) {
	fmt.Println("👋 Welcome to Threader! Let's set up your personality and project.")
	reader := bufio.NewReader(os.Stdin)

	list, _ := m.List()
	var c *container.Container

	if len(list) > 0 {
		fmt.Println("\n--- Existing Personas ---")
		for i, persona := range list {
			fmt.Printf("%d) %s\n", i+1, persona.Name)
		}
		fmt.Printf("%d) Create new persona\n", len(list)+1)
		fmt.Print("Select a persona (default: 1): ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		idx, err := strconv.Atoi(choice)
		if choice == "" {
			c = list[0]
		} else if err == nil && idx >= 1 && idx <= len(list) {
			c = list[idx-1]
		}
	}

	if c == nil {
		fmt.Print("Enter Persona Name (default: 'default'): ")
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)
		if name == "" {
			name = "default"
		}

		fmt.Print("Enter Persona Description: ")
		desc, _ := reader.ReadString('\n')
		desc = strings.TrimSpace(desc)

		var err error
		c, err = m.Create(name, desc)
		if err != nil {
			fmt.Printf("Error creating container: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("🧵 Personality %q created.\n", c.Name)
	} else {
		fmt.Printf("🧵 Using personality %q.\n", c.Name)
	}

	reg, _ := project.NewRegistry(config.ProjectsPath())
	projects := reg.List()

	var p *project.Project
	useExisting := false
	if len(projects) > 0 {
		fmt.Println("\n--- Project Selection ---")
		fmt.Printf("Found %d existing project(s):\n", len(projects))
		for i, existingProj := range projects {
			fmt.Printf("%d) %s (%s)\n", i+1, existingProj.Name, existingProj.ID)
		}
		fmt.Print("Select a project index to link, or press Enter to create a new one: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		if choice != "" {
			idx, err := strconv.Atoi(choice)
			if err == nil && idx >= 1 && idx <= len(projects) {
				p = projects[idx-1]
				fmt.Printf("✅ Linked to existing project: %s\n", p.Name)
				useExisting = true
			}
		}
	}

	if !useExisting {
		// Create initial project
		fmt.Println("\n--- Initial Project Setup ---")
		
		defaultProjName := c.Name
		fmt.Printf("Enter Project Name (default: %q): ", defaultProjName)
		projName, _ := reader.ReadString('\n')
		projName = strings.TrimSpace(projName)
		if projName == "" {
			projName = defaultProjName
		}

		fmt.Print("Enter Brand Voice (e.g. casual, professional): ")
		voice, _ := reader.ReadString('\n')
		voice = strings.TrimSpace(voice)

		fmt.Print("Enter Website URL: ")
		site, _ := reader.ReadString('\n')
		site = strings.TrimSpace(site)

		fmt.Print("Enter Codebase URL (optional, for Open Source): ")
		code, _ := reader.ReadString('\n')
		code = strings.TrimSpace(code)

		var err error
		p, err = reg.Register(projName, c.Description, voice, site, code)
		if err != nil {
			fmt.Printf("Error creating project: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Project %q initialized.\n", p.Name)
	}

	configureProject(p)
}

func validateAndSelectProject(m *container.Manager, reg *project.Registry, target string) *project.Project {
	projects := reg.List()
	if len(projects) == 0 {
		return nil
	}

	aiClient := ai.NewClient()
	
	// Helper to check if a project is "valid"
	isValid := func(p *project.Project) bool {
		// Check for project-specific token
		token, err := aiClient.VaultGet(fmt.Sprintf("THREADS_TOKEN_%s", p.ID))
		return err == nil && token != ""
	}

	if target != "" {
		for _, p := range projects {
			if p.Name == target || p.ID == target {
				if isValid(p) {
					return p
				}
				fmt.Printf("⚠️  Project %q found but is not fully configured (missing token).\n", p.Name)
				return nil
			}
		}
		fmt.Printf("⚠️  Project %q not found.\n", target)
		return nil
	}

	// Search for valid projects
	var validProjects []*project.Project
	for _, p := range projects {
		if isValid(p) {
			validProjects = append(validProjects, p)
		}
	}

	if len(validProjects) == 0 {
		return nil
	}

	if len(validProjects) == 1 {
		return validProjects[0]
	}

	fmt.Println("\n--- Multiple Configured Projects Found ---")
	for i, p := range validProjects {
		fmt.Printf("%d) %s (%s)\n", i+1, p.Name, p.ID)
	}
	fmt.Print("Select project index to use: ")
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	idx, err := strconv.Atoi(choice)
	if err == nil && idx >= 1 && idx <= len(validProjects) {
		return validProjects[idx-1]
	}

	return nil
}

func startAgent(p *project.Project) {
	if verbose {
		fmt.Printf("🧵 Threader is weaving for project %q (foreground mode)...\n", p.Name)
	}

	_ = os.WriteFile(config.PIDPath(), []byte(strconv.Itoa(os.Getpid())), 0644)
	
	m := container.NewManager(config.DataDir())
	active, err := m.GetDefault()
	if err == nil {
		fmt.Printf("🧵 Active Persona: %s\n", active.Name)
	}

	// Initialize Spine
	s := spine.NewSpine(30 * time.Second)
	
	// Attach Cells
	aiClient := ai.NewClient()
	marketingCell := threads.NewMarketingCell(aiClient)
	marketingCell.TargetProjectID = p.ID
	
	s.Attach(marketingCell)

	fmt.Println("🧵 Threader daemon is active.")
	
	ctx := context.Background()
	go s.Breathes(ctx)

	if isDaemon {
		select {}
	}
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. Need to send signal 0 to check existence.
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false
	}

	// Linux-specific: check the process name to ensure it's actually 'threader'
	// and not another process that reused the PID.
	commPath := fmt.Sprintf("/proc/%d/comm", pid)
	if data, err := os.ReadFile(commPath); err == nil {
		comm := string(data)
		// Check if it contains 'threader'
		return os.Args[0] == "threader" || filepath.Base(os.Args[0]) == "threader" || 
			   filepath.Base(comm) == "threader\n" || filepath.Base(comm) == "threader"
	}

	return true
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.threader.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output (run in foreground)")
	rootCmd.PersistentFlags().BoolVarP(&kill, "kill", "k", false, "kill the running threader process")
	rootCmd.PersistentFlags().StringVarP(&targetProject, "project", "p", "", "target project to run")
	rootCmd.PersistentFlags().BoolVar(&isDaemon, "daemon", false, "internal daemon flag")
	_ = rootCmd.PersistentFlags().MarkHidden("daemon")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".threader")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}
}
