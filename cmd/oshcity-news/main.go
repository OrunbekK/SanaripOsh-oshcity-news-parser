package main

import (
	"fmt"
	"log"
	"os"

	"oshcity-news-parser/internal/config"
)

func main() {
	configPath := "configs/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("✓ Config loaded successfully\n")
	fmt.Printf("  Driver: %s\n", cfg.Storage.Driver)
	fmt.Printf("  Languages: RU=%s, KY=%s\n", cfg.BaseURLs.RU, cfg.BaseURLs.KY)
	fmt.Printf("  Rate Limit: %d concurrent, %d rpm\n", cfg.RateLimit.MaxConcurrentPerHost, cfg.RateLimit.RPM)
	fmt.Printf("  Scheduler: %s (interval: %ds)\n", cfg.Scheduler.Mode, cfg.Scheduler.IntervalS)
}
