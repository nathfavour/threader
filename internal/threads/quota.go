package threads

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Quota struct {
	PublishCount int       `json:"publish_count"`
	LastReset    time.Time `json:"last_reset"`
}

type QuotaManager struct {
	mu       sync.Mutex
	dir      string
	maxPosts int
}

func NewQuotaManager(dataDir string) *QuotaManager {
	dir := filepath.Join(dataDir, "quotas")
	_ = os.MkdirAll(dir, 0755)
	return &QuotaManager{
		dir:      dir,
		maxPosts: 250, // Standard developer limit
	}
}

func (q *QuotaManager) CanPublish(containerName string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	quota := q.load(containerName)
	if time.Since(quota.LastReset) > 24*time.Hour {
		quota.PublishCount = 0
		quota.LastReset = time.Now()
		_ = q.save(containerName, quota)
	}

	return quota.PublishCount < q.maxPosts
}

func (q *QuotaManager) RecordPublish(containerName string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	quota := q.load(containerName)
	quota.PublishCount++
	_ = q.save(containerName, quota)
}

func (q *QuotaManager) load(name string) Quota {
	path := filepath.Join(q.dir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Quota{LastReset: time.Now()}
	}

	var quota Quota
	if json.Unmarshal(data, &quota) != nil {
		return Quota{LastReset: time.Now()}
	}
	return quota
}

func (q *QuotaManager) save(name string, quota Quota) error {
	path := filepath.Join(q.dir, name+".json")
	data, _ := json.MarshalIndent(quota, "", "  ")
	return os.WriteFile(path, data, 0644)
}

func (q *QuotaManager) Status(name string) string {
	quota := q.load(name)
	return fmt.Sprintf("%d/%d posts used today (Resets in %v)", 
		quota.PublishCount, q.maxPosts, time.Until(quota.LastReset.Add(24*time.Hour)).Round(time.Minute))
}
