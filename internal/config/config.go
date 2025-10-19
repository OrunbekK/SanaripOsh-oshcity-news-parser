// internal/config/config.go (ПОЛНЫЙ ОБНОВЛЕННЫЙ)
package config

import (
	"fmt"
	"time"
)

type Config struct {
	Rod                 RodConfig           `yaml:"rod"`
	Backoff             BackoffConfig       `yaml:"backoff"`
	RobotsCacheTTLHours int                 `yaml:"robots_cache_ttl_hours"`
	BaseURLs            BaseURLsConfig      `yaml:"base_urls"`
	HTTP                HttpConfig          `yaml:"http"`
	RateLimit           RateLimitConfig     `yaml:"rate_limit"`
	Pagination          PaginationConfig    `yaml:"pagination"`
	SelectorsFile       SelectorsFileConfig `yaml:"selectors_file"`
	Normalize           NormalizeConfig     `yaml:"normalize"`
	Storage             StorageConfig       `yaml:"storage"`
	Scheduler           SchedulerConfig     `yaml:"scheduler"`
	Observability       ObservabilityConfig `yaml:"observability"`
}

type BaseURLsConfig struct {
	RU string `yaml:"ru"`
	KY string `yaml:"ky"`
}

type RodConfig struct {
	Enabled          bool   `yaml:"enabled"`
	ChromePath       string `yaml:"chrome_path"`
	PageTimeoutS     int    `yaml:"page_timeout_s"`
	WaitLoadTimeoutS int    `yaml:"wait_load_timeout_s"`
	LazyLoadDelayS   int    `yaml:"lazy_load_delay_s"`
}

type BackoffConfig struct {
	MinMS     int `yaml:"min_ms"`
	MaxMS     int `yaml:"max_ms"`
	JitterPct int `yaml:"jitter_pct"`
}

type HttpConfig struct {
	UserAgent                 string `yaml:"user_agent"`
	ConnectTimeoutMS          int    `yaml:"connect_timeout_ms"`
	TotalTimeoutMS            int    `yaml:"total_timeout_ms"`
	MaxRetries                int    `yaml:"max_retries"`
	MaxIdleConnections        int    `yaml:"max_idle_connections"`
	MaxIdleConnectionsPerHost int    `yaml:"max_idle_connections_per_host"`
	IdleConnectionTimeoutS    int    `yaml:"idle_connection_timeout_s"`
	AcceptLanguageRU          string `yaml:"accept_language_ru"`
	AcceptLanguageKY          string `yaml:"accept_language_ky"`
}

type RateLimitConfig struct {
	MaxConcurrentPerHost int `yaml:"max_concurrent_per_host"`
	RPM                  int `yaml:"rpm"`
}

type PaginationConfig struct {
	Strategy              string `yaml:"strategy"`
	MaxPagesRU            int    `yaml:"max_pages_ru"`
	MaxPagesKY            int    `yaml:"max_pages_ky"`
	StopOnKnownChainPages int    `yaml:"stop_on_known_chain_pages"`
	DaysBackThreshold     int    `yaml:"days_back_threshold"`
}

type SelectorsFileConfig struct {
	RU string `yaml:"ru"`
	KY string `yaml:"ky"`
}

type NormalizeConfig struct {
	StripBlocks     []string `yaml:"strip_blocks"`
	TrimNBSP        bool     `yaml:"trim_nbsp"`
	CollapseSpaces  bool     `yaml:"collapse_spaces"`
	MaxPreviewChars int      `yaml:"max_preview_chars"`
}

type StorageConfig struct {
	Driver           string `yaml:"driver"`
	DSN              string `yaml:"dsn"`
	CommandTimeoutMS int    `yaml:"command_timeout_ms"`
	BatchSize        int    `yaml:"batch_size"`
	TxPerPage        bool   `yaml:"tx_per_page"`
}

type SchedulerConfig struct {
	Mode      string `yaml:"mode"`
	IntervalS int    `yaml:"interval_s"`
	CronExpr  string `yaml:"cron_expr"`
}

type ObservabilityConfig struct {
	LogPath     string `yaml:"log_path"`
	LogLevel    string `yaml:"log_level"`
	MetricsPath string `yaml:"metrics_path"`
}

