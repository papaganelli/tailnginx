package config

const DefaultLogPath = "/var/log/nginx/access.log"

// Config holds runtime configuration for the monitoring app.
type Config struct {
	LogPath string
	FromEnd bool
}
