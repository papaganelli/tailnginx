package tailer

import (
	"io"

	"github.com/hpcloud/tail"
)

// TailLines tails the given file path and sends lines to the returned channel. Caller should close the done channel to stop.
func TailLines(path string, fromEnd bool, done <-chan struct{}) (<-chan string, error) {
	config := tail.Config{Follow: true, ReOpen: true}
	if fromEnd {
		config.Location = &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd}
	}
	t, err := tail.TailFile(path, config)
	if err != nil {
		return nil, err
	}
	out := make(chan string)
	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				t.Cleanup()
				return
			case line, ok := <-t.Lines:
				if !ok {
					return
				}
				out <- line.Text
			}
		}
	}()
	return out, nil
}
