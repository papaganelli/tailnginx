package main

import (
	"flag"
	"log"

	"github.com/papaganelli/tailnginx/internal/config"
	"github.com/papaganelli/tailnginx/pkg/tailer"
	"github.com/papaganelli/tailnginx/ui"
)

func main() {
	var cfg config.Config
	flag.StringVar(&cfg.LogPath, "log", config.DefaultLogPath, "path to nginx access log")
	flag.BoolVar(&cfg.FromEnd, "from-end", true, "start tailing from end of file")
	flag.Parse()

	done := make(chan struct{})
	defer close(done)

	lines, err := tailer.TailLines(cfg.LogPath, cfg.FromEnd, done)
	if err != nil {
		log.Fatalf("failed to tail file: %v", err)
	}

	app := ui.NewApp(lines)
	if err := app.Run(); err != nil {
		log.Fatalf("app error: %v", err)
	}
}
