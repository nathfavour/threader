package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nathfavour/threader/pkg/config"
	"github.com/nathfavour/threader/pkg/nostrutil"
)

// PlatformConfig holds connection credentials and target options for a single social platform.
type PlatformConfig struct {
	Enabled        bool              `json:"enabled"`
	AccessToken    string            `json:"access_token,omitempty"`
	AccessTokenEnv string            `json:"access_token_env,omitempty"`
	Nsec           string            `json:"nsec,omitempty"`
	NsecEnv        string            `json:"nsec_env,omitempty"`
	Npub           string            `json:"npub,omitempty"`
	Relays         []string          `json:"relays,omitempty"`
	PublishKinds   []int             `json:"publish_kinds,omitempty"`
	NIP05          string            `json:"nip05,omitempty"`
	MediaHost      string            `json:"media_host,omitempty"`
	Custom         map[string]string `json:"custom,omitempty"`
}

type Project struct {
	ID                string                     `json:"id"`
	Name              string                     `json:"name"`
	Description       string                     `json:"description"`
	BrandVoice        string                     `json:"brand_voice"`
	WebsiteURL        string                     `json:"website_url,omitempty"`
	CodebaseURL       string                     `json:"codebase_url,omitempty"`
	AccessToken       string                     `json:"access_token,omitempty"` // Legacy field, synced to Platforms["threads"]
	Platforms         map[string]*PlatformConfig `json:"platforms,omitempty"`
	CreatedAt         time.Time                  `json:"created_at"`
	ManifestPath      string                     `json:"manifest_path,omitempty"`
	LastCTAIndex      int                        `json:"last_cta_index"`
	PostIntervalHours int                        `json:"post_interval_hours,omitempty"`
	PostIntervalMins  int                        `json:"post_interval_mins,omitempty"`
	GenerationMode    string                     `json:"generation_mode,omitempty"` // Mode A: "vibe", Mode B: "completion" (default)
}

// EnsurePlatforms initializes and syncs platform maps and legacy fields.
func (p *Project) EnsurePlatforms() {
	if p.Platforms == nil {
		p.Platforms = make(map[string]*PlatformConfig)
	}

	// Legacy migration: If top-level AccessToken is populated, ensure Threads target is present
	if p.AccessToken != "" {
		if _, ok := p.Platforms["threads"]; !ok {
			p.Platforms["threads"] = &PlatformConfig{
				Enabled:     true,
				AccessToken: p.AccessToken,
			}
		} else if p.Platforms["threads"].AccessToken == "" {
			p.Platforms["threads"].AccessToken = p.AccessToken
			p.Platforms["threads"].Enabled = true
		}
	}

	// Ensure Nostr target exists with defaults if not present
	if _, ok := p.Platforms["nostr"]; !ok {
		p.Platforms["nostr"] = &PlatformConfig{
			Enabled:      false,
			PublishKinds: []int{1},
		}
	}

	// Calculate npub for nostr if nsec is present
	if nostrCfg, ok := p.Platforms["nostr"]; ok && nostrCfg != nil {
		if nostrCfg.Npub == "" && (nostrCfg.Nsec != "" || nostrCfg.NsecEnv != "") {
			nsec := nostrCfg.Nsec
			if nsec == "" && nostrCfg.NsecEnv != "" {
				nsec = os.Getenv(nostrCfg.NsecEnv)
			}
			if nsec != "" {
				if pubHex, err := nostrutil.GetPublicKeyHex(nsec); err == nil {
					if npub, err := nostrutil.EncodeNpub(pubHex); err == nil {
						nostrCfg.Npub = npub
					}
				}
			}
		}
	}
}

// GetPlatformConfig returns the platform configuration for a given platform ID.
func (p *Project) GetPlatformConfig(platformID string) *PlatformConfig {
	p.EnsurePlatforms()
	return p.Platforms[platformID]
}

// SetPlatformConfig updates or sets the platform configuration for a given platform.
func (p *Project) SetPlatformConfig(platformID string, cfg *PlatformConfig) {
	p.EnsurePlatforms()
	p.Platforms[platformID] = cfg
	if platformID == "threads" && cfg != nil && cfg.AccessToken != "" {
		p.AccessToken = cfg.AccessToken
	}
	p.EnsurePlatforms()
}

