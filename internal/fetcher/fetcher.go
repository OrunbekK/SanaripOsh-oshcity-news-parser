package fetcher

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"oshcity-news-parser/internal/observability"
	"time"

	"oshcity-news-parser/internal/config"
)

type Fetcher struct {
	client      *http.Client
	cfg         *config.Config
	logger      *observability.Logger
	robotsCache *RobotsCache
	rateLimiter *RateLimiter
}

type FetchResponse struct {
	StatusCode int
	Body       []byte
	URL        string
	Headers    http.Header
}

func NewFetcher(cfg *config.Config, logger *observability.Logger) *Fetcher {
	client := &http.Client{
		Timeout: cfg.GetTotalTimeout(),
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &Fetcher{
		client:      client,
		cfg:         cfg,
		logger:      logger,
		robotsCache: NewRobotsCache(12 * time.Hour),
		rateLimiter: NewRateLimiter(cfg.RateLimit.MaxConcurrentPerHost, cfg.RateLimit.RPM),
	}
}

func (f *Fetcher) Fetch(ctx context.Context, urlStr string, lang string) (*FetchResponse, error) {
	// Parse URL to get host
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	host := parsedURL.Host

	// Check robots.txt
	allowed, err := f.robotsCache.IsAllowed(ctx, host, urlStr, f.client)
	if err != nil {
		return nil, fmt.Errorf("robots.txt check failed: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("URL disallowed by robots.txt: %s", urlStr)
	}

	// Apply rate limiting
	if err := f.rateLimiter.Wait(ctx, host); err != nil {
		return nil, fmt.Errorf("rate limit error: %w", err)
	}

	// Fetch with retries
	var lastErr error
	for attempt := 0; attempt <= f.cfg.HTTP.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := f.calculateBackoff(attempt)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := f.fetchOnce(ctx, urlStr, lang)
		if err != nil {
			lastErr = err
			continue
		}

		// Retry on 5xx or 429
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			if attempt < f.cfg.HTTP.MaxRetries {
				continue
			}
		}

		return resp, nil
	}

	return nil, fmt.Errorf("fetch failed after %d retries: %w", f.cfg.HTTP.MaxRetries, lastErr)
}

func (f *Fetcher) fetchOnce(ctx context.Context, urlStr string, lang string) (*FetchResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", f.cfg.HTTP.UserAgent)
	req.Header.Set("Accept-Language", f.getAcceptLanguage(lang))
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Connection", "keep-alive")

	// Set connect timeout separately
	ctx, cancel := context.WithTimeout(ctx, f.cfg.GetConnectTimeout())
	defer cancel()

	resp, err := f.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	reader := resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer func() { _ = gzipReader.Close() }()
		reader = gzipReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// LOG: проверяем как пришёл ответ
	f.logger.Debug("Response Headers:\n")
	f.logger.Debug("  Content-Encoding: %s\n", resp.Header.Get("Content-Encoding"))
	f.logger.Debug("  Content-Type: %s\n", resp.Header.Get("Content-Type"))
	f.logger.Debug("  Content-Length: %s\n", resp.Header.Get("Content-Length"))
	f.logger.Debug("  Actual Body Size: %d bytes\n", len(body))

	return &FetchResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		URL:        resp.Request.URL.String(),
		Headers:    resp.Header,
	}, nil
}

func (f *Fetcher) calculateBackoff(attempt int) time.Duration {
	minMS := f.cfg.HTTP.BackoffMinMS
	maxMS := f.cfg.HTTP.BackoffMaxMS
	jitterPct := f.cfg.HTTP.JitterPct

	// Exponential backoff: min * 2^attempt
	exponential := minMS * (1 << uint(attempt-1))
	if exponential > maxMS {
		exponential = maxMS
	}

	// Apply jitter: ±jitterPct%
	jitterRange := float64(exponential) * float64(jitterPct) / 100
	jitter := (rand.Float64() - 0.5) * 2 * jitterRange
	finalMS := float64(exponential) + jitter

	if finalMS < float64(minMS) {
		finalMS = float64(minMS)
	}

	return time.Duration(math.Max(finalMS, 0)) * time.Millisecond
}

func (f *Fetcher) getAcceptLanguage(lang string) string {
	if lang == "ky" {
		return f.cfg.HTTP.AcceptLanguageKY
	}
	return f.cfg.HTTP.AcceptLanguageRU
}
