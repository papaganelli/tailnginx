// Package tailer provides functionality to tail log files in real-time.
package tailer

import (
	"bufio"
	"io"
	"os"

	"github.com/nxadm/tail"
)

// TailLines tails the given file path and sends lines to the returned channel.
// If fromEnd is false, it first reads the last 500 lines before tailing new entries.
// If fromEnd is true, it only tails new entries from the end of the file.
// The done channel should be closed by the caller to stop tailing and cleanup resources.
// Returns a channel that will be closed when tailing stops or an error occurs.
func TailLines(path string, fromEnd bool, done <-chan struct{}) (<-chan string, error) {
	out := make(chan string, 1000) // Buffered channel for better performance

	// Always start tailing immediately, then async load historical data
	go func() {
		// First, read last 500 lines in background and send them quickly
		if !fromEnd {
			_ = readLastNLines(path, 500, out)
		}
		// Then start tailing - use fromEnd parameter to determine if we tail from end or continue from current position
		startTailing(path, fromEnd, out, done)
	}()

	return out, nil
}

// readLastNLines reads the last N lines from a file and sends to channel
func readLastNLines(path string, n int, out chan<- string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	fileSize := stat.Size()

	// Estimate bytes to read (avg 150 bytes per line)
	estimatedBytes := int64(n * 150)
	if estimatedBytes > fileSize {
		estimatedBytes = fileSize
	}

	// Seek to near end
	offset := fileSize - estimatedBytes
	if offset < 0 {
		offset = 0
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	// Read lines with buffer limits to prevent memory exhaustion
	scanner := bufio.NewScanner(file)
	// Set max line length to 1MB (nginx default max is typically 4-8KB)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Check for scanner errors (e.g., line too long)
	if err := scanner.Err(); err != nil {
		return err
	}

	// Send last N lines
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	for i := start; i < len(lines); i++ {
		select {
		case out <- lines[i]:
		default:
			return nil // Channel full, skip
		}
	}

	return nil
}

// startTailing starts tailing the file
func startTailing(path string, fromEnd bool, out chan<- string, done <-chan struct{}) {
	config := tail.Config{Follow: true, ReOpen: true, Logger: tail.DiscardingLogger}
	if fromEnd {
		config.Location = &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd}
	} else {
		config.Location = &tail.SeekInfo{Offset: 0, Whence: io.SeekStart}
	}
	t, err := tail.TailFile(path, config)
	if err != nil {
		close(out)
		return
	}

	for {
		select {
		case <-done:
			t.Cleanup()
			close(out)
			return
		case line, ok := <-t.Lines:
			if !ok {
				close(out)
				return
			}
			out <- line.Text
		}
	}
}
