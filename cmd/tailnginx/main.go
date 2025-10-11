package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/papaganelli/tailnginx/internal/config"
	"github.com/papaganelli/tailnginx/internal/version"
	"github.com/papaganelli/tailnginx/pkg/detector"
	"github.com/papaganelli/tailnginx/pkg/geoip"
	"github.com/papaganelli/tailnginx/pkg/tailer"
	"github.com/papaganelli/tailnginx/ui"
)

func main() {
	var cfg config.Config
	var refreshMs int
	var logPath string
	var showVersion bool

	flag.StringVar(&logPath, "log", "", "path to nginx access log (auto-detect if not specified)")
	flag.IntVar(&refreshMs, "refresh", 1000, "refresh rate in milliseconds (100-10000)")
	flag.BoolVar(&showVersion, "version", false, "show version information and exit")
	flag.Parse()

	// Handle version flag
	if showVersion {
		fmt.Println(version.Info())
		os.Exit(0)
	}

	// Autodetect log file if not specified
	if logPath == "" {
		logs, err := detector.DetectLogFiles()
		if err != nil {
			log.Fatalf("Error: No nginx log files found. Please specify one with -log flag.\nTried common locations: /var/log/nginx/, /usr/local/nginx/logs/, /opt/nginx/logs/")
		}

		// Use the best detected log file
		bestLog := detector.GetBestLogFile(logs)
		cfg.LogPath = bestLog.Path
		log.Printf("Auto-detected nginx log: %s", cfg.LogPath)

		// If multiple logs found, show them
		if len(logs) > 1 {
			fmt.Fprintf(os.Stderr, "\nFound %d nginx log files:\n", len(logs))
			for i, l := range logs {
				marker := " "
				if l.Path == bestLog.Path {
					marker = "→"
				}
				serverInfo := ""
				if l.ServerName != "" {
					serverInfo = fmt.Sprintf(" (%s)", l.ServerName)
				}
				fmt.Fprintf(os.Stderr, "  %s %d. %s%s [%d KB]\n", marker, i+1, l.Path, serverInfo, l.Size/1024)
			}
			fmt.Fprintf(os.Stderr, "\nMonitoring: %s\n", bestLog.Path)
			fmt.Fprintf(os.Stderr, "Use -log flag to specify a different file\n\n")
		}
	} else {
		// Validate user-provided log path
		if err := validateLogPath(logPath); err != nil {
			log.Fatalf("Error: Invalid log path: %v", err)
		}
		cfg.LogPath = logPath
	}

	// Verify log file exists and is readable
	if info, err := os.Stat(cfg.LogPath); os.IsNotExist(err) {
		log.Fatalf("Error: Log file does not exist: %s", cfg.LogPath)
	} else if err != nil {
		log.Fatalf("Error: Cannot access log file: %v", err)
	} else if info.IsDir() {
		log.Fatalf("Error: Path is a directory, not a file: %s", cfg.LogPath)
	}

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

	// Read last 500 lines for quick startup, then tail for new entries
	lines, err := tailer.TailLines(cfg.LogPath, false, done)
	if err != nil {
		log.Fatalf("failed to tail file: %v", err)
	}

	app := ui.NewTviewApp(lines, cfg.LogPath, cfg.RefreshRate, geoLocator)
	if err := app.Run(); err != nil {
		log.Fatalf("app error: %v", err)
	}
}

// validateLogPath validates that the provided log path is safe to read.
func validateLogPath(path string) error {
	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve absolute path: %w", err)
	}

	// Check if it's a symlink (evaluate it)
	evalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// File might not exist yet, which is okay for validation
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot evaluate symlinks: %w", err)
		}
		evalPath = absPath
	}

	// Security: Prevent reading sensitive system files
	dangerousPaths := []string{
		"/etc/shadow",
		"/etc/passwd",
		"/etc/sudoers",
		"/root/.ssh",
		"/home/*/. ssh/id_rsa",
		"/.ssh/",
	}

	for _, dangerous := range dangerousPaths {
		if strings.Contains(evalPath, dangerous) {
			return fmt.Errorf("access denied: cannot read sensitive system file")
		}
	}

	// Warn if reading from unusual location (not in typical log directories)
	// This is a warning, not an error
	typicalLogDirs := []string{"/var/log", "/usr/local/nginx", "/opt/nginx", "/tmp", "."}
	isTypical := false
	for _, dir := range typicalLogDirs {
		if strings.HasPrefix(evalPath, dir) || strings.HasPrefix(absPath, dir) {
			isTypical = true
			break
		}
	}

	if !isTypical {
		log.Printf("Warning: Reading log file from unusual location: %s", evalPath)
	}

	return nil
}
