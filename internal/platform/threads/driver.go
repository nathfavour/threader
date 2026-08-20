package threads

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nathfavour/threader/internal/platform"
	"github.com/nathfavour/threader/internal/threads"
)

type ThreadsDriver struct{}

func init() {
	platform.Register(&ThreadsDriver{})
}

func (d *ThreadsDriver) ID() string {
	return "threads"
}

func (d *ThreadsDriver) Capabilities() platform.Capabilities {
	return platform.Capabilities{
		SupportsThreading:  true,
		SupportsLongForm:   false,
		MaxTextLength:      500,
		AcceptedMediaTypes: []string{".jpg", ".jpeg", ".png", ".mp4", ".mov"},
		SupportsLiveListen: false,
	}
}

func (d *ThreadsDriver) resolveToken(cfg map[string]string) (string, error) {
	// 1. Direct access_token
	if tok, ok := cfg["access_token"]; ok && strings.TrimSpace(tok) != "" {
		return strings.TrimSpace(tok), nil
	}

	// 2. Custom access_token_env
	if envVar, ok := cfg["access_token_env"]; ok && strings.TrimSpace(envVar) != "" {
		if val := os.Getenv(envVar); strings.TrimSpace(val) != "" {
			return strings.TrimSpace(val), nil
		}
	}

	// 3. Fallback standard THREADS_ACCESS_TOKEN or THREADS_TOKEN
	if val := os.Getenv("THREADS_ACCESS_TOKEN"); strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val), nil
	}
	if val := os.Getenv("THREADS_TOKEN"); strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val), nil
	}

	return "", fmt.Errorf("no Threads access token provided or found in environment")
}

func (d *ThreadsDriver) ValidateConfig(cfg map[string]string) error {
	_, err := d.resolveToken(cfg)
	return err
}

func (d *ThreadsDriver) Publish(ctx context.Context, cfg map[string]string, content platform.PostContent) (*platform.PublishResult, error) {
	token, err := d.resolveToken(cfg)
	if err != nil {
		return nil, fmt.Errorf("threads authentication error: %w", err)
	}

	client := threads.NewClient(token)
	text := content.Text

	// Enforce 500 characters limit
	if len(text) > 500 {
		text = text[:497] + "..."
	}

	var postID string

	// Handle media if provided
	if len(content.MediaURLs) > 0 {
		mediaURL := content.MediaURLs[0]

		if content.ReplyToID != "" {
			// Reply with image
			postID, err = client.CreateImageReply(mediaURL, text, content.ReplyToID)
			if err != nil {
				return nil, fmt.Errorf("failed to create image reply: %w", err)
			}
		} else {
			// Top-level post with image
			containerID, err := client.CreateImageContainer(mediaURL, text)
			if err != nil {
				return nil, fmt.Errorf("failed to create image container: %w", err)
			}
			postID, err = client.PublishContainer(containerID)
			if err != nil {
				return nil, fmt.Errorf("failed to publish image container: %w", err)
			}
		}
	} else {
		// Text-only post or reply
		if content.ReplyToID != "" {
			postID, err = client.CreateReply(text, content.ReplyToID)
			if err != nil {
				return nil, fmt.Errorf("failed to create reply: %w", err)
			}
		} else {
			postID, err = client.CreateTextPost(text)
			if err != nil {
				return nil, fmt.Errorf("failed to create text post: %w", err)
			}
		}
	}

	return &platform.PublishResult{
		Platform: "threads",
		PostID:   postID,
		URL:      fmt.Sprintf("https://www.threads.net/t/%s", postID),
		RawData: map[string]any{
			"post_id": postID,
		},
	}, nil
}
