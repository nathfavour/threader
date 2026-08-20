package platform_test

import (
	"context"
	"testing"

	"github.com/nathfavour/threader/internal/platform"
	_ "github.com/nathfavour/threader/internal/platform/nostr"
	_ "github.com/nathfavour/threader/internal/platform/threads"
	"github.com/nathfavour/threader/pkg/nostrutil"
)

func TestPlatformRegistry(t *testing.T) {
	nostrDriver, err := platform.Get("nostr")
	if err != nil {
		t.Fatalf("expected nostr driver to be registered, got err: %v", err)
	}
	if nostrDriver.ID() != "nostr" {
		t.Fatalf("expected id nostr, got %s", nostrDriver.ID())
	}
	if !nostrDriver.Capabilities().SupportsThreading {
		t.Fatalf("expected nostr to support threading")
	}

	threadsDriver, err := platform.Get("threads")
	if err != nil {
		t.Fatalf("expected threads driver to be registered, got err: %v", err)
	}
	if threadsDriver.ID() != "threads" {
		t.Fatalf("expected id threads, got %s", threadsDriver.ID())
	}
	if threadsDriver.Capabilities().MaxTextLength != 500 {
		t.Fatalf("expected threads max length 500, got %d", threadsDriver.Capabilities().MaxTextLength)
	}

	// Generate valid test nsec
	testHex := "0000000000000000000000000000000000000000000000000000000000000001"
	validNsec, err := nostrutil.EncodeNsec(testHex)
	if err != nil {
		t.Fatalf("failed to encode test nsec: %v", err)
	}

	// Validate config tests
	err = nostrDriver.ValidateConfig(map[string]string{
		"nsec": validNsec,
	})
	if err != nil {
		t.Fatalf("expected valid nsec config to pass validation, got %v", err)
	}

	err = nostrDriver.ValidateConfig(map[string]string{})
	if err == nil {
		t.Fatalf("expected empty nostr config to fail validation")
	}

	err = threadsDriver.ValidateConfig(map[string]string{
		"access_token": "TH_TEST_TOKEN_123",
	})
	if err != nil {
		t.Fatalf("expected valid threads token config to pass validation, got %v", err)
	}

	err = threadsDriver.ValidateConfig(map[string]string{})
	if err == nil {
		t.Fatalf("expected empty threads config to fail validation")
	}
}

type mockDriver struct{}

func (m *mockDriver) ID() string                                                               { return "mock" }
func (m *mockDriver) ValidateConfig(cfg map[string]string) error                               { return nil }
func (m *mockDriver) Capabilities() platform.Capabilities                                      { return platform.Capabilities{} }
func (m *mockDriver) Publish(ctx context.Context, cfg map[string]string, content platform.PostContent) (*platform.PublishResult, error) {
	return &platform.PublishResult{Platform: "mock", PostID: "mock-123"}, nil
}

func TestCustomDriverRegistration(t *testing.T) {
	platform.Register(&mockDriver{})
	d, err := platform.Get("mock")
	if err != nil {
		t.Fatalf("expected mock driver to be registered: %v", err)
	}
	if d.ID() != "mock" {
		t.Fatalf("expected mock id, got %s", d.ID())
	}
}
