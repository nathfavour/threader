package spine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nathfavour/threader/pkg/biology"
)

type Cell interface {
	Pulse(ctx context.Context) error
	Name() string
}

type Spine struct {
	mu     sync.RWMutex
	cells  []Cell
	rate   time.Duration
	energy biology.Energy
}

func NewSpine(rate time.Duration) *Spine {
	s := &Spine{
		cells: []Cell{},
		rate:  rate,
	}
	s.energy, _ = biology.CheckThermodynamics()
	go s.sense(context.Background())
	return s
}

func (s *Spine) Attach(cell Cell) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cells = append(s.cells, cell)
}

func (s *Spine) sense(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			energy, err := biology.CheckThermodynamics()
			if err == nil {
				s.mu.Lock()
				s.energy = energy
				s.mu.Unlock()
			}
		}
	}
}

func (s *Spine) Breathes(ctx context.Context) {
	fmt.Printf("🧵 SPINE: Starting pulse at %v\n", s.rate)

	for {
		s.mu.RLock()
		energy := s.energy
		s.mu.RUnlock()

		currentRate := s.rate
		metabolism := biology.GetMetabolism()
		_, uptime := metabolism.GetStats()
		lastActivity, _ := metabolism.GetStats()
		idleTime := time.Since(lastActivity)

		// Sleep Mode Logic
		if uptime > 1*time.Minute && idleTime > 5*time.Minute {
			// Deep Sleep: Pulse every 10 minutes
			currentRate = 10 * time.Minute
		} else if energy.EnergyLevel < 0.2 {
			// Low Power Mode: Pulse every 1 minute
			currentRate = 1 * time.Minute
		}

		if currentRate != s.rate {
			fmt.Printf("🧵 SPINE: Sleep mode active. Next pulse in %v\n", currentRate)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(currentRate):
			s.pulse(ctx)
		}
	}
}

func (s *Spine) pulse(ctx context.Context) {
	s.mu.RLock()
	cells := make([]Cell, len(s.cells))
	copy(cells, s.cells)
	s.mu.RUnlock()

	for _, cell := range cells {
		go func(c Cell) {
			if err := c.Pulse(ctx); err != nil {
				fmt.Printf("🧵 SPINE: Cell %q failed pulse: %v\n", c.Name(), err)
			}
		}(cell)
	}
}
