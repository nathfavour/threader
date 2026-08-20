package orchestrator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nathfavour/threader/internal/orchestrator"
	"github.com/nathfavour/threader/internal/platform"
	"github.com/nathfavour/threader/internal/project"
)

type dummyDriver struct {
	id string
}

func (d *dummyDriver) ID() string                                                               { return d.id }
func (d *dummyDriver) ValidateConfig(cfg map[string]string) error                               { return nil }
func (d *dummyDriver) Capabilities() platform.Capabilities                                      { return platform.Capabilities{} }
func (d *dummyDriver) Publish(ctx context.Context, cfg map[string]string, content platform.PostContent) (*platform.PublishResult, error) {
	return &platform.PublishResult{
		Platform: d.id,
		PostID:   "post-" + d.id + "-123",
		URL:      "https://example.com/" + d.id,
	}, nil
}

func TestDispatcher(t *testing.T) {
	platform.Register(&dummyDriver{id: "mock_plat_1"})
	platform.Register(&dummyDriver{id: "mock_plat_2"})

	tempDir, err := os.MkdirTemp("", "threader_dispatch_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	reg, err := project.NewRegistry(filepath.Join(tempDir, "projects.json"))
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	p, err := reg.Register("DispatchTest", "desc", "voice", "", "")
	if err != nil {
		t.Fatalf("failed to register project: %v", err)
	}

	// Enable mock platforms
	p.SetPlatformConfig("mock_plat_1", &project.PlatformConfig{Enabled: true})
	p.SetPlatformConfig("mock_plat_2", &project.PlatformConfig{Enabled: true})
	// Disable default nostr and threads for this test
	p.SetPlatformConfig("nostr", &project.PlatformConfig{Enabled: false})
	p.SetPlatformConfig("threads", &project.PlatformConfig{Enabled: false})

	disp := orchestrator.NewDispatcher()
	results, errors := disp.Dispatch(context.Background(), p, platform.PostContent{
		Text: "Hello cross-platform decentralized world",
	})

	if len(errors) > 0 {
		t.Fatalf("unexpected dispatch errors: %v", errors)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 publish results, got %d", len(results))
	}

	// Test targeting single platform
	resultsSingle, errorsSingle := disp.Dispatch(context.Background(), p, platform.PostContent{
		Text: "Targeted single post",
	}, "mock_plat_1")

	if len(errorsSingle) > 0 {
		t.Fatalf("unexpected single target error: %v", errorsSingle)
	}
	if len(resultsSingle) != 1 || resultsSingle[0].Platform != "mock_plat_1" {
		t.Fatalf("expected single result for mock_plat_1, got %v", resultsSingle)
	}
}
