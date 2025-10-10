// Package detector provides functionality to autodetect nginx log files.
package detector

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// LogFile represents a detected nginx log file.
type LogFile struct {
	Path       string
	ServerName string // Server name from config, if available
	Exists     bool
	Size       int64
}

// commonLogPaths contains standard nginx log file locations.
var commonLogPaths = []string{
	"/var/log/nginx/access.log",
	"/var/log/nginx/access.log.1",
	"/usr/local/nginx/logs/access.log",
	"/opt/nginx/logs/access.log",
	"/var/log/nginx/*.log",
}

// nginxConfigPaths contains common nginx configuration file locations.
var nginxConfigPaths = []string{
	"/etc/nginx/nginx.conf",
	"/usr/local/nginx/conf/nginx.conf",
	"/opt/nginx/conf/nginx.conf",
}

// DetectLogFiles attempts to find nginx log files on the system.
// It searches common log paths and parses nginx configuration files to discover log locations.
// Returns a slice of LogFile structs with path, size, and server name information.
// Returns an error if no nginx log files are found on the system.
func DetectLogFiles() ([]LogFile, error) {
	var logs []LogFile
	seen := make(map[string]bool)

	// 1. Check common log paths
	for _, pattern := range commonLogPaths {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Printf("Warning: failed to glob %s: %v", pattern, err)
			continue
		}
		for _, path := range matches {
			if seen[path] {
				continue
			}
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				logs = append(logs, LogFile{
					Path:   path,
					Exists: true,
					Size:   info.Size(),
				})
				seen[path] = true
			}
		}
	}

	// 2. Parse nginx config files for custom log paths
	configLogs := parseNginxConfigs()
	for _, log := range configLogs {
		if seen[log.Path] {
			continue
		}
		if info, err := os.Stat(log.Path); err == nil && !info.IsDir() {
			log.Exists = true
			log.Size = info.Size()
			logs = append(logs, log)
			seen[log.Path] = true
		}
	}

	if len(logs) == 0 {
		return nil, fmt.Errorf("no nginx log files found")
	}

	return logs, nil
}

// parseNginxConfigs parses nginx configuration files to extract access_log directives.
func parseNginxConfigs() []LogFile {
	var logs []LogFile

	for _, configPath := range nginxConfigPaths {
		// Check if config file exists
		if _, err := os.Stat(configPath); err != nil {
			continue
		}

		// Parse main config
		logPaths := parseConfigFile(configPath)
		for path, serverName := range logPaths {
			logs = append(logs, LogFile{
				Path:       path,
				ServerName: serverName,
			})
		}

		// Parse included configs (sites-enabled/*, conf.d/*)
		configDir := filepath.Dir(configPath)
		includes := []string{
			filepath.Join(configDir, "sites-enabled", "*.conf"),
			filepath.Join(configDir, "sites-enabled", "*"),
			filepath.Join(configDir, "conf.d", "*.conf"),
		}

		for _, pattern := range includes {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				log.Printf("Warning: failed to glob %s: %v", pattern, err)
				continue
			}
			for _, includePath := range matches {
				if info, err := os.Stat(includePath); err == nil && !info.IsDir() {
					includeLogs := parseConfigFile(includePath)
					for path, serverName := range includeLogs {
						logs = append(logs, LogFile{
							Path:       path,
							ServerName: serverName,
						})
					}
				}
			}
		}
	}

	return logs
}

// parseConfigFile parses a single nginx config file and extracts access_log paths.
func parseConfigFile(path string) map[string]string {
	logs := make(map[string]string)

	file, err := os.Open(path)
	if err != nil {
		return logs
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	accessLogRegex := regexp.MustCompile(`access_log\s+([^\s;]+)`)
	serverNameRegex := regexp.MustCompile(`server_name\s+([^\s;]+)`)

	var currentServerName string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Extract server_name
		if matches := serverNameRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentServerName = matches[1]
		}

		// Extract access_log
		if matches := accessLogRegex.FindStringSubmatch(line); len(matches) > 1 {
			logPath := matches[1]

			// Skip special values
			if logPath == "off" || logPath == "syslog:" {
				continue
			}

			// Store with server name context
			logs[logPath] = currentServerName
		}
	}

	return logs
}

// GetBestLogFile returns the most likely nginx log file to monitor from the provided list.
// Selection priority:
//  1. Main access.log if it exists (excluding rotated .1, .2, etc files)
//  2. Largest log file by size
//  3. First detected log file
//
// Returns an empty LogFile if the input slice is empty.
func GetBestLogFile(logs []LogFile) LogFile {
	if len(logs) == 0 {
		return LogFile{}
	}

	// Try to find main access.log
	for _, log := range logs {
		if strings.Contains(log.Path, "access.log") && !strings.Contains(log.Path, ".1") {
			return log
		}
	}

	// Return largest file
	largest := logs[0]
	for _, log := range logs[1:] {
		if log.Size > largest.Size {
			largest = log
		}
	}

	return largest
}
