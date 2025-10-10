// Package geoip provides IP geolocation functionality using an embedded database.
package geoip

import (
	"net"
	"sync"

	"github.com/phuslu/iploc"
)

// Locator provides IP geolocation lookups using an embedded database with caching
type Locator struct {
	cache sync.Map // map[string]*Location for concurrent access
}

// Location represents a geographic location
type Location struct {
	Country     string
	CountryCode string
	City        string
}

// NewLocator creates a new Locator instance with an embedded GeoIP database.
// The database is embedded in the binary, so no external files are required.
// Returns a Locator ready for IP lookups with an internal cache for performance.
func NewLocator() (*Locator, error) {
	// iploc has embedded data, no initialization needed
	return &Locator{}, nil
}

// Lookup looks up the geographic location for an IP address with caching.
// Results are cached internally to improve performance for repeated lookups.
// Returns a Location with country information, or "Unknown" if the IP cannot be located.
// IPv4 and IPv6 addresses are both supported.
func (l *Locator) Lookup(ipStr string) (*Location, error) {
	if l == nil {
		return &Location{
			Country:     "Unknown",
			CountryCode: "??",
			City:        "Unknown",
		}, nil
	}

	// Check cache first
	if cached, ok := l.cache.Load(ipStr); ok {
		return cached.(*Location), nil
	}

	// Parse IP address
	ip := net.ParseIP(ipStr)
	if ip == nil {
		loc := &Location{
			Country:     "Unknown",
			CountryCode: "??",
			City:        "Unknown",
		}
		// Cache invalid IPs too to avoid repeated parsing
		l.cache.Store(ipStr, loc)
		return loc, nil
	}

	// Use iploc to get country code
	country := iploc.Country(ip)

	var loc *Location
	if country == "" {
		loc = &Location{
			Country:     "Unknown",
			CountryCode: "??",
			City:        "Unknown",
		}
	} else {
		loc = &Location{
			Country:     country,
			CountryCode: country,
			City:        "Unknown", // iploc only provides country
		}
	}

	// Store in cache
	l.cache.Store(ipStr, loc)

	return loc, nil
}

// Close closes the locator and releases any resources.
// This is a no-op for the embedded database but included for interface compatibility.
func (l *Locator) Close() error {
	return nil
}
