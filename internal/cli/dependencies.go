package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func CheckAndInstallDependencies() error {
	if isCommandAvailable("tesseract") {
		return nil
	}

	fmt.Println("🧵 Tesseract OCR not found. Attempting autonomous installation...")

	switch runtime.GOOS {
	case "linux":
		return installLinuxTesseract()
	case "darwin":
		return installMacTesseract()
	case "windows":
		return installWindowsTesseract()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func installLinuxTesseract() error {
	// Detect package manager
	if isCommandAvailable("apt-get") {
		fmt.Println("🧵 Detected Debian/Ubuntu. Installing libtesseract-dev libleptonica-dev tesseract-ocr...")
		cmd := exec.Command("sudo", "apt-get", "update")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()

		cmd = exec.Command("sudo", "apt-get", "install", "-y", "libtesseract-dev", "libleptonica-dev", "tesseract-ocr")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	} else if isCommandAvailable("pacman") {
		fmt.Println("🧵 Detected Arch Linux. Installing tesseract tesseract-data-eng leptonica...")
		cmd := exec.Command("sudo", "pacman", "-Sy", "--needed", "--noconfirm", "tesseract", "tesseract-data-eng", "leptonica")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}

	return fmt.Errorf("no supported package manager found (apt or pacman)")
}

func installMacTesseract() error {
	if !isCommandAvailable("brew") {
		return fmt.Errorf("homebrew not found. Please install Homebrew first: https://brew.sh/")
	}

	fmt.Println("🧵 Detected macOS. Installing tesseract and leptonica via Homebrew...")
	cmd := exec.Command("brew", "install", "tesseract", "leptonica")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func installWindowsTesseract() error {
	if isCommandAvailable("winget") {
		fmt.Println("🧵 Detected Windows. Attempting installation via winget...")
		cmd := exec.Command("winget", "install", "UB-Mannheim.TesseractOCR")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	} else if isCommandAvailable("choco") {
		fmt.Println("🧵 Detected Windows. Attempting installation via Chocolatey...")
		cmd := exec.Command("choco", "install", "tesseract")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return fmt.Errorf("no supported package manager found (winget or choco). Please download Tesseract manually from: https://github.com/UB-Mannheim/tesseract/wiki")
}
