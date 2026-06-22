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
	_ = db.SQL.QueryRow(`SELECT posted_at FROM assets WHERE posted = 1 ORDER BY posted_at DESC LIMIT 1`).Scan(&postedAtStr)

	var repliedAtStr sql.NullString
	_ = db.SQL.QueryRow(`SELECT replied_at FROM replied_threads ORDER BY replied_at DESC LIMIT 1`).Scan(&repliedAtStr)

	var lastTime time.Time
	if postedAtStr.Valid && postedAtStr.String != "" {
		if t, err := time.Parse(time.RFC3339, postedAtStr.String); err == nil {
			lastTime = t
		}
	}

	if repliedAtStr.Valid && repliedAtStr.String != "" {
		if t, err := time.Parse(time.RFC3339, repliedAtStr.String); err == nil {
			if t.After(lastTime) {
				lastTime = t
			}
		}
	}

	return lastTime, nil
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
			// Waiting calmly; not time to post/reply yet
			return nil
		}
	}

	// 1. Run automatic indexing scanner first to discover and pull in any new images dropped by the user
	projectMediaDir := filepath.Join(config.MediaDir(), p.ID, "media")
	projectDir := config.ProjectDir(p.ID)
	_ = media.ScanAndIndex(p.ID, projectMediaDir, projectDir)

	token := p.AccessToken
	if token == "" {
		return fmt.Errorf("token not found for project %s", p.ID)
	}

	db, err := media.OpenDB(projectDir)
	if err != nil {
		return err
	}
	defer db.Close()

	manifest := synthesis.GetProjectManifest(p)
	client := NewClient(token)

	// 2. Generate Search Keywords/Queries
	queryPrompt := fmt.Sprintf(`Given the product manifest:
%s

Generate 3 search keywords or short topics (maximum 3 words each) that target users would post when experiencing pain points that this product solves.
Output ONLY a JSON array of strings. Do not include markdown formatting or tags. Example: ["notion slow", "markdown editor offline", "password manager alternative"]`, manifest)

	intent := p.GenerationMode
	if intent == "" {
		intent = "completion"
	}

	queryResp, err := c.AI.Query(queryPrompt, intent, "github-models", "")
	var keywords []string
	if err == nil {
		cleaned := strings.Trim(strings.TrimSpace(queryResp), " \n\r`\t")
		if strings.HasPrefix(cleaned, "```json") {
			cleaned = strings.TrimPrefix(cleaned, "```json")
			cleaned = strings.TrimSuffix(cleaned, "```")
		} else if strings.HasPrefix(cleaned, "```") {
			cleaned = strings.TrimPrefix(cleaned, "```")
			cleaned = strings.TrimSuffix(cleaned, "```")
		}
		cleaned = strings.TrimSpace(cleaned)
		_ = json.Unmarshal([]byte(cleaned), &keywords)
	}

	if len(keywords) == 0 {
		// General defaults based on project
		keywords = []string{p.Name + " alternative", "frustrated with notion", "markdown notes app"}
	}

	fmt.Printf("MarketingCell: Generated search keywords: %v\n", keywords)

	// 3. Search and Find Target Pain Point Posts
	var targetPost *Post
	for _, kw := range keywords {
		posts, err := client.SearchPosts(kw)
		if err != nil {
			fmt.Printf("MarketingCell: Search failed for keyword %q: %v. Continuing.\n", kw, err)
			continue
		}

		for _, post := range posts {
			alreadyReplied, err := db.HasReplied(post.ID)
			if err != nil || alreadyReplied {
				continue
			}

			// Evaluate if the post is a relevant pain point
			evalPrompt := fmt.Sprintf(`Product Manifest:
%s

User Post:
"%s"

Evaluate if the user post expresses a genuine problem, frustration, or need that our product directly addresses.
Output ONLY 'YES' or 'NO'.`, manifest, post.Text)

			evalResp, err := c.AI.Query(evalPrompt, intent, "github-models", "")
			if err == nil && strings.Contains(strings.ToUpper(evalResp), "YES") {
				targetPost = &post
				break
			}
		}
		if targetPost != nil {
			break
		}
	}

	// 4. Executing Reply Pitch
	if targetPost != nil {
		fmt.Printf("MarketingCell: Found pain point post to reply to: %s by %s: %q\n", targetPost.ID, targetPost.Username, targetPost.Text)

		// Get all assets to see if we can attach visual context
		var allAssets []*media.Asset
		unposted, err := db.GetUnpostedAssets()
		if err == nil {
			allAssets = append(allAssets, unposted...)
		}

		// Also get posted assets as fallback/library
		rows, err := db.SQL.Query(`SELECT id, file_path, ocr_text, ai_summary FROM assets WHERE posted = 1`)
		if err == nil {
			for rows.Next() {
				var a media.Asset
				if rows.Scan(&a.ID, &a.FilePath, &a.OCRText, &a.AISummary) == nil {
					allAssets = append(allAssets, &a)
				}
			}
			rows.Close()
		}

		var selectedAsset *media.Asset
		if len(allAssets) > 0 {
			var assetContext strings.Builder
			for _, a := range allAssets {
				assetContext.WriteString(fmt.Sprintf("ID: %s | OCR: %s | Summary: %s\n", a.ID, a.OCRText, a.AISummary))
			}

			matchPrompt := fmt.Sprintf(`Target User Post:
"%s"

Product Assets Available:
%s

Analyze the target post and select the single most relevant asset ID to attach to our reply.
If none of the assets are relevant or helpful context, output 'NONE'.
Output ONLY the selected asset ID or 'NONE'. Do not add any text.`, targetPost.Text, assetContext.String())

			matchResp, err := c.AI.Query(matchPrompt, intent, "github-models", "")
			if err == nil {
				matchResp = strings.TrimSpace(matchResp)
				for _, a := range allAssets {
					if a.ID == matchResp || strings.Contains(matchResp, a.ID) {
						selectedAsset = a
						break
					}
				}
			}
		}

		// Rotate CTA
		reg, _ := project.NewRegistry(config.ProjectsPath())
		cta, _ := reg.RotateCTA(p.ID)

		var assetsForReply []*media.Asset
		if selectedAsset != nil {
			assetsForReply = append(assetsForReply, selectedAsset)
			fmt.Printf("MarketingCell: Selected visual asset %s (%s) for reply\n", selectedAsset.ID, filepath.Base(selectedAsset.FilePath))
		}

		replyText, err := c.Synth.CraftReply(ctx, p, targetPost.Text, assetsForReply)
		if err != nil {
			return err
		}

		if cta != "" {
			replyText = replyText + " " + cta
		}

		if len(replyText) > 500 {
			replyText = replyText[:497] + "..."
		}

		var threadID string
		if selectedAsset != nil {
			mediaURL := selectedAsset.FilePath
			var cleanup func()
			if !strings.HasPrefix(selectedAsset.FilePath, "http") {
				u, c, err := HostLocalFile(selectedAsset.FilePath)
				if err != nil {
					return err
				}
				mediaURL = u
				cleanup = c
			}
			if cleanup != nil {
				defer cleanup()
			}
			threadID, err = client.CreateImageReply(mediaURL, replyText, targetPost.ID)
		} else {
			threadID, err = client.CreateReply(replyText, targetPost.ID)
		}

		if err != nil {
			return err
		}

		_ = db.MarkReplied(targetPost.ID)
		if selectedAsset != nil {
			_ = db.MarkPosted(selectedAsset.ID, threadID)
		}
		c.Quota.RecordPublish(cont.Name)
		fmt.Printf("MarketingCell: Successfully replied to post %s (Reply ThreadID: %s)\n", targetPost.ID, threadID)
		return nil
	}

	// 5. Fallback raw posting (only if no relevant reply post was found)
	fmt.Println("MarketingCell: No relevant pain point posts found. Fallback to raw post.")
	unposted, err := db.GetUnpostedAssets()
	if err != nil || len(unposted) == 0 {
		return nil // No media to post
	}

	targetAsset := unposted[0]
	if err := c.validateMedia(targetAsset); err != nil {
		fmt.Printf("MarketingCell: Media validation failed for asset %s: %v\n", targetAsset.ID, err)
		_ = db.MarkPosted(targetAsset.ID, "SKIPPED_INVALID_MEDIA")
		return nil
	}

	reg, _ := project.NewRegistry(config.ProjectsPath())
	cta, _ := reg.RotateCTA(p.ID)

	goal := "Create a viral marketing post for this product."
	postText, err := c.Synth.CraftPost(ctx, p, []*media.Asset{targetAsset}, goal, cta)
	if err != nil {
		return err
	}

	if len(postText) > 500 {
		postText = postText[:497] + "..."
	}

	threadID, err := client.CreateTextPost(postText)
	if err != nil {
		return err
	}

	c.Quota.RecordPublish(cont.Name)
	_ = db.MarkPosted(targetAsset.ID, threadID)
	fmt.Printf("MarketingCell: Successfully fallback-posted to Threads for project %s (ThreadID: %s)\n", p.Name, threadID)
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
