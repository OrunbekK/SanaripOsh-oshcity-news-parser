package fetcher

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"oshcity-news-parser/internal/observability"
)

type RobotsCache struct {
	cache  map[string]*RobotsTxt
	ttl    time.Duration
	mu     sync.RWMutex
	logger *observability.Logger
}

type RobotsTxt struct {
	content   string
	expiresAt time.Time
}

func NewRobotsCache(ttl time.Duration) *RobotsCache {
	return &RobotsCache{
		cache: make(map[string]*RobotsTxt),
		ttl:   ttl,
	}
}

func (rc *RobotsCache) IsAllowed(ctx context.Context, host, urlStr string, client *http.Client) (bool, error) {
	rc.mu.RLock()
	cached, exists := rc.cache[host]
	rc.mu.RUnlock()

	if exists && time.Now().Before(cached.expiresAt) {
		// Cache hit
		return !isDisallowedByRobots(cached.content, urlStr), nil
	}

	// Fetch robots.txt
	robotsURL := fmt.Sprintf("https://%s/robots.txt", host)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		// If we can't fetch, assume allowed
		return true, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		// Network error: assume allowed
		return true, nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		// No robots.txt: assume allowed
		return true, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return true, nil
	}

	content := string(body)

	// Cache it
	rc.mu.Lock()
	rc.cache[host] = &RobotsTxt{
		content:   content,
		expiresAt: time.Now().Add(rc.ttl),
	}
	rc.mu.Unlock()

	return !isDisallowedByRobots(content, urlStr), nil
}

func isDisallowedByRobots(robotsContent, urlStr string) bool {
	// TODO: implement full robots.txt parsing with proper User-agent matching
	// For now: allow all (production: use github.com/viktorruskai/urllib)
	return false
}
