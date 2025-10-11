package ui

import (
	"testing"
	"time"

	"github.com/papaganelli/tailnginx/pkg/parser"
)

// TestGetCountryName tests the country code to name mapping.
func TestGetCountryName(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{"US code", "US", "United States"},
		{"Germany code", "DE", "Germany"},
		{"Japan code", "JP", "Japan"},
		{"Unknown code", "XX", "XX"},
		{"Empty code", "", ""},
		{"Lowercase code", "us", "us"}, // Should return code as-is (not found)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCountryName(tt.code)
			if result != tt.expected {
				t.Errorf("getCountryName(%q) = %q, want %q", tt.code, result, tt.expected)
			}
		})
	}
}

// TestApplyFilters tests the filter logic.
func TestApplyFilters(t *testing.T) {
	now := time.Now()

	// Create test app
	lines := make(chan string)
	app := NewTviewApp(lines, "/test.log", time.Second, nil)

	// Add test data
	app.allVisitors = []parser.Visitor{
		{Time: now.Add(-1 * time.Minute), Status: 200, IP: "1.2.3.4", Path: "/test1"},
		{Time: now.Add(-2 * time.Minute), Status: 404, IP: "5.6.7.8", Path: "/test2"},
		{Time: now.Add(-3 * time.Minute), Status: 500, IP: "9.10.11.12", Path: "/test3"},
		{Time: now.Add(-61 * time.Minute), Status: 200, IP: "13.14.15.16", Path: "/test4"}, // Old
	}

	tests := []struct {
		name          string
		statusFilter  int
		timeWindow    time.Duration
		expectedCount int
	}{
		{"No filters", 0, 0, 4},
		{"2xx status only", 2, 0, 2},
		{"4xx status only", 4, 0, 1},
		{"5xx status only", 5, 0, 1},
		{"Last 5 minutes", 0, 5 * time.Minute, 3},
		{"2xx + last 5 minutes", 2, 5 * time.Minute, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.statusFilter = tt.statusFilter
			app.timeWindow = tt.timeWindow
			app.applyFilters()

			if len(app.visitors) != tt.expectedCount {
				t.Errorf("applyFilters() filtered %d visitors, want %d", len(app.visitors), tt.expectedCount)
			}
		})
	}
}

// TestProcessBatch tests batch processing logic.
func TestProcessBatch(t *testing.T) {
	lines := make(chan string)
	app := NewTviewApp(lines, "/test.log", time.Second, nil)

	now := time.Now()
	batch := []parser.Visitor{
		{Time: now, Status: 200, IP: "1.2.3.4", Path: "/api", Method: "GET", Bytes: 1024},
		{Time: now, Status: 404, IP: "5.6.7.8", Path: "/missing", Method: "GET", Bytes: 512},
		{Time: now, Status: 200, IP: "1.2.3.4", Path: "/api", Method: "POST", Bytes: 2048},
	}

	app.processBatch(batch)

	// Check that visitors were added
	if len(app.allVisitors) != 3 {
		t.Errorf("processBatch() added %d visitors, want 3", len(app.allVisitors))
	}

	// Check that rate tracker recorded them
	stats := app.rateTracker.GetStats()
	if stats.Total != 3 {
		t.Errorf("rateTracker.Total = %d, want 3", stats.Total)
	}

	// Check dataChanged flag
	if !app.dataChanged {
		t.Error("processBatch() should set dataChanged to true")
	}
}

// TestProcessBatchMemoryLimit tests that old visitors are removed.
func TestProcessBatchMemoryLimit(t *testing.T) {
	lines := make(chan string)
	app := NewTviewApp(lines, "/test.log", time.Second, nil)

	now := time.Now()

	// Add 10,001 visitors (exceeds 10,000 limit)
	batch := make([]parser.Visitor, 10001)
	for i := range batch {
		batch[i] = parser.Visitor{
			Time:   now,
			Status: 200,
			IP:     "1.2.3.4",
			Path:   "/test",
			Method: "GET",
			Bytes:  1024,
		}
	}

	app.processBatch(batch)

	// Should keep only last 10,000
	if len(app.allVisitors) != 10000 {
		t.Errorf("processBatch() kept %d visitors, want 10000", len(app.allVisitors))
	}
}

// TestNewTviewApp tests app initialization.
func TestNewTviewApp(t *testing.T) {
	lines := make(chan string)
	logPath := "/var/log/nginx/access.log"
	refreshRate := 500 * time.Millisecond

	app := NewTviewApp(lines, logPath, refreshRate, nil)

	if app == nil {
		t.Fatal("NewTviewApp() returned nil")
	}

	if app.logFilePath != logPath {
		t.Errorf("app.logFilePath = %q, want %q", app.logFilePath, logPath)
	}

	if app.refreshRate != refreshRate {
		t.Errorf("app.refreshRate = %v, want %v", app.refreshRate, refreshRate)
	}

	if app.timeWindow != 0 {
		t.Errorf("app.timeWindow = %v, want 0 (all time)", app.timeWindow)
	}

	if app.timeWindowIndex != len(timeWindowPresets)-1 {
		t.Errorf("app.timeWindowIndex = %d, want %d (all time)", app.timeWindowIndex, len(timeWindowPresets)-1)
	}

	if app.rateTracker == nil {
		t.Error("app.rateTracker should not be nil")
	}

	// Check that maps are initialized
	if app.statusCodes == nil {
		t.Error("app.statusCodes map not initialized")
	}
	if app.pathsData == nil {
		t.Error("app.pathsData map not initialized")
	}
	if app.ips == nil {
		t.Error("app.ips map not initialized")
	}
}

// TestTimeWindowPresets tests that time window presets are correctly defined.
func TestTimeWindowPresets(t *testing.T) {
	expected := []int{5, 30, 60, 180, 720, 1440, 10080, 43200, 0}

	if len(timeWindowPresets) != len(expected) {
		t.Fatalf("timeWindowPresets length = %d, want %d", len(timeWindowPresets), len(expected))
	}

	for i, preset := range timeWindowPresets {
		if preset != expected[i] {
			t.Errorf("timeWindowPresets[%d] = %d, want %d", i, preset, expected[i])
		}
	}
}

// BenchmarkApplyFilters benchmarks the filter performance.
func BenchmarkApplyFilters(b *testing.B) {
	lines := make(chan string)
	app := NewTviewApp(lines, "/test.log", time.Second, nil)

	now := time.Now()
	// Create 1000 visitors
	app.allVisitors = make([]parser.Visitor, 1000)
	for i := range app.allVisitors {
		app.allVisitors[i] = parser.Visitor{
			Time:   now.Add(-time.Duration(i) * time.Second),
			Status: 200 + (i % 5 * 100), // Mix of status codes
			IP:     "1.2.3.4",
			Path:   "/test",
		}
	}

	app.timeWindow = 10 * time.Minute
	app.statusFilter = 0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.applyFilters()
	}
}

// BenchmarkProcessBatch benchmarks batch processing performance.
func BenchmarkProcessBatch(b *testing.B) {
	lines := make(chan string)
	app := NewTviewApp(lines, "/test.log", time.Second, nil)

	now := time.Now()
	batch := make([]parser.Visitor, 100)
	for i := range batch {
		batch[i] = parser.Visitor{
			Time:   now,
			Status: 200,
			IP:     "1.2.3.4",
			Path:   "/test",
			Method: "GET",
			Bytes:  1024,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.processBatch(batch)
	}
}
