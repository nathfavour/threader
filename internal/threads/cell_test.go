package threads

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nathfavour/threader/internal/ai"
	"github.com/nathfavour/threader/internal/container"
	"github.com/nathfavour/threader/internal/project"
	"github.com/nathfavour/threader/pkg/config"
)

func TestForcePulse(t *testing.T) {
	fmt.Println("🧵 Loading project and container...")
	m := container.NewManager(config.DataDir())
	_, err := m.GetDefault()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	reg, _ := project.NewRegistry(config.ProjectsPath())
	projects := reg.List()
	if len(projects) == 0 {
		t.Fatal("No projects found.")
	}
	p := projects[0]

	fmt.Printf("🧵 Target project: %s\n", p.Name)

	aiClient := ai.NewClient()
	cell := NewMarketingCell(aiClient)
	cell.TargetProjectID = p.ID

	// Temporarily bypass the spacing check on disk
	originalInterval := p.PostIntervalHours
	_, _ = reg.Update(p.ID, "", "", "", "", "", "", "", -1, "")
	defer func() {
		_, _ = reg.Update(p.ID, "", "", "", "", "", "", "", originalInterval, "")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err = cell.Pulse(ctx)
	if err != nil {
		t.Fatalf("❌ Pulse returned error: %v", err)
	}
	fmt.Println("✅ Pulse completed successfully.")
}
