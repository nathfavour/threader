package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nathfavour/threader/internal/ai"
	"github.com/nathfavour/threader/internal/container"
	"github.com/nathfavour/threader/internal/media"
	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/internal/synthesis"
	"github.com/nathfavour/threader/pkg/biology"
	"github.com/nathfavour/threader/pkg/config"
)

type MarketingCell struct {
	AI              *ai.Client
	Synth           *synthesis.Synthesizer
	Quota           *QuotaManager
	TargetProjectID string
}

func NewMarketingCell(aiClient *ai.Client) *MarketingCell {
	return &MarketingCell{
		AI:    aiClient,
		Synth: synthesis.NewSynthesizer(aiClient),
		Quota: NewQuotaManager(config.DataDir()),
	}
}

func (c *MarketingCell) Name() string {
	return "MarketingCell"
}

func (c *MarketingCell) Pulse(ctx context.Context) error {
	m := container.NewManager(config.DataDir())
	active, err := m.GetDefault()
	if err != nil {
		return err
	}

	if !c.Quota.CanPublish(active.Name) {
		fmt.Printf("MarketingCell: Quota exceeded for %s. Sleeping.\n", active.Name)
		return nil
	}

	// Record activity to keep spine awake if we are doing work
	biology.GetMetabolism().RecordActivity()

	reg, _ := project.NewRegistry(config.ProjectsPath())
	projects := reg.List()

	for _, p := range projects {
		if c.TargetProjectID != "" && p.ID != c.TargetProjectID && p.Name != c.TargetProjectID {
			continue
		}
		if err := c.processProject(ctx, p, active); err != nil {
			fmt.Printf("MarketingCell: Failed to process project %s: %v\n", p.Name, err)
		}
	}

	return nil
}

func (c *MarketingCell) processProject(ctx context.Context, p *project.Project, cont *container.Container) error {
	// 1. Find unposted media
	projectMediaDir := filepath.Join(config.MediaDir(), p.ID, "media")
	
	files, err := os.ReadDir(projectMediaDir)
	if err != nil {
		return nil // No media yet
	}

	var targetAsset *media.Asset
	var targetPath string
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".json" {
			path := filepath.Join(projectMediaDir, f.Name())
			data, _ := os.ReadFile(path)
			var asset media.Asset
			if json.Unmarshal(data, &asset) == nil && !asset.Posted {
				targetAsset = &asset
				targetPath = path
				break
			}
		}
	}

	if targetAsset == nil {
		return nil
	}

	// 2. Pre-publication Validation
	if err := c.validateMedia(targetAsset); err != nil {
		fmt.Printf("MarketingCell: Media validation failed for asset %s: %v\n", targetAsset.ID, err)
		// Mark as skipped/failed to avoid infinite loop on bad asset
		targetAsset.Posted = true
		targetAsset.ThreadID = "SKIPPED_INVALID_MEDIA"
		updatedData, _ := json.MarshalIndent(targetAsset, "", "  ")
		_ = os.WriteFile(targetPath, updatedData, 0644)
		return nil
	}

	// 3. Pick one and craft post
	// Fetch token for this container
	vaultKey := fmt.Sprintf("THREADS_TOKEN_%s", strings.ToUpper(cont.Name))
	token, err := c.AI.VaultGet(vaultKey)
	if err != nil {
		return fmt.Errorf("token not found in vault: %s", vaultKey)
	}

	goal := "Create a viral marketing post for this product."
	postText, err := c.Synth.CraftPost(ctx, p, []*media.Asset{targetAsset}, goal)
	if err != nil {
		return err
	}

	// 4. Validate Character Limit (Threads: 500 chars)
	if len(postText) > 500 {
		fmt.Printf("MarketingCell: Post text too long (%d chars). Truncating.\n", len(postText) )
		postText = postText[:497] + "..."
	}

	// 5. Post to Threads
	client := NewClient(token)
	threadID, err := client.CreateTextPost(postText)
	if err != nil {
		return err
	}

	// 6. Record usage and mark as posted
	c.Quota.RecordPublish(cont.Name)
	targetAsset.Posted = true
	targetAsset.ThreadID = threadID
	
	updatedData, _ := json.MarshalIndent(targetAsset, "", "  ")
	_ = os.WriteFile(targetPath, updatedData, 0644)

	fmt.Printf("MarketingCell: Successfully posted to Threads for project %s (ThreadID: %s)\n", p.Name, threadID)
	return nil
}

func (c *MarketingCell) validateMedia(a *media.Asset) error {
	info, err := os.Stat(a.FilePath)
	if err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(a.FilePath))
	size := info.Size()

	switch ext {
	case ".jpg", ".jpeg", ".png":
		if size > 8*1024*1024 {
			return fmt.Errorf("image size exceeds 8MB limit: %.2fMB", float64(size)/(1024*1024))
		}
	case ".mp4", ".mov":
		if size > 1024*1024*1024 {
			return fmt.Errorf("video size exceeds 1GB limit: %.2fGB", float64(size)/(1024*1024*1024))
		}
	default:
		return fmt.Errorf("unsupported media format: %s", ext)
	}

	return nil
}
