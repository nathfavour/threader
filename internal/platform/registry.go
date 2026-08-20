package platform

import (
	"fmt"
	"sync"
)

var (
	mu      sync.RWMutex
	drivers = make(map[string]PlatformDriver)
)

// Register registers a platform driver into the global registry.
func Register(driver PlatformDriver) {
	mu.Lock()
	defer mu.Unlock()
	drivers[driver.ID()] = driver
}

// Get retrieves a platform driver by its unique ID.
func Get(id string) (PlatformDriver, error) {
	mu.RLock()
	defer mu.RUnlock()
	d, ok := drivers[id]
	if !ok {
		return nil, fmt.Errorf("platform driver %q not found", id)
	}
	return d, nil
}

// Drivers returns a map copy of all registered platform drivers.
func Drivers() map[string]PlatformDriver {
	mu.RLock()
	defer mu.RUnlock()
	res := make(map[string]PlatformDriver, len(drivers))
	for k, v := range drivers {
		res[k] = v
	}
	return res
}

// List returns a slice of all registered platform drivers.
func List() []PlatformDriver {
	mu.RLock()
	defer mu.RUnlock()
	list := make([]PlatformDriver, 0, len(drivers))
	for _, d := range drivers {
		list = append(list, d)
	}
	return list
}
