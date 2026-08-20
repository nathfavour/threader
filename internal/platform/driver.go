package platform

import (
	"context"
)

// PostContent represents unified content to be published across any social platform.
type PostContent struct {
	Text          string            // Plain text or markdown copy
	MediaURLs     []string          // Publicly resolved media URLs (images, videos)
	Tags          []string          // Topic hashtags / indexing tags
	ReplyToID     string            // Event ID or Post ID for threading/replies
	ReplyToAuthor string            // Optional author ID/Pubkey for targeted reply tags
	Metadata      map[string]any    // Platform-specific overrides (e.g., Nostr Kind, Content Warning)
}

// PublishResult contains the outcome of a successful publish operation on a platform.
type PublishResult struct {
	Platform string         // Platform identifier (e.g. "nostr", "threads")
	PostID   string         // Global event ID, note ID, or container ID
	URL      string         // Direct web URL to the post (if applicable)
	RawData  map[string]any // Raw response from relays/API
}

// Capabilities indicates the operational bounds and features of a platform driver.
type Capabilities struct {
	SupportsThreading  bool
	SupportsLongForm   bool
	MaxTextLength      int // 0 for unlimited / Nostr Kind 1 standard
	AcceptedMediaTypes []string
	SupportsLiveListen bool
}

// PlatformDriver defines the unified provider contract for any platform.
type PlatformDriver interface {
	// ID returns the canonical unique slug (e.g., "nostr", "threads", "bluesky")
	ID() string

	// ValidateConfig checks if the project namespace contains valid credentials/configs
	ValidateConfig(cfg map[string]string) error

	// Publish dispatches standard post content to the destination
	Publish(ctx context.Context, cfg map[string]string, content PostContent) (*PublishResult, error)

	// Capabilities returns what this driver supports
	Capabilities() Capabilities
}
