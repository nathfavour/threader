package config

import (
	"os"
	"path/filepath"
)

func DataDir() string {
	dir, _ := os.UserConfigDir()
	path := filepath.Join(dir, "threader")
	_ = os.MkdirAll(path, 0755)
	return path
}

func PIDPath() string {
	return filepath.Join(DataDir(), "threader.pid")
}

func ProjectsPath() string {
	return filepath.Join(DataDir(), "projects.json")
}

func MediaDir() string {
	path := filepath.Join(DataDir(), "media")
	_ = os.MkdirAll(path, 0755)
	return path
}