// GetEnabledPlatforms returns a slice of platform IDs that are explicitly enabled.
func (p *Project) GetEnabledPlatforms() []string {
	p.EnsurePlatforms()
	var enabled []string
	for k, v := range p.Platforms {
		if v != nil && v.Enabled {
			enabled = append(enabled, k)
		}
	}
	return enabled
}

// HasValidTarget checks if at least one platform target has valid credentials and is enabled.
func (p *Project) HasValidTarget() bool {
	p.EnsurePlatforms()
	for platID, cfg := range p.Platforms {
		if cfg != nil && cfg.Enabled {
			driverCfg := p.GetDriverConfig(platID)
			switch platID {
			case "threads":
				if driverCfg["access_token"] != "" || os.Getenv("THREADS_ACCESS_TOKEN") != "" || os.Getenv("THREADS_TOKEN") != "" {
					return true
				}
			case "nostr":
				if driverCfg["nsec"] != "" || driverCfg["nsec_env"] != "" || os.Getenv("NOSTR_NSEC") != "" || os.Getenv("NOSTR_PRIVATE_KEY") != "" {
					return true
				}
			default:
				if len(driverCfg) > 0 {
					return true
				}
			}
		}
	}
	return false
}

// GetDriverConfig extracts a flat map[string]string suitable for PlatformDriver.ValidateConfig & Publish.
func (p *Project) GetDriverConfig(platformID string) map[string]string {
	p.EnsurePlatforms()
	cfg := p.Platforms[platformID]
	res := make(map[string]string)
	if cfg == nil {
		return res
	}

	if cfg.AccessToken != "" {
		res["access_token"] = cfg.AccessToken
	}
	if cfg.AccessTokenEnv != "" {
		res["access_token_env"] = cfg.AccessTokenEnv
	}
	if cfg.Nsec != "" {
		res["nsec"] = cfg.Nsec
	}
	if cfg.NsecEnv != "" {
		res["nsec_env"] = cfg.NsecEnv
	}
	if cfg.Npub != "" {
		res["npub"] = cfg.Npub
	}
	if len(cfg.Relays) > 0 {
		res["relays"] = strings.Join(cfg.Relays, ",")
	}
	if cfg.NIP05 != "" {
		res["nip05"] = cfg.NIP05
	}
	if cfg.MediaHost != "" {
		res["media_host"] = cfg.MediaHost
	}
	for k, v := range cfg.Custom {
		res[k] = v
	}

	return res
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
		for _, p := range r.projects {
			p.EnsurePlatforms()
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
		PostIntervalMins:  15,           // Default to 15 minutes for even, continuous flow
		GenerationMode:    "completion", // Default to Mode B
		Platforms: map[string]*PlatformConfig{
			"threads": {
				Enabled: false,
			},
			"nostr": {
				Enabled:      true, // First-class default enabled
				PublishKinds: []int{1},
			},
		},
	}

	// Write boilerplate
	boilerplate := fmt.Sprintf(`# %s Brand Manifest
Place your system architecture description, manifest specifications, and targeted technical vocabulary here.
The automated marketing and post-generation engine will read this file to ground its copy guidelines.
`, name)
	_ = os.WriteFile(manifestPath, []byte(boilerplate), 0644)

	r.projects[p.ID] = p
	return p, r.save()
}

func (r *Registry) Get(id string) (*Project, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.projects[id]
	if ok {
		p.EnsurePlatforms()
	}
	return p, ok
}

func (r *Registry) List() []*Project {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Project, 0, len(r.projects))
	for _, p := range r.projects {
		p.EnsurePlatforms()
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

	p.EnsurePlatforms()

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
		if p.Platforms["threads"] == nil {
			p.Platforms["threads"] = &PlatformConfig{}
		}
		p.Platforms["threads"].AccessToken = token
		p.Platforms["threads"].Enabled = true
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

func (r *Registry) UpdatePlatform(id string, platformID string, cfg *PlatformConfig) (*Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.projects[id]
	if !ok {
		return nil, fmt.Errorf("project %q not found", id)
	}

	p.SetPlatformConfig(platformID, cfg)
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
