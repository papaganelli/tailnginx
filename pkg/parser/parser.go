// Package parser provides functionality to parse nginx access log entries.
package parser

import (
	"regexp"
	"strconv"
	"time"
)

// Visitor represents a parsed nginx access log entry.
type Visitor struct {
	Time     time.Time
	IP       string
	Method   string
	Path     string
	Protocol string
	Referer  string
	Agent    string
	Country  string // Added by geoip lookup, not from log
	Status   int
	Bytes    int
}

// combinedRegex matches the nginx combined log format
var combinedRegex = regexp.MustCompile(`(?P<ip>[^ ]+) [^ ]+ [^ ]+ \[(?P<time>[^\]]+)\] "(?P<method>\S+) (?P<path>[^ ]+) (?P<proto>[^\"]+)" (?P<status>\d{3}) (?P<bytes>\d+|-) "(?P<referer>[^"]*)" "(?P<agent>[^"]+)"`)

// Parse parses a nginx combined log format line into a Visitor struct.
// It expects the standard nginx combined log format:
// <IP> - - [<time>] "<method> <path> <proto>" <status> <bytes> "<referer>" "<agent>"
// Returns nil if the line doesn't match the expected format or parsing fails.
func Parse(line string) *Visitor {
	m := combinedRegex.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	result := &Visitor{}
	for i, name := range combinedRegex.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		val := m[i]
		switch name {
		case "ip":
			result.IP = val
		case "time":
			// example: 08/Oct/2025:12:00:00 +0000
			if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", val); err == nil {
				result.Time = t
			}
		case "method":
			result.Method = val
		case "path":
			result.Path = val
		case "proto":
			result.Protocol = val
		case "status":
			if v, err := strconv.Atoi(val); err == nil {
				result.Status = v
			}
		case "bytes":
			if val == "-" {
				result.Bytes = 0
			} else if v, err := strconv.Atoi(val); err == nil {
				result.Bytes = v
			}
		case "referer":
			result.Referer = val
		case "agent":
			result.Agent = val
		}
	}
	return result
}
