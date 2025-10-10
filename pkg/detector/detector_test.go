package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetBestLogFile(t *testing.T) {
	tests := []struct {
		name     string
		logs     []LogFile
		expected string
	}{
		{
			name:     "empty list",
			expected: "",
			logs:     []LogFile{},
		},
		{
			name:     "prefer main access.log",
			expected: "/var/log/nginx/access.log",
			logs: []LogFile{
				{Path: "/var/log/nginx/access.log.1", Size: 5000},
				{Path: "/var/log/nginx/access.log", Size: 1000},
				{Path: "/var/log/nginx/error.log", Size: 3000},
			},
		},
		{
			name:     "largest file when no access.log",
			expected: "/var/log/nginx/custom.log",
			logs: []LogFile{
				{Path: "/var/log/nginx/error.log", Size: 1000},
				{Path: "/var/log/nginx/custom.log", Size: 5000},
				{Path: "/var/log/nginx/debug.log", Size: 2000},
			},
		},
		{
			name:     "single file",
			expected: "/var/log/nginx/access.log",
			logs: []LogFile{
				{Path: "/var/log/nginx/access.log", Size: 1000},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBestLogFile(tt.logs)
			if result.Path != tt.expected {
				t.Errorf("GetBestLogFile() = %v, want %v", result.Path, tt.expected)
			}
		})
	}
}

func TestParseConfigFile(t *testing.T) {
	// Create a temporary test config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nginx.conf")

	configContent := `
# Test nginx config
server {
    server_name example.com;
    access_log /var/log/nginx/example.log;

    location / {
        # comment
        access_log /var/log/nginx/location.log;
    }
}

server {
    server_name test.com;
    access_log /var/log/nginx/test.log;
}

# Should be ignored
access_log off;
access_log syslog:;
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	logs := parseConfigFile(configPath)

	expectedLogs := map[string]bool{
		"/var/log/nginx/example.log":  true,
		"/var/log/nginx/location.log": true,
		"/var/log/nginx/test.log":     true,
	}

	if len(logs) != len(expectedLogs) {
		t.Errorf("Expected %d logs, got %d", len(expectedLogs), len(logs))
	}

	for path := range logs {
		if !expectedLogs[path] {
			t.Errorf("Unexpected log path found: %s", path)
		}
	}

	// Verify "off" and "syslog:" were skipped
	if _, exists := logs["off"]; exists {
		t.Error("Should not include 'off' as a log path")
	}
	if _, exists := logs["syslog:"]; exists {
		t.Error("Should not include 'syslog:' as a log path")
	}
}

func TestParseConfigFileWithServerNames(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nginx.conf")

	configContent := `
server {
    server_name api.example.com;
    access_log /var/log/nginx/api.log;
}
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	logs := parseConfigFile(configPath)

	if serverName, exists := logs["/var/log/nginx/api.log"]; !exists {
		t.Error("Expected to find /var/log/nginx/api.log")
	} else if serverName != "api.example.com" {
		t.Errorf("Expected server_name 'api.example.com', got '%s'", serverName)
	}
}

func TestParseConfigFileNonExistent(t *testing.T) {
	logs := parseConfigFile("/nonexistent/path/to/nginx.conf")
	if len(logs) != 0 {
		t.Errorf("Expected empty map for non-existent file, got %d entries", len(logs))
	}
}

func TestDetectLogFiles(t *testing.T) {
	// This test will only work if the system has nginx logs
	// We'll make it optional
	logs, err := DetectLogFiles()

	if err != nil {
		// It's okay if no logs are found on test system
		t.Logf("No nginx logs detected (expected on test systems): %v", err)
		return
	}

	if len(logs) == 0 {
		t.Error("DetectLogFiles returned no error but also no logs")
	}

	// Verify all detected logs have required fields
	for _, log := range logs {
		if log.Path == "" {
			t.Error("Log file has empty path")
		}
		if !log.Exists {
			t.Errorf("Log file %s marked as not existing", log.Path)
		}
		if log.Size < 0 {
			t.Errorf("Log file %s has negative size", log.Path)
		}
	}
}
