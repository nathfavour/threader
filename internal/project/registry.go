package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	BrandVoice  string    `json:"brand_voice"`
	WebsiteURL  string    `json:"website_url,omitempty"`
	CodebaseURL string    `json:"codebase_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Registry struct {
	mu       sync.RWMutex
	projects map[string]*Project
	dataPath string
}

func NewRegistry(dataPath string) (*Registry, error) {
	r := &Registry{
		projects: make(map[string]*Project),
		dataPath: dataPath,
	}

	if err := os.MkdirAll(filepath.Dir(dataPath), 0755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(dataPath); err == nil {
		data, err := os.ReadFile(dataPath)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &r.projects); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (r *Registry) Register(name, desc, voice, site, code string) (*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p := &Project{
		ID:          uuid.New().String(),
		Name:        name,
		Description: desc,
		BrandVoice:  voice,
		WebsiteURL:  site,
		CodebaseURL: code,
		CreatedAt:   time.Now(),
	}

	r.projects[p.ID] = p
	return p, r.save()
}

func (r *Registry) Get(id string) (*Project, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.projects[id]
	return p, ok
}

func (r *Registry) List() []*Project {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Project, 0, len(r.projects))
	for _, p := range r.projects {
		list = append(list, p)
	}
	return list
}

func (r *Registry) save() error {
	data, err := json.MarshalIndent(r.projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.dataPath, data, 0644)
}
