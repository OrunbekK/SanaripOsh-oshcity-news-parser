package config

import (
	"fmt"
	"time"
)

type Config struct {
	Languages           []LanguageConfig    `yaml:"languages"`
	Rod                 RodConfig           `yaml:"rod"`
	Backoff             BackoffConfig       `yaml:"backoff"`
	RobotsCacheTTLHours int                 `yaml:"robots_cache_ttl_hours"`
	HTTP                HttpConfig          `yaml:"http"`
	RateLimit           RateLimitConfig     `yaml:"rate_limit"`
	Pagination          PaginationConfig    `yaml:"pagination"`
	SelectorsFile       SelectorsFileConfig `yaml:"selectors_file"`
	Normalize           NormalizeConfig     `yaml:"normalize"`
	Storage             StorageConfig       `yaml:"storage"`
	Scheduler           SchedulerConfig     `yaml:"scheduler"`
	Observability       ObservabilityConfig `yaml:"observability"`
}

type LanguageConfig struct {
	Name           string `yaml:"name"`
	BaseURL        string `yaml:"base_url"`
	SelectorsFile  string `yaml:"selectors_file"`
	AcceptLanguage string `yaml:"accept_language"`
	MaxPages       int    `yaml:"max_pages"`
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
	Mode                     string `yaml:"mode"`
	IntervalS                int    `yaml:"interval_s"`
	CronExpr                 string `yaml:"cron_expr"`
	GracefulShutdownTimeoutS int    `yaml:"graceful_shutdown_timeout_s"`
}

type ObservabilityConfig struct {
	LogPath       string `yaml:"log_path"`
	LogLevel      string `yaml:"log_level"`
	MetricsPath   string `yaml:"metrics_path"`
	MaxLogAgeDays int    `yaml:"max_log_age_days"`
	MaxLogSizeMB  int    `yaml:"max_log_size_mb"`
	MaxBackups    int    `yaml:"max_backups"`
}

// Validation
func (c *Config) Validate() error {
	// Валидация Languages
	if len(c.Languages) == 0 {
		return fmt.Errorf("languages is required and must contain at least one language")
	}

	languageNames := make(map[string]bool)
	for i, lang := range c.Languages {
		if lang.Name == "" {
			return fmt.Errorf("languages[%d].name is required", i)
		}
		if languageNames[lang.Name] {
			return fmt.Errorf("duplicate language name: %s", lang.Name)
		}
		languageNames[lang.Name] = true

		if lang.BaseURL == "" {
			return fmt.Errorf("languages[%d].base_url is required", i)
		}
		if lang.SelectorsFile == "" {
			return fmt.Errorf("languages[%d].selectors_file is required", i)
		}
		if lang.AcceptLanguage == "" {
			return fmt.Errorf("languages[%d].accept_language is required", i)
		}
		if lang.MaxPages <= 0 {
			return fmt.Errorf("languages[%d].max_pages must be > 0", i)
		}
	}

	// Валидация HTTP
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

	// Валидация RateLimit
	if c.RateLimit.MaxConcurrentPerHost <= 0 {
		return fmt.Errorf("rate_limit.max_concurrent_per_host must be > 0")
	}
	if c.RateLimit.RPM <= 0 {
		return fmt.Errorf("rate_limit.rpm must be > 0")
	}

	// Валидация Pagination
	if c.Pagination.StopOnKnownChainPages <= 0 {
		return fmt.Errorf("pagination.stop_on_known_chain_pages must be > 0")
	}
	if c.Pagination.DaysBackThreshold < 0 {
		return fmt.Errorf("pagination.days_back_threshold must be >= 0")
	}

	// Валидация Storage
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

	// Валидация Scheduler
	if c.Scheduler.Mode == "" || (c.Scheduler.Mode != "interval" && c.Scheduler.Mode != "cron" && c.Scheduler.Mode != "oneshot") {
		return fmt.Errorf("scheduler.mode must be 'interval', 'cron' or 'oneshot'")
	}
	if c.Scheduler.Mode == "interval" && c.Scheduler.IntervalS <= 0 {
		return fmt.Errorf("scheduler.interval_s must be > 0 when mode is 'interval'")
	}
	if c.Scheduler.Mode == "cron" && c.Scheduler.CronExpr == "" {
		return fmt.Errorf("scheduler.cron_expr must be set when mode is 'cron'")
	}

	// Валидация Observability
	if c.Observability.LogPath == "" {
		return fmt.Errorf("observability.log_path is required")
	}
	if c.Observability.LogLevel == "" {
		return fmt.Errorf("observability.log_level is required")
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
