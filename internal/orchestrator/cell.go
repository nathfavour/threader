package orchestrator

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nathfavour/threader/internal/ai"
	"github.com/nathfavour/threader/internal/container"
	"github.com/nathfavour/threader/internal/media"
	"github.com/nathfavour/threader/internal/platform"
	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/internal/synthesis"
	"github.com/nathfavour/threader/internal/threads"
	"github.com/nathfavour/threader/pkg/biology"
	"github.com/nathfavour/threader/pkg/config"
)

type MarketingCell struct {
	AI              *ai.Client
	Synth           *synthesis.Synthesizer
	Quota           *threads.QuotaManager
	Dispatcher      *Dispatcher
	TargetProjectID string
}

func NewMarketingCell(aiClient *ai.Client) *MarketingCell {
	return &MarketingCell{
		AI:         aiClient,
		Synth:      synthesis.NewSynthesizer(aiClient),
		Quota:      threads.NewQuotaManager(config.DataDir()),
		Dispatcher: NewDispatcher(),
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
		if !p.HasValidTarget() {
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
	p.EnsurePlatforms()

	// 0. Check Spacing/Scheduling (Distribute activity over the course of a day)
	lastPost, err := c.getLastPostTime(p.ID)
	if err == nil && !lastPost.IsZero() {
		var minInterval time.Duration
		if p.PostIntervalMins > 0 {
			minInterval = time.Duration(p.PostIntervalMins) * time.Minute
		} else if p.PostIntervalHours < 0 {
			minInterval = 0
		} else {
			intervalHours := p.PostIntervalHours
			if intervalHours == 0 && p.PostIntervalMins == 0 {
				intervalHours = 4 // Fallback if both are empty
			}
			minInterval = time.Duration(intervalHours) * time.Hour
		}
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

	db, err := media.OpenDB(projectDir)
	if err != nil {
		return err
	}
	defer db.Close()

	var recentCopies []string
	rows, err := db.SQL.Query(`SELECT post_text FROM assets WHERE posted = 1 AND post_text IS NOT NULL AND post_text != "" ORDER BY posted_at DESC LIMIT 5`)
	if err == nil {
		for rows.Next() {
			var text string
			if rows.Scan(&text) == nil {
				recentCopies = append(recentCopies, text)
			}
		}
		rows.Close()
	}

	manifest := synthesis.GetProjectManifest(p)

	// Check if Threads is enabled for pain point search & reply
	threadsCfg := p.GetPlatformConfig("threads")
	hasThreads := threadsCfg != nil && threadsCfg.Enabled && (threadsCfg.AccessToken != "" || os.Getenv("THREADS_ACCESS_TOKEN") != "")

	var targetPost *threads.Post
	if hasThreads {
		threadsToken := threadsCfg.AccessToken
		if threadsToken == "" {
			threadsToken = os.Getenv("THREADS_ACCESS_TOKEN")
		}
		client := threads.NewClient(threadsToken)

		// 2. Generate Search Keywords/Queries
		queryPrompt := fmt.Sprintf(`Given the product manifest:
%s

Generate 3 search keywords or short topics (maximum 3 words each) that target users would post when experiencing pain points that this product solves.
Ensure each keyword targets a different angle (e.g. tool bloat, offline speed, note-taking frustration, password/credential isolation).
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
			keywords = []string{p.Name + " alternative", "frustrated with notion", "markdown notes app"}
		}

		fmt.Printf("MarketingCell: Generated search keywords for Threads: %v\n", keywords)

		// 3. Search and Find Target Pain Point Posts
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

				// Evaluate if the post is a relevant pain point (Score 1-10)
				evalPrompt := fmt.Sprintf(`Product Manifest:
%s

User Post:
"%s"

Evaluate if the user post expresses a genuine problem, frustration, or need that our product directly addresses.
Score the post from 1 to 10 based on how likely this user would benefit from our product.
Output ONLY a single integer score from 1 to 10. Do not write anything else.`, manifest, post.Text)

				evalResp, err := c.AI.Query(evalPrompt, intent, "github-models", "")
				if err == nil {
					cleanedScore := strings.TrimSpace(evalResp)
					if score, err := strconv.Atoi(cleanedScore); err == nil && score >= 7 {
						targetPost = &post
						break
					} else if strings.Contains(cleanedScore, "7") || strings.Contains(cleanedScore, "8") || strings.Contains(cleanedScore, "9") || strings.Contains(cleanedScore, "10") {
						targetPost = &post
						break
					}
				}
			}
			if targetPost != nil {
				break
			}
		}
	}

	// 4. Executing Reply Pitch (if relevant post found on Threads)
	if targetPost != nil {
		fmt.Printf("MarketingCell: Found pain point post to reply to: %s by %s: %q\n", targetPost.ID, targetPost.Username, targetPost.Text)

		var allAssets []*media.Asset
		unposted, err := db.GetUnpostedAssets()
		if err == nil {
			allAssets = append(allAssets, unposted...)
		}

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

			intent := p.GenerationMode
			if intent == "" {
				intent = "completion"
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

		reg, _ := project.NewRegistry(config.ProjectsPath())
		cta, _ := reg.RotateCTA(p.ID)

		var assetsForReply []*media.Asset
		if selectedAsset != nil {
			assetsForReply = append(assetsForReply, selectedAsset)
			fmt.Printf("MarketingCell: Selected visual asset %s (%s) for reply\n", selectedAsset.ID, filepath.Base(selectedAsset.FilePath))
		}

		replyText, err := c.Synth.CraftReply(ctx, p, targetPost.Text, assetsForReply, recentCopies)
		if err != nil {
			return err
		}

		if cta != "" {
			replyText = replyText + " " + cta
		}

		if len(replyText) > 500 {
			replyText = replyText[:497] + "..."
		}

		var mediaURLs []string
		var cleanup func()
		if selectedAsset != nil {
			mediaURL := selectedAsset.FilePath
			if !strings.HasPrefix(selectedAsset.FilePath, "http") {
				u, cl, err := threads.HostLocalFile(selectedAsset.FilePath)
				if err == nil {
					mediaURL = u
					cleanup = cl
				}
			}
			mediaURLs = append(mediaURLs, mediaURL)
		}
		if cleanup != nil {
			defer cleanup()
		}

		postContent := platform.PostContent{
			Text:          replyText,
			MediaURLs:     mediaURLs,
			ReplyToID:     targetPost.ID,
			ReplyToAuthor: targetPost.Username,
		}

		results, errors := c.Dispatcher.Dispatch(ctx, p, postContent, "threads")
		if len(errors) > 0 && len(results) == 0 {
			return fmt.Errorf("failed replying on threads: %v", errors[0])
		}

		_ = db.MarkReplied(targetPost.ID)
		if selectedAsset != nil && len(results) > 0 {
			_ = db.MarkPosted(selectedAsset.ID, results[0].PostID, replyText)
		}
		c.Quota.RecordPublish(cont.Name)
		fmt.Printf("MarketingCell: Successfully replied to Threads post %s\n", targetPost.ID)
		return nil
	}

	// 5. Automated Multi-Platform Post Dispatch (Nostr + Threads + all enabled targets)
	fmt.Printf("MarketingCell: Dispatching broadcast post to targets (%s)...\n", strings.Join(p.GetEnabledPlatforms(), ", "))
	unposted, err := db.GetUnpostedAssets()
	var targetAsset *media.Asset
	if err == nil && len(unposted) > 0 {
		targetAsset = unposted[0]
		if err := c.validateMedia(targetAsset); err != nil {
			fmt.Printf("MarketingCell: Media validation failed for asset %s: %v\n", targetAsset.ID, err)
			_ = db.MarkPosted(targetAsset.ID, "SKIPPED_INVALID_MEDIA", "")
			targetAsset = nil
		}
	}

	reg, _ := project.NewRegistry(config.ProjectsPath())
	cta, _ := reg.RotateCTA(p.ID)

	goal := "Create an engaging marketing post highlighting product architecture and capabilities."
	var assets []*media.Asset
	if targetAsset != nil {
		assets = append(assets, targetAsset)
	}

	postText, err := c.Synth.CraftPost(ctx, p, assets, goal, cta, recentCopies)
	if err != nil {
		return err
	}

	var mediaURLs []string
	var cleanup func()
	if targetAsset != nil {
		mediaURL := targetAsset.FilePath
		if !strings.HasPrefix(targetAsset.FilePath, "http") {
			u, cl, err := threads.HostLocalFile(targetAsset.FilePath)
			if err == nil {
				mediaURL = u
				cleanup = cl
			}
		}
		mediaURLs = append(mediaURLs, mediaURL)
	}
	if cleanup != nil {
		defer cleanup()
	}

	postContent := platform.PostContent{
		Text:      postText,
		MediaURLs: mediaURLs,
	}

	results, errors := c.Dispatcher.Dispatch(ctx, p, postContent)
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Printf("MarketingCell: Target publish warning: %v\n", err)
		}
	}

	if len(results) == 0 {
		return fmt.Errorf("failed publishing post to any enabled platform target")
	}

	primaryID := results[0].PostID
	c.Quota.RecordPublish(cont.Name)
	if targetAsset != nil {
		_ = db.MarkPosted(targetAsset.ID, primaryID, postText)
	}

	for _, res := range results {
		fmt.Printf("MarketingCell: Successfully posted to [%s] (ID: %s, URL: %s)\n", strings.ToUpper(res.Platform), res.PostID, res.URL)
	}

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
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
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
