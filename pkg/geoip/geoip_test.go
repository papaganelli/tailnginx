package geoip

import (
	"testing"
)

func TestNewLocator(t *testing.T) {
	locator, err := NewLocator()
	if err != nil {
		t.Fatalf("NewLocator() error = %v", err)
	}
	if locator == nil {
		t.Fatal("NewLocator() returned nil locator")
	}
	defer locator.Close()
}

func TestLookupValidIPs(t *testing.T) {
	locator, err := NewLocator()
	if err != nil {
		t.Fatalf("NewLocator() error = %v", err)
	}
	defer locator.Close()

	tests := []struct {
		name string
		ip   string
		want string
	}{
		{
			name: "Google DNS (US)",
			ip:   "8.8.8.8",
			want: "US",
		},
		{
			name: "Cloudflare DNS",
			ip:   "1.1.1.1",
			want: "AU", // Cloudflare's 1.1.1.1 is registered in Australia
		},
		{
			name: "IPv6 Google DNS",
			ip:   "2001:4860:4860::8888",
			want: "US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := locator.Lookup(tt.ip)
			if err != nil {
				t.Errorf("Lookup() error = %v", err)
				return
			}
			if loc == nil {
				t.Error("Lookup() returned nil location")
				return
			}
			if loc.CountryCode != tt.want {
				t.Logf("Lookup(%s) = %s, want %s (database may vary)", tt.ip, loc.CountryCode, tt.want)
			}
		})
	}
}

func TestLookupInvalidIP(t *testing.T) {
	locator, err := NewLocator()
	if err != nil {
		t.Fatalf("NewLocator() error = %v", err)
	}
	defer locator.Close()

	tests := []struct {
		name string
		ip   string
	}{
		{
			name: "invalid format",
			ip:   "not-an-ip",
		},
		{
			name: "empty string",
			ip:   "",
		},
		{
			name: "partial IP",
			ip:   "192.168",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := locator.Lookup(tt.ip)
			if err != nil {
				t.Errorf("Lookup() should not error on invalid IP, got error = %v", err)
				return
			}
			if loc == nil {
				t.Error("Lookup() should return unknown location, not nil")
				return
			}
			if loc.Country != "Unknown" {
				t.Errorf("Lookup() for invalid IP should return 'Unknown', got %s", loc.Country)
			}
		})
	}
}

func TestLookupPrivateIPs(t *testing.T) {
	locator, err := NewLocator()
	if err != nil {
		t.Fatalf("NewLocator() error = %v", err)
	}
	defer locator.Close()

	privateIPs := []string{
		"192.168.1.1",
		"10.0.0.1",
		"172.16.0.1",
		"127.0.0.1",
	}

	for _, ip := range privateIPs {
		t.Run(ip, func(t *testing.T) {
			loc, err := locator.Lookup(ip)
			if err != nil {
				t.Errorf("Lookup() error = %v", err)
				return
			}
			if loc == nil {
				t.Error("Lookup() returned nil location")
				return
			}
			// Private IPs typically return empty or Unknown
			t.Logf("Private IP %s returned country: %s", ip, loc.Country)
		})
	}
}

func TestLookupNilLocator(t *testing.T) {
	var locator *Locator // nil locator

	loc, err := locator.Lookup("8.8.8.8")
	if err != nil {
		t.Errorf("Lookup() on nil locator should not error, got %v", err)
		return
	}
	if loc == nil {
		t.Error("Lookup() should return unknown location, not nil")
		return
	}
	if loc.Country != "Unknown" {
		t.Errorf("Lookup() on nil locator should return 'Unknown', got %s", loc.Country)
	}
}

func TestClose(t *testing.T) {
	locator, err := NewLocator()
	if err != nil {
		t.Fatalf("NewLocator() error = %v", err)
	}

	err = locator.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Should be able to close multiple times without error
	err = locator.Close()
	if err != nil {
		t.Errorf("Close() called twice should not error, got %v", err)
	}
}

func BenchmarkLookup(b *testing.B) {
	locator, err := NewLocator()
	if err != nil {
		b.Fatalf("NewLocator() error = %v", err)
	}
	defer locator.Close()

	ips := []string{
		"8.8.8.8",
		"1.1.1.1",
		"208.67.222.222",
		"2001:4860:4860::8888",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ip := ips[i%len(ips)]
		_, _ = locator.Lookup(ip)
	}
}
