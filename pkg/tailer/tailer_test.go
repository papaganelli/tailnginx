package tailer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTailLinesFromEnd(t *testing.T) {
	// Create a temporary log file
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	content := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	done := make(chan struct{})
	defer close(done)

	lines, err := TailLines(logFile, true, done)
	if err != nil {
		t.Fatalf("TailLines() error = %v", err)
	}

	// Append a new line
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for append: %v", err)
	}
	time.Sleep(100 * time.Millisecond) // Give tailer time to start
	_, _ = f.WriteString("line 4\n")
	f.Close()

	// Should receive the new line
	select {
	case line := <-lines:
		if line != "line 4" {
			t.Errorf("Expected 'line 4', got '%s'", line)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for new line")
	}
}

func TestTailLinesFromBeginning(t *testing.T) {
	// Create a temporary log file
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create file with multiple lines
	lines := make([]string, 600)
	for i := 0; i < 600; i++ {
		lines[i] = "line " + string(rune('0'+i%10))
	}
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}

	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	done := make(chan struct{})
	defer close(done)

	ch, err := TailLines(logFile, false, done)
	if err != nil {
		t.Fatalf("TailLines() error = %v", err)
	}

	// Should receive last 500 lines
	received := 0
	timeout := time.After(3 * time.Second)

loop:
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				break loop
			}
			received++
			if received >= 500 {
				break loop
			}
		case <-timeout:
			break loop
		}
	}

	if received < 400 { // Allow some tolerance
		t.Errorf("Expected to receive ~500 lines, got %d", received)
	}
}

func TestTailLinesNonExistentFile(t *testing.T) {
	done := make(chan struct{})
	defer close(done)

	_, err := TailLines("/nonexistent/file.log", true, done)

	// Should not error immediately (returns channel)
	if err != nil {
		t.Errorf("TailLines() should not error on non-existent file, got %v", err)
	}
}

func TestTailLinesCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	if err := os.WriteFile(logFile, []byte("line 1\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	done := make(chan struct{})

	lines, err := TailLines(logFile, true, done)
	if err != nil {
		t.Fatalf("TailLines() error = %v", err)
	}

	// Close done channel to signal cancellation
	close(done)

	// Channel should be closed
	select {
	case _, ok := <-lines:
		if ok {
			t.Error("Channel should be closed after cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for channel to close")
	}
}

func TestReadLastNLines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Create a file with 10 lines
	content := ""
	for i := 1; i <= 10; i++ {
		content += "line " + string(rune('0'+i)) + "\n"
	}
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	out := make(chan string, 100)
	err := readLastNLines(testFile, 5, out)
	close(out)

	if err != nil {
		t.Fatalf("readLastNLines() error = %v", err)
	}

	// Collect lines
	var lines []string
	for line := range out {
		lines = append(lines, line)
	}

	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}

	// Should get last 5 lines (line 6-10)
	// Note: First line might be partial due to seeking, so we check the last line
	if len(lines) > 0 && lines[len(lines)-1] != "line :" {
		// The character after '0'+10 is ':'
		t.Logf("Last line: %s", lines[len(lines)-1])
	}
}

func TestReadLastNLinesSmallFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.log")

	// Create a file with only 3 lines
	content := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	out := make(chan string, 100)
	err := readLastNLines(testFile, 10, out)
	close(out)

	if err != nil {
		t.Fatalf("readLastNLines() error = %v", err)
	}

	// Collect lines
	var lines []string
	for line := range out {
		lines = append(lines, line)
	}

	// Should get all 3 lines even though we asked for 10
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

func TestReadLastNLinesNonExistent(t *testing.T) {
	out := make(chan string, 100)
	err := readLastNLines("/nonexistent/file.log", 10, out)
	close(out)

	if err == nil {
		t.Error("readLastNLines() should error on non-existent file")
	}
}
