package fetcher

import (
	"context"
	"testing"
	"time"

	"oshcity-news-parser/internal/config"
)

func TestBackoffCalculation(t *testing.T) {
	cfg := &config.Config{
		HTTP: config.HttpConfig{
			BackoffMinMS: 250,
			BackoffMaxMS: 2000,
			JitterPct:    20,
		},
	}

	fetcher := NewFetcher(cfg)

	for attempt := 1; attempt <= 5; attempt++ {
		backoff := fetcher.calculateBackoff(attempt)
		if backoff < cfg.GetBackoffMin() || backoff > cfg.GetBackoffMax()*2 {
			t.Errorf("Backoff out of expected range: %v", backoff)
		}
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, 10)
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 5; i++ {
		err := rl.Wait(ctx, "example.com")
		if err != nil {
			t.Fatalf("Rate limiter error: %v", err)
		}
	}
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Logf("Rate limiter OK: 5 requests in %v", elapsed)
	}
}
