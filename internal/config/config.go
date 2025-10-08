package config

import "time"

const DefaultLogPath = "/var/log/nginx/access.log"
const DefaultRefreshRate = 1 * time.Second
const MinRefreshRate = 100 * time.Millisecond
const MaxRefreshRate = 10 * time.Second

// Config holds runtime configuration for the monitoring app.
type Config struct {
	LogPath     string
	FromEnd     bool
	RefreshRate time.Duration
}
