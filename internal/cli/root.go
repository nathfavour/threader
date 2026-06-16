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
	cfgFile   string
	isDaemon  bool
	verbose   bool
	kill      bool
	containerName string
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

		// 2. Interactive Setup if no args and no containers
		if len(os.Args) == 1 || (len(os.Args) == 2 && verbose) {
			m := container.NewManager(config.DataDir())
			list, _ := m.List()
			if len(list) == 0 {
				runInitialSetup(m)
			}
		}

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
		startAgent()
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

func runInitialSetup(m *container.Manager) {
	fmt.Println("👋 Welcome to Threader! Let's set up your first personality and project.")
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Persona Name (default: 'default'): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}

	fmt.Print("Enter Persona Description: ")
	desc, _ := reader.ReadString('\n')
	desc = strings.TrimSpace(desc)

	c, err := m.Create(name, desc)
	if err != nil {
		fmt.Printf("Error creating container: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🧵 Personality %q created.\n", c.Name)

	// Create initial project
	fmt.Println("\n--- Initial Project Setup ---")
	fmt.Print("Enter Project Name (e.g. MyProduct): ")
	projName, _ := reader.ReadString('\n')
	projName = strings.TrimSpace(projName)
	if projName == "" {
		projName = name // Fallback to container name
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

	reg, _ := project.NewRegistry(config.ProjectsPath())
	p, err := reg.Register(projName, desc, voice, site, code)
	if err != nil {
		fmt.Printf("Error creating project: %v\n", err)
	} else {
		fmt.Printf("✅ Project %q initialized.\n", p.Name)
	}

	aiClient := ai.NewClient()
	fmt.Print("\nEnter Threads Access Token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)
	if token != "" {
		vaultKey := fmt.Sprintf("THREADS_TOKEN_%s", strings.ToUpper(c.Name))
		_ = aiClient.VaultSet(vaultKey, token)
		fmt.Printf("✅ Token saved to vault as %s\n", vaultKey)
	}
}

func startAgent() {
	if verbose {
		fmt.Println("🧵 Threader is weaving (foreground mode)...")
	}

	_ = os.WriteFile(config.PIDPath(), []byte(strconv.Itoa(os.Getpid())), 0644)
	
	m := container.NewManager(config.DataDir())
	active, err := m.GetDefault()
	if err == nil {
		fmt.Printf("🧵 Active Container: %s\n", active.Name)
	}

	// Initialize Spine
	s := spine.NewSpine(30 * time.Second)
	
	// Attach Cells
	aiClient := ai.NewClient()
	marketingCell := threads.NewMarketingCell(aiClient)
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
