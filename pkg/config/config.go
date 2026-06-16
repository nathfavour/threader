package config

import (
	"os"
	"path/filepath"
)

func DataDir() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".threader")
	_ = os.MkdirAll(path, 0755)
	return path
}

func ProjectsPath() string {
	return filepath.Join(DataDir(), "projects.json")
}

func MediaDir() string {
	path := filepath.Join(DataDir(), "media")
	_ = os.MkdirAll(path, 0755)
	return path
}
