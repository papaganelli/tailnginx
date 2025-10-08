package parser

import "testing"

func TestParseCombined(t *testing.T) {
	line := `127.0.0.1 - - [08/Oct/2025:12:00:00 +0000] "GET /index.html HTTP/1.1" 200 612 "-" "curl/7.68.0"`
	v := Parse(line)
	if v == nil {
		t.Fatalf("expected parse, got nil")
	}
	if v.IP != "127.0.0.1" {
		t.Fatalf("unexpected ip: %s", v.IP)
	}
	if v.Method != "GET" {
		t.Fatalf("unexpected method: %s", v.Method)
	}
	if v.Path != "/index.html" {
		t.Fatalf("unexpected path: %s", v.Path)
	}
	if v.Status != 200 {
		t.Fatalf("unexpected status: %d", v.Status)
	}
}
