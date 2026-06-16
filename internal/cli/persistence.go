package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func EnsurePersistence() error {
	if runtime.GOOS != "linux" {
		return nil // Currently only supporting systemd on Linux
	}

	home, _ := os.UserHomeDir()
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	unitPath := filepath.Join(unitDir, "threader.service")

	if _, err := os.Stat(unitPath); err == nil {
		return nil // Service already exists
	}

	if err := os.MkdirAll(unitDir, 0755); err != nil {
		return err
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	content := fmt.Sprintf(`[Unit]
Description=Threader Agentic Marketing System
After=network.target

[Service]
ExecStart=%s --daemon
Restart=always

[Install]
WantedBy=default.target
`, execPath)

	if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		return err
	}

	// Enable and start service
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	_ = exec.Command("systemctl", "--user", "enable", "threader.service").Run()

	return nil
}
