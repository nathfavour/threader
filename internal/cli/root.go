package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/nathfavour/threader/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	isDaemon bool
	verbose  bool
	kill     bool
)

var rootCmd = &cobra.Command{
	Use:   "threader",
	Short: "Threader is an agentic marketing system for Threads",
	Long:  `A specialized agent that handles product marketing on Meta's Threads platform using AI and OCR.`,
	Run: func(cmd *cobra.Command, args []string) {
		pidFile := config.PIDPath()

		// 1. Handle Kill Flag
		if kill {
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
				fmt.Println("🧵 No PID file found. Threader is likely not running.")
			}
			return
		}

		// 2. Check if already running
		if pidData, err := os.ReadFile(pidFile); err == nil {
			pid, _ := strconv.Atoi(string(pidData))
			if isProcessRunning(pid) {
				fmt.Printf("🧵 Threader is already running (PID: %d)\n", pid)
				return
			}
		}

		// 3. Handle Daemonization
		if !isDaemon && !verbose {
			// Re-exec as daemon
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

		// 4. Persistence Setup (Ensure systemd service exists)
		if isDaemon {
			if err := EnsurePersistence(); err != nil {
				fmt.Printf("Warning: Could not configure persistence: %v\n", err)
			}
		}

		// 5. Dependency Check (Tesseract)
		if err := CheckAndInstallDependencies(); err != nil {
			fmt.Printf("Warning: Dependency check failed: %v\n", err)
			fmt.Println("OCR features may not work correctly until Tesseract is installed.")
		}

		// 6. Main process logic
		if verbose {
			fmt.Println("🧵 Threader is weaving (foreground mode)...")
		}

		// Save PID
		_ = os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
		defer os.Remove(pidFile)

		// TODO: Implement the persistent agent loop here
		fmt.Println("🧵 Threader daemon is active.")
		
		// For now, just keep it alive if it's the daemon
		if isDaemon {
			select {} // Hang forever
		}
	},
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
