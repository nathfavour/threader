package nostr

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nathfavour/threader/internal/platform"
	"github.com/nathfavour/threader/pkg/nostrutil"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

var DefaultRelays = []string{
	"wss://relay.damus.io",
	"wss://nos.lol",
	"wss://relay.nostr.band",
	"wss://relay.primal.net",
}

type NostrDriver struct{}

func init() {
	platform.Register(&NostrDriver{})
}

func (d *NostrDriver) ID() string {
	return "nostr"
}

func (d *NostrDriver) Capabilities() platform.Capabilities {
	return platform.Capabilities{
		SupportsThreading:  true,
		SupportsLongForm:   true,
		MaxTextLength:      0, // Unlimited
		AcceptedMediaTypes: []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".mp4", ".mov"},
		SupportsLiveListen: true,
	}
}

func (d *NostrDriver) resolvePrivateKey(cfg map[string]string) (string, error) {
	// 1. Direct nsec/hex
	if key, ok := cfg["nsec"]; ok && strings.TrimSpace(key) != "" {
		return nostrutil.ParsePrivateKey(key)
	}

	// 2. Custom nsec env var
	if envVar, ok := cfg["nsec_env"]; ok && strings.TrimSpace(envVar) != "" {
		if val := os.Getenv(envVar); strings.TrimSpace(val) != "" {
			return nostrutil.ParsePrivateKey(val)
		}
	}

	// 3. Fallback standard NOSTR_NSEC env var
	if val := os.Getenv("NOSTR_NSEC"); strings.TrimSpace(val) != "" {
		return nostrutil.ParsePrivateKey(val)
	}

	// 4. Fallback NOSTR_PRIVATE_KEY env var
	if val := os.Getenv("NOSTR_PRIVATE_KEY"); strings.TrimSpace(val) != "" {
		return nostrutil.ParsePrivateKey(val)
	}

	return "", fmt.Errorf("no Nostr private key (nsec) provided or found in environment")
}

func (d *NostrDriver) resolveRelays(cfg map[string]string) []string {
	if relaysStr, ok := cfg["relays"]; ok && strings.TrimSpace(relaysStr) != "" {
		parts := strings.Split(relaysStr, ",")
		var cleaned []string
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		if len(cleaned) > 0 {
			return cleaned
		}
	}
	return DefaultRelays
}

func (d *NostrDriver) ValidateConfig(cfg map[string]string) error {
	_, err := d.resolvePrivateKey(cfg)
	return err
}

func (d *NostrDriver) Publish(ctx context.Context, cfg map[string]string, content platform.PostContent) (*platform.PublishResult, error) {
	privKeyHex, err := d.resolvePrivateKey(cfg)
	if err != nil {
		return nil, fmt.Errorf("nostr authentication error: %w", err)
	}

	pubKeyHex, err := nostrutil.GetPublicKeyHex(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to derive public key: %w", err)
	}

	relays := d.resolveRelays(cfg)

	// Construct post text with media links if provided
	fullContent := content.Text
	if len(content.MediaURLs) > 0 {
		var mediaLinks []string
		for _, u := range content.MediaURLs {
			if !strings.Contains(fullContent, u) {
				mediaLinks = append(mediaLinks, u)
			}
		}
		if len(mediaLinks) > 0 {
			if fullContent != "" {
				fullContent += "\n\n"
			}
			fullContent += strings.Join(mediaLinks, "\n")
		}
	}

	// Determine kind
	kind := nostr.KindTextNote
	if kVal, ok := content.Metadata["kind"]; ok {
		if kInt, ok := kVal.(int); ok {
			kind = kInt
		}
	}

	ev := nostr.Event{
		PubKey:    pubKeyHex,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      kind,
		Tags:      nostr.Tags{},
		Content:   fullContent,
	}

	// Tags
	for _, tag := range content.Tags {
		cleanTag := strings.TrimPrefix(strings.TrimSpace(tag), "#")
		if cleanTag != "" {
			ev.Tags = append(ev.Tags, nostr.Tag{"t", cleanTag})
		}
	}

	// Thread / Reply tags (NIP-10)
	if content.ReplyToID != "" {
		ev.Tags = append(ev.Tags, nostr.Tag{"e", content.ReplyToID, "", "reply"})
	}
	if content.ReplyToAuthor != "" {
		ev.Tags = append(ev.Tags, nostr.Tag{"p", content.ReplyToAuthor})
	}

	// Client attribution tag
	ev.Tags = append(ev.Tags, nostr.Tag{"client", "threader"})

	// Sign event
	if err := ev.Sign(privKeyHex); err != nil {
		return nil, fmt.Errorf("failed to sign nostr event: %w", err)
	}

	// Broadcast via relay pool
	pool := nostr.NewSimplePool(ctx)
	publishCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	relayResponses := make(map[string]any)
	successCount := 0

	for _, relayURL := range relays {
		relay, err := pool.EnsureRelay(relayURL)
		if err != nil {
			relayResponses[relayURL] = fmt.Sprintf("connection error: %v", err)
			continue
		}

		err = relay.Publish(publishCtx, ev)
		if err != nil {
			relayResponses[relayURL] = fmt.Sprintf("publish error: %v", err)
		} else {
			relayResponses[relayURL] = "OK"
			successCount++
		}
	}

	if successCount == 0 && len(relays) > 0 {
		return nil, fmt.Errorf("failed to publish to any Nostr relay (tried %d relays: %v)", len(relays), relayResponses)
	}

	noteID, err := nip19.EncodeNote(ev.ID)
	if err != nil {
		noteID = ev.ID
	}

	postURL := fmt.Sprintf("https://njump.me/%s", noteID)

	return &platform.PublishResult{
		Platform: "nostr",
		PostID:   ev.ID,
		URL:      postURL,
		RawData: map[string]any{
			"event_id":      ev.ID,
			"note_id":       noteID,
			"pubkey":        pubKeyHex,
			"kind":          ev.Kind,
			"relays":        relays,
			"relay_results": relayResponses,
			"success_count": successCount,
		},
	}, nil
}
