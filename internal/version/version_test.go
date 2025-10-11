package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestInfo(t *testing.T) {
	info := Info()

	// Should contain all key information
	expectedFields := []string{
		"tailnginx version",
		"Git commit:",
		"Build date:",
		"Go version:",
		"OS/Arch:",
	}

	for _, field := range expectedFields {
		if !strings.Contains(info, field) {
			t.Errorf("Info() missing field %q", field)
		}
	}

	// Should contain current version
	if !strings.Contains(info, Version) {
		t.Errorf("Info() does not contain version %q", Version)
	}

	// Should contain runtime info
	if !strings.Contains(info, runtime.GOOS) {
		t.Errorf("Info() does not contain OS %q", runtime.GOOS)
	}
	if !strings.Contains(info, runtime.GOARCH) {
		t.Errorf("Info() does not contain arch %q", runtime.GOARCH)
	}
}

func TestShort(t *testing.T) {
	tests := []struct {
		name      string
		commit    string
		wantShort bool // Should return short hash
	}{
		{"dev build", "dev", false},
		{"release build", "abc123def456", true},
		{"short hash", "1234567", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original and restore after test
			origCommit := GitCommit
			defer func() { GitCommit = origCommit }()

			GitCommit = tt.commit
			short := Short()

			if tt.wantShort {
				// Should contain version and first 7 chars of commit
				expectedHash := tt.commit[:7]
				if !strings.Contains(short, Version) {
					t.Errorf("Short() = %q, should contain version %q", short, Version)
				}
				if !strings.Contains(short, expectedHash) {
					t.Errorf("Short() = %q, should contain hash %q", short, expectedHash)
				}
			} else {
				// Should only contain version
				if short != Version {
					t.Errorf("Short() = %q, want %q", short, Version)
				}
			}
		})
	}
}

func TestShortFormat(t *testing.T) {
	origCommit := GitCommit
	defer func() { GitCommit = origCommit }()

	// Test with a real commit hash
	GitCommit = "abc123def456"
	short := Short()

	expected := Version + " (abc123d)"
	if short != expected {
		t.Errorf("Short() = %q, want %q", short, expected)
	}
}

func TestGoVersion(t *testing.T) {
	if GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}

	if GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", GoVersion, runtime.Version())
	}
}

func TestVersionConstants(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// GitCommit can be "dev" or a hash
	if GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}

	// BuildDate can be "unknown" initially
	if BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
}