// Validation
func (c *Config) Validate() error {
	if c.BaseURLs.RU == "" {
		return fmt.Errorf("base_urls.ru is required")
	}
	if c.BaseURLs.KY == "" {
		return fmt.Errorf("base_urls.ky is required")
	}
	if c.HTTP.UserAgent == "" {
		return fmt.Errorf("http.user_agent is required")
	}
	if c.HTTP.ConnectTimeoutMS <= 0 {
		return fmt.Errorf("http.connect_timeout_ms must be > 0")
	}
	if c.HTTP.TotalTimeoutMS <= 0 {
		return fmt.Errorf("http.total_timeout_ms must be > 0")
	}
	if c.HTTP.MaxRetries < 0 {
		return fmt.Errorf("http.max_retries must be >= 0")
	}
	if c.RateLimit.MaxConcurrentPerHost <= 0 {
		return fmt.Errorf("rate_limit.max_concurrent_per_host must be > 0")
	}
	if c.RateLimit.RPM <= 0 {
		return fmt.Errorf("rate_limit.rpm must be > 0")
	}
	if c.Pagination.MaxPagesRU <= 0 {
		return fmt.Errorf("pagination.max_pages_ru must be > 0")
	}
	if c.Pagination.MaxPagesKY <= 0 {
		return fmt.Errorf("pagination.max_pages_ky must be > 0")
	}
	if c.Storage.Driver == "" || (c.Storage.Driver != "mssql" && c.Storage.Driver != "postgres") {
		return fmt.Errorf("storage.driver must be 'mssql' or 'postgres'")
	}
	if c.Storage.DSN == "" {
		return fmt.Errorf("storage.dsn is required")
	}
	if c.Storage.CommandTimeoutMS <= 0 {
		return fmt.Errorf("storage.command_timeout_ms must be > 0")
	}
	if c.Storage.BatchSize <= 0 {
		return fmt.Errorf("storage.batch_size must be > 0")
	}
	if c.Scheduler.Mode == "" || (c.Scheduler.Mode != "interval" && c.Scheduler.Mode != "cron" && c.Scheduler.Mode != "oneshot") {
		return fmt.Errorf("scheduler.mode must be 'interval', 'cron' or 'oneshot'")
	}
	if c.Scheduler.Mode == "interval" && c.Scheduler.IntervalS <= 0 {
		return fmt.Errorf("scheduler.interval_s must be > 0 when mode is 'interval'")
	}
	if c.Scheduler.Mode == "cron" && c.Scheduler.CronExpr == "" {
		return fmt.Errorf("scheduler.cron_expr must be set when mode is 'cron'")
	}
	if c.Observability.LogPath == "" {
		return fmt.Errorf("observability.log_path is required")
	}
	if c.Observability.LogLevel == "" {
		return fmt.Errorf("observability.log_level is required")
	}
	if c.RobotsCacheTTLHours <= 0 {
		return fmt.Errorf("robots_cache_ttl_hours must be > 0")
	}
	if c.Backoff.MinMS <= 0 {
		return fmt.Errorf("backoff.min_ms must be > 0")
	}
	if c.Backoff.MaxMS <= 0 {
		return fmt.Errorf("backoff.max_ms must be > 0")
	}
	if c.Backoff.MinMS > c.Backoff.MaxMS {
		return fmt.Errorf("backoff.min_ms must be <= backoff.max_ms")
	}
	if c.Backoff.JitterPct < 0 || c.Backoff.JitterPct > 100 {
		return fmt.Errorf("backoff.jitter_pct must be between 0 and 100")
	}
	if c.Rod.Enabled {
		if c.Rod.ChromePath == "" {
			return fmt.Errorf("rod.chrome_path is required when rod.enabled is true")
		}
		if c.Rod.PageTimeoutS <= 0 {
			return fmt.Errorf("rod.page_timeout_s must be > 0")
		}
		if c.Rod.WaitLoadTimeoutS <= 0 {
			return fmt.Errorf("rod.wait_load_timeout_s must be > 0")
		}
		if c.Rod.LazyLoadDelayS < 0 {
			return fmt.Errorf("rod.lazy_load_delay_s must be >= 0")
		}
	}
	return nil
}

// Getters
func (c *Config) GetConnectTimeout() time.Duration {
	return time.Duration(c.HTTP.ConnectTimeoutMS) * time.Millisecond
}

func (c *Config) GetTotalTimeout() time.Duration {
	return time.Duration(c.HTTP.TotalTimeoutMS) * time.Millisecond
}

func (c *Config) GetIdleConnectionTimeout() time.Duration {
	return time.Duration(c.HTTP.IdleConnectionTimeoutS) * time.Second
}

func (c *Config) GetBackoffMin() time.Duration {
	return time.Duration(c.Backoff.MinMS) * time.Millisecond
}

func (c *Config) GetBackoffMax() time.Duration {
	return time.Duration(c.Backoff.MaxMS) * time.Millisecond
}

func (c *Config) GetCommandTimeout() time.Duration {
	return time.Duration(c.Storage.CommandTimeoutMS) * time.Millisecond
}

func (c *Config) GetSchedulerInterval() time.Duration {
	return time.Duration(c.Scheduler.IntervalS) * time.Second
}

func (c *Config) GetRobotsCacheTTL() time.Duration {
	return time.Duration(c.RobotsCacheTTLHours) * time.Hour
}

func (c *Config) GetRodPageTimeout() time.Duration {
	return time.Duration(c.Rod.PageTimeoutS) * time.Second
}

func (c *Config) GetRodWaitLoadTimeout() time.Duration {
	return time.Duration(c.Rod.WaitLoadTimeoutS) * time.Second
}

func (c *Config) GetRodLazyLoadDelay() time.Duration {
	return time.Duration(c.Rod.LazyLoadDelayS) * time.Second
}
