package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nathfavour/threader/pkg/config"
)

type Project struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	BrandVoice        string    `json:"brand_voice"`
	WebsiteURL        string    `json:"website_url,omitempty"`
	CodebaseURL       string    `json:"codebase_url,omitempty"`
	AccessToken       string    `json:"access_token,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	ManifestPath      string    `json:"manifest_path,omitempty"`
	LastCTAIndex      int       `json:"last_cta_index"`
	PostIntervalHours int       `json:"post_interval_hours,omitempty"`
	PostIntervalMins  int       `json:"post_interval_mins,omitempty"`
	GenerationMode    string    `json:"generation_mode,omitempty"` // Mode A: "vibe", Mode B: "completion" (default)
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

	id := uuid.New().String()
	manifestPath := filepath.Join(config.ProjectDir(id), "README.md")

	p := &Project{
		ID:                id,
		Name:              name,
		Description:       desc,
		BrandVoice:        voice,
		WebsiteURL:        site,
		CodebaseURL:       code,
		CreatedAt:         time.Now(),
		ManifestPath:      manifestPath,
		PostIntervalHours: 0,
		PostIntervalMins:  15, // Default to 15 minutes for even, continuous flow
		GenerationMode:    "completion", // Default to Mode B
	}

	// Write boilerplate
	boilerplate := fmt.Sprintf(`# %s Brand Manifest
Place your system architecture description, manifest specifications, and targeted technical vocabulary here.
The automated Threads post-generation engine will read this file to ground its copy guidelines.
`, name)
	_ = os.WriteFile(manifestPath, []byte(boilerplate), 0644)

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

func (r *Registry) Update(id string, name, desc, voice, site, code, token string, manifestPath string, postIntervalHours int, generationMode string) (*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.projects[id]
	if !ok {
		return nil, fmt.Errorf("project %q not found", id)
	}

	if name != "" {
		p.Name = name
	}
	if desc != "" {
		p.Description = desc
	}
	if voice != "" {
		p.BrandVoice = voice
	}
	if site != "" {
		p.WebsiteURL = site
	}
	if code != "" {
		p.CodebaseURL = code
	}
	if token != "" {
		p.AccessToken = token
	}
	if manifestPath != "" {
		p.ManifestPath = manifestPath
	}
	if postIntervalHours != 0 {
		p.PostIntervalHours = postIntervalHours
		p.PostIntervalMins = 0 // Clear minutes when hours is set
	}
	if generationMode != "" {
		p.GenerationMode = generationMode
	}

	return p, r.save()
}

func (r *Registry) save() error {
	data, err := json.MarshalIndent(r.projects, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.dataPath, data, 0644)
}

var CTAMatrix = []string{
	"Link is on the profile.",
	"Check the bio for the link.",
	"The link is on the page.",
	"Head to the profile for access.",
	"The link is pinned to the profile.",
	"Visit the page for the link.",
	"Link is at the top of the page.",
	"Go to the profile for the link.",
	"Grab the link from the profile.",
}

func (r *Registry) RotateCTA(id string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.projects[id]
	if !ok {
		return "", fmt.Errorf("project %q not found", id)
	}

	cta := CTAMatrix[p.LastCTAIndex%len(CTAMatrix)]
	p.LastCTAIndex = (p.LastCTAIndex + 1) % len(CTAMatrix)
	_ = r.save()
	return cta, nil
}
