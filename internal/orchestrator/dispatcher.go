package orchestrator

import (
	"context"
	"fmt"

	"github.com/nathfavour/threader/internal/platform"
	_ "github.com/nathfavour/threader/internal/platform/nostr"
	_ "github.com/nathfavour/threader/internal/platform/threads"
	"github.com/nathfavour/threader/internal/project"
)

type Dispatcher struct{}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

// Dispatch broadcasts unified content across enabled platform targets for the project.
func (d *Dispatcher) Dispatch(ctx context.Context, p *project.Project, content platform.PostContent, targetPlatforms ...string) ([]*platform.PublishResult, []error) {
	targets := targetPlatforms
	if len(targets) == 0 {
		targets = p.GetEnabledPlatforms()
	}

	if len(targets) == 0 {
		return nil, []error{fmt.Errorf("no enabled platform targets found for project %s", p.Name)}
	}

	var results []*platform.PublishResult
	var errors []error

	for _, target := range targets {
		driver, err := platform.Get(target)
		if err != nil {
			errors = append(errors, fmt.Errorf("platform driver %q not found: %w", target, err))
			continue
		}

		driverCfg := p.GetDriverConfig(target)
		if err := driver.ValidateConfig(driverCfg); err != nil {
			errors = append(errors, fmt.Errorf("invalid config for platform %q: %w", target, err))
			continue
		}

		res, err := driver.Publish(ctx, driverCfg, content)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed publishing to %q: %w", target, err))
			continue
		}

		results = append(results, res)
	}

	return results, errors
}
