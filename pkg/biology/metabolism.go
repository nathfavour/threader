package biology

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Energy struct {
	EnergyLevel float64 // 0.0 to 1.0
	CPUUsage    float64 // percentage
	MemoryUsage float64 // percentage
}

type Metabolism struct {
	mu           sync.RWMutex
	LastActivity time.Time
	Uptime       time.Time
}

var (
	m    *Metabolism
	once sync.Once
)

func GetMetabolism() *Metabolism {
	once.Do(func() {
		m = &Metabolism{
			LastActivity: time.Now(),
			Uptime:       time.Now(),
		}
	})
	return m
}

func (m *Metabolism) RecordActivity() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastActivity = time.Now()
}

func (m *Metabolism) GetStats() (time.Time, time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.LastActivity, time.Since(m.Uptime)
}

func CheckThermodynamics() (Energy, error) {
	c, _ := cpu.Percent(0, false)
	v, _ := mem.VirtualMemory()

	cpuVal := 0.0
	if len(c) > 0 {
		cpuVal = c[0]
	}

	memVal := 0.0
	if v != nil {
		memVal = v.UsedPercent
	}

	// Simple energy level calculation
	// High CPU or High Mem reduces energy
	level := 1.0 - (cpuVal/200.0) - (memVal/200.0)
	if level < 0 {
		level = 0
	}

	return Energy{
		EnergyLevel: level,
		CPUUsage:    cpuVal,
		MemoryUsage: memVal,
	}, nil
}

func ShouldApoptose() bool {
	// Autonomous death if resources are critically low
	e, _ := CheckThermodynamics()
	return e.EnergyLevel < 0.05
}

func Apoptosis(reason string) {
	// Clean exit
	osExit(reason)
}

func osExit(reason string) {
	// This is a placeholder for actual process termination
	panic("Apoptosis: " + reason)
}
