package main

import (
	"flag"
	"log"
	"time"

	"github.com/papaganelli/tailnginx/internal/config"
	"github.com/papaganelli/tailnginx/pkg/geoip"
	"github.com/papaganelli/tailnginx/pkg/tailer"
	"github.com/papaganelli/tailnginx/ui"
)

func main() {
	var cfg config.Config
	var refreshMs int

	flag.StringVar(&cfg.LogPath, "log", config.DefaultLogPath, "path to nginx access log")
	flag.BoolVar(&cfg.FromEnd, "from-end", true, "start tailing from end of file")
	flag.IntVar(&refreshMs, "refresh", 1000, "refresh rate in milliseconds (100-10000)")
	flag.Parse()

	// Convert milliseconds to duration and validate
	cfg.RefreshRate = time.Duration(refreshMs) * time.Millisecond
	if cfg.RefreshRate < config.MinRefreshRate {
		cfg.RefreshRate = config.MinRefreshRate
	}
	if cfg.RefreshRate > config.MaxRefreshRate {
		cfg.RefreshRate = config.MaxRefreshRate
	}

	// Initialize GeoIP locator with automatic database management
	geoLocator, err := geoip.NewLocator()
	if err != nil {
		geoLocator = nil
	} else {
		defer geoLocator.Close()
	}

	done := make(chan struct{})
	defer close(done)

	lines, err := tailer.TailLines(cfg.LogPath, cfg.FromEnd, done)
	if err != nil {
		log.Fatalf("failed to tail file: %v", err)
	}

	app := ui.NewTviewApp(lines, cfg.RefreshRate, geoLocator)
	if err := app.Run(); err != nil {
		log.Fatalf("app error: %v", err)
	}
}
