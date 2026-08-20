package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/pkg/nostrutil"
)

func TestProjectPlatformsAndTargets(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "threader_project_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dataPath := filepath.Join(tempDir, "projects.json")
	reg, err := project.NewRegistry(dataPath)
	if err != nil {
		t.Fatalf("failed to init registry: %v", err)
	}

	// Register project
	p, err := reg.Register("Product-X", "A decentralized tool", "minimalist", "https://example.com", "https://github.com/example/x")
	if err != nil {
		t.Fatalf("failed to register project: %v", err)
	}

	if !p.GetPlatformConfig("nostr").Enabled {
		t.Fatalf("expected nostr to be enabled by default")
	}
	if p.GetPlatformConfig("threads").Enabled {
		t.Fatalf("expected threads to be disabled by default until token added")
	}

	// Configure Nostr with test nsec
	testHex := "0000000000000000000000000000000000000000000000000000000000000001"
	testNsec, err := nostrutil.EncodeNsec(testHex)
	if err != nil {
		t.Fatalf("failed to encode test nsec: %v", err)
	}

	nostrCfg := p.GetPlatformConfig("nostr")
	nostrCfg.Nsec = testNsec
	nostrCfg.Enabled = true

	_, err = reg.UpdatePlatform(p.ID, "nostr", nostrCfg)
	if err != nil {
		t.Fatalf("failed to update nostr platform: %v", err)
	}

	// Reload from disk to verify persistence
	reg2, err := project.NewRegistry(dataPath)
	if err != nil {
		t.Fatalf("failed to reload registry: %v", err)
	}

	p2, ok := reg2.Get(p.ID)
	if !ok {
		t.Fatalf("failed to get project after reload")
	}

	if !p2.HasValidTarget() {
		t.Fatalf("expected project to have valid target")
	}

	enabled := p2.GetEnabledPlatforms()
	if len(enabled) != 1 || enabled[0] != "nostr" {
		t.Fatalf("expected [nostr] enabled, got %v", enabled)
	}

	driverCfg := p2.GetDriverConfig("nostr")
	if driverCfg["nsec"] != testNsec {
		t.Fatalf("expected nsec in driver config, got %v", driverCfg)
	}
	if p2.GetPlatformConfig("nostr").Npub == "" {
		t.Fatalf("expected npub to be auto-derived from nsec")
	}

	// Add Threads target
	threadsCfg := &project.PlatformConfig{
		Enabled:     true,
		AccessToken: "TH_ACCESS_123",
	}
	_, err = reg2.UpdatePlatform(p2.ID, "threads", threadsCfg)
	if err != nil {
		t.Fatalf("failed to update threads config: %v", err)
	}

	enabled2 := p2.GetEnabledPlatforms()
	if len(enabled2) != 2 {
		t.Fatalf("expected both nostr and threads enabled, got %v", enabled2)
	}
}
