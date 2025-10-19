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
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"

	"oshcity-news-parser/internal/config"
	"oshcity-news-parser/internal/observability"
)

type Fetcher struct {
	client      *http.Client
	cfg         *config.Config
	logger      *observability.Logger
	robotsCache *RobotsCache
	rateLimiter *RateLimiter
	browser     *rod.Browser
	useRod      bool
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
			MaxIdleConns:        cfg.HTTP.MaxIdleConnections,
			MaxIdleConnsPerHost: cfg.HTTP.MaxIdleConnectionsPerHost,
			IdleConnTimeout:     cfg.GetIdleConnectionTimeout(),
		},
	}

	fetcher := &Fetcher{
		client:      client,
		cfg:         cfg,
		logger:      logger,
		robotsCache: NewRobotsCache(12 * time.Hour),
		rateLimiter: NewRateLimiter(cfg.RateLimit.MaxConcurrentPerHost, cfg.RateLimit.RPM),
		useRod:      true,
	}

	// Инициализируем Rod браузер
	if fetcher.useRod {
		fetcher.initRod()
	}

	return fetcher
}

func (f *Fetcher) initRod() {
	defer func() {
		if r := recover(); r != nil {
			f.logger.Error("Failed to initialize Rod", "error", fmt.Sprintf("%v", r))
			f.useRod = false
		}
	}()

	// Используем установленный Chrome вместо скачивания
	u, err := launcher.New().
		Bin(f.cfg.Rod.ChromePath).
		Launch()

	if err != nil {
		f.logger.Error("Failed to launch browser", "error", err.Error())
		f.useRod = false
		return
	}

	f.browser = rod.New().ControlURL(u)
	if err := f.browser.Connect(); err != nil {
		f.logger.Error("Failed to connect to browser", "error", err.Error())
		f.useRod = false
		return
	}

	f.logger.Info("Rod browser initialized successfully")
}

func (f *Fetcher) Close() error {
	if f.browser != nil {
		return f.browser.Close()
	}
	return nil
}

func (f *Fetcher) Fetch(ctx context.Context, urlStr string, acceptLanguage string) (*FetchResponse, error) {
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

		resp, err := f.fetchOnce(ctx, urlStr, acceptLanguage)
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
	// Если Rod доступен и инициализирован, используем его
	if f.useRod && f.browser != nil {
		return f.fetchWithRod(ctx, urlStr, lang)
	}

	// Иначе используем обычный HTTP
	return f.fetchWithHTTP(ctx, urlStr, lang)
}

func (f *Fetcher) fetchWithRod(ctx context.Context, urlStr string, lang string) (*FetchResponse, error) {
	f.logger.Debug("Fetching with Rod", "url", urlStr)

	page := f.browser.MustPage(urlStr)
	defer func() {
		_ = page.Close()
	}()

	// Ждём загрузки с timeout
	if err := page.Timeout(time.Duration(f.cfg.Rod.WaitLoadTimeoutS) * time.Second).WaitLoad(); err != nil {
		return nil, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Задержка для lazy-load
	time.Sleep(time.Duration(f.cfg.Rod.LazyLoadDelayS) * time.Second)

	// Получаем полный HTML
	html := page.MustHTML()

	f.logger.Info("Fetched with Rod successfully", "size", len(html))

	return &FetchResponse{
		StatusCode: 200,
		Body:       []byte(html),
		URL:        urlStr,
		Headers:    make(http.Header),
	}, nil
}

func (f *Fetcher) fetchWithHTTP(ctx context.Context, urlStr string, acceptLanguage string) (*FetchResponse, error) {
	f.logger.Info("Fetching with HTTP", "url", urlStr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", f.cfg.HTTP.UserAgent)
	req.Header.Set("Accept-Language", acceptLanguage)
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

	f.logger.Info("Fetched with HTTP successfully", "size", len(body))

	return &FetchResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		URL:        resp.Request.URL.String(),
		Headers:    resp.Header,
	}, nil
}

func (f *Fetcher) calculateBackoff(attempt int) time.Duration {
	minMS := f.cfg.Backoff.MinMS
	maxMS := f.cfg.Backoff.MaxMS
	jitterPct := f.cfg.Backoff.JitterPct

	exponential := minMS * (1 << uint(attempt-1))
	if exponential > maxMS {
		exponential = maxMS
	}

	jitterRange := float64(exponential) * float64(jitterPct) / 100
	jitter := (rand.Float64() - 0.5) * 2 * jitterRange
	finalMS := float64(exponential) + jitter

	if finalMS < float64(minMS) {
		finalMS = float64(minMS)
	}

	return time.Duration(math.Max(finalMS, 0)) * time.Millisecond
}
