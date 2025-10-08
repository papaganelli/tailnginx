// Package geoip provides IP geolocation functionality using an embedded database.
package geoip

import (
	"net"

	"github.com/phuslu/iploc"
)

// Locator provides IP geolocation lookups using an embedded database
type Locator struct {
	// iploc is stateless and doesn't need instance data
}

// Location represents a geographic location
type Location struct {
	Country     string
	CountryCode string
	City        string
}

// NewLocator creates a new Locator with embedded database
func NewLocator() (*Locator, error) {
	// iploc has embedded data, no initialization needed
	return &Locator{}, nil
}

// Lookup looks up the location of an IP address
func (l *Locator) Lookup(ipStr string) (*Location, error) {
	if l == nil {
		return &Location{
			Country:     "Unknown",
			CountryCode: "??",
			City:        "Unknown",
		}, nil
	}

	// Parse IP address
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return &Location{
			Country:     "Unknown",
			CountryCode: "??",
			City:        "Unknown",
		}, nil
	}

	// Use iploc to get country code
	country := iploc.Country(ip)

	if country == "" {
		return &Location{
			Country:     "Unknown",
			CountryCode: "??",
			City:        "Unknown",
		}, nil
	}

	return &Location{
		Country:     country,
		CountryCode: country,
		City:        "Unknown", // iploc only provides country
	}, nil
}

// Close closes the database (no-op for iploc)
func (l *Locator) Close() error {
	return nil
}
