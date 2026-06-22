package threads

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func (c *MarketingCell) getLastPostTime(projectID string) (time.Time, error) {
	db, err := media.OpenDB(config.ProjectDir(projectID))
	if err != nil {
		return time.Time{}, err
	}
	defer db.Close()

	var postedAtStr sql.NullString
	err = db.db.QueryRow(`SELECT posted_at FROM assets WHERE posted = 1 ORDER BY posted_at DESC LIMIT 1`).Scan(&postedAtStr)
	if err != nil || !postedAtStr.Valid || postedAtStr.String == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, postedAtStr.String)
}

func (c *MarketingCell) processProject(ctx context.Context, p *project.Project, cont *container.Container) error {
	// 0. Check Spacing/Scheduling (Distribute activity over the course of a day)
	lastPost, err := c.getLastPostTime(p.ID)
	if err == nil && !lastPost.IsZero() {
		intervalHours := p.PostIntervalHours
		if intervalHours <= 0 {
			intervalHours = 4 // default to 4 hours
		}
		minInterval := time.Duration(intervalHours) * time.Hour
		timeSinceLastPost := time.Since(lastPost)
		if timeSinceLastPost < minInterval {
			// Waiting calmly; not time to post yet
			return nil
		}
	}

	// 1. Run automatic indexing scanner first to discover and pull in any new images dropped by the user
	projectMediaDir := filepath.Join(config.MediaDir(), p.ID, "media")
	projectDir := config.ProjectDir(p.ID)
	_ = media.ScanAndIndex(p.ID, projectMediaDir, projectDir)

	// 2. Open DB and get unposted assets
	db, err := media.OpenDB(projectDir)
	if err != nil {
		return err
	}
	defer db.Close()

	unposted, err := db.GetUnpostedAssets()
	if err != nil || len(unposted) == 0 {
		return nil // No unposted media
	}

	targetAsset := unposted[0]

	// 3. Pre-publication Validation
	if err := c.validateMedia(targetAsset); err != nil {
		fmt.Printf("MarketingCell: Media validation failed for asset %s: %v\n", targetAsset.ID, err)
		// Mark as skipped/failed to avoid infinite loop on bad asset
		_ = db.MarkPosted(targetAsset.ID, "SKIPPED_INVALID_MEDIA")
		return nil
	}

	// 4. Rotate CTA and craft post
	token := p.AccessToken
	if token == "" {
		return fmt.Errorf("token not found for project %s", p.ID)
	}

	reg, _ := project.NewRegistry(config.ProjectsPath())
	cta, err := reg.RotateCTA(p.ID)
	if err != nil {
		return err
	}

	goal := "Create a viral marketing post for this product."
	postText, err := c.Synth.CraftPost(ctx, p, []*media.Asset{targetAsset}, goal, cta)
	if err != nil {
		return err
	}

	// 5. Validate Character Limit (Threads: 500 chars)
	if len(postText) > 500 {
		fmt.Printf("MarketingCell: Post text too long (%d chars). Truncating.\n", len(postText) )
		postText = postText[:497] + "..."
	}

	// 6. Post to Threads
	client := NewClient(token)
	threadID, err := client.CreateTextPost(postText)
	if err != nil {
		return err
	}

	// 7. Record usage and mark as posted
	c.Quota.RecordPublish(cont.Name)
	_ = db.MarkPosted(targetAsset.ID, threadID)

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
