package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Container struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

type Manager struct {
	mu        sync.RWMutex
	dir       string
	active    string
}

func NewManager(dataDir string) *Manager {
	dir := filepath.Join(dataDir, "containers")
	_ = os.MkdirAll(dir, 0755)
	return &Manager{dir: dir}
}

func (m *Manager) Create(name, desc string) (*Container, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := filepath.Join(m.dir, name+".json")
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("container %q already exists", name)
	}

	c := &Container{
		Name:        name,
		Description: desc,
	}

	// If it's the first container, make it default
	files, _ := os.ReadDir(m.dir)
	if len(files) == 0 {
		c.IsDefault = true
	}

	return c, m.save(c)
}

func (m *Manager) List() ([]*Container, error) {
	files, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, err
	}

	var list []*Container
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(m.dir, f.Name()))
			if err == nil {
				var c Container
				if json.Unmarshal(data, &c) == nil {
					list = append(list, &c)
				}
			}
		}
	}
	return list, nil
}

func (m *Manager) GetDefault() (*Container, error) {
	list, err := m.List()
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		if c.IsDefault {
			return c, nil
		}
	}
	if len(list) > 0 {
		return list[0], nil
	}
	return nil, fmt.Errorf("no containers found")
}

func (m *Manager) SetDefault(name string) error {
	list, err := m.List()
	if err != nil {
		return err
	}

	found := false
	for _, c := range list {
		c.IsDefault = (c.Name == name)
		if c.IsDefault {
			found = true
		}
		_ = m.save(c)
	}

	if !found {
		return fmt.Errorf("container %q not found", name)
	}
	return nil
}

func (m *Manager) save(c *Container) error {
	path := filepath.Join(m.dir, c.Name+".json")
	data, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(path, data, 0644)
}
