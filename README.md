# tailnginx

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Test Coverage](https://img.shields.io/badge/coverage-70.8%25-brightgreen)](https://github.com/papaganelli/tailnginx)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/papaganelli/tailnginx)](https://github.com/papaganelli/tailnginx/releases)

A beautiful, secure Go TUI application that monitors nginx access logs in real-time using tview.

## Features

### 🚀 Monitoring
- **Real-time log tailing** - Instant updates as requests hit your server
- **Auto-Detection** - Automatically finds nginx log files on your system
- **Time Windows** - View last 5/30min, 1/3/12h, 1/7/30 days, or all time (press `t` to toggle)
- **Live statistics** - Requests, unique visitors, uptime tracking
- **Recent activity stream** - Live feed of incoming requests

### 📊 Analytics
- **Request rate tracking** - Real-time requests/second with trend indicators (↑/↓/→)
- **Status code distribution** - Color-coded bars (2xx=green, 3xx=blue, 4xx=yellow, 5xx=red)
- **Top paths** - Most frequently accessed URLs
- **Top visitors** - Most active IP addresses
- **Browser/client detection** - Chrome, Firefox, Safari, curl, bots, etc.
- **HTTP methods breakdown** - GET, POST, PUT, DELETE, PATCH distribution
- **Geographic insights** - Visitor countries with embedded GeoIP database (no external files needed)
- **Traffic sources** - Top referrers (Google, social media, etc.)

### 🔒 Security & Performance
- **Path validation** - Prevents reading sensitive system files
- **Buffer limits** - Protection against memory exhaustion
- **GeoIP caching** - 10-50x speedup for repeated IP lookups
- **High test coverage** - 70.8% code coverage with comprehensive tests

### ⚡ Controls
- **Configurable refresh rate** - Adjust update speed from 100ms to 10s
- **Pause/Resume** - Press space to pause/resume monitoring
- **Status filtering** - Filter by HTTP status codes (press `2`-`5`)
- **Responsive UI** - Professional TUI built with tview

## Requirements

- Go 1.24+
- nginx access logs in combined format

## Installation

```bash
git clone https://github.com/papaganelli/tailnginx.git
cd tailnginx
make tidy
make build
```

## Usage

```bash
# Auto-detect nginx logs (scans common paths and nginx config)
./tailnginx

# Monitor specific log file
./tailnginx -log /path/to/access.log

# Test with sample data
./tailnginx -log ./sample_logs/access.log
```

### Auto-Detection

When run without `-log` flag, tailnginx automatically:
1. Scans common nginx log locations (`/var/log/nginx/`, `/usr/local/nginx/logs/`, etc.)
2. Parses nginx config files to find custom `access_log` directives
3. Lists all detected log files with their server names
4. Selects the best log file (prioritizes `access.log`, then largest file)

Example output:
```
Found 3 nginx log files:
  → 1. /var/log/nginx/access.log [1.2 MB]
    2. /var/log/nginx/example.com-access.log (example.com) [850 KB]
    3. /var/log/nginx/api.log (api.example.com) [450 KB]

Monitoring: /var/log/nginx/access.log
Use -log flag to specify a different file
```

### Options

- `-log` - Path to nginx access log (auto-detect if not specified)
- `-refresh` - Refresh rate in milliseconds, 100-10000 (default: `1000`)

### Controls

- `q` or `Ctrl+C` - Quit
- `Space` - Pause/Resume monitoring
- `t` - **Toggle time window** (5m → 30m → 1h → 3h → 12h → 1d → 7d → 30d → All time)
- `+` - Increase refresh rate (faster updates)
- `-` - Decrease refresh rate (slower updates)
- `2` - Filter 2xx status codes
- `3` - Filter 3xx status codes
- `4` - Filter 4xx status codes
- `5` - Filter 5xx status codes
- `Esc` - Clear status filter

### Time Windows

By default, tailnginx shows **all time** data. Press `t` to cycle through different time windows:
- **5m** - Last 5 minutes
- **30m** - Last 30 minutes
- **1h** - Last 1 hour
- **3h** - Last 3 hours
- **12h** - Last 12 hours
- **1d** - Last 1 day
- **7d** - Last 7 days
- **30d** - Last 30 days
- **All time** - All data since app start (default)

This makes it easy to focus on recent traffic or analyze historical patterns!

### Request Rate Tracking

The request rate feature displays real-time requests/second with trend indicators in the overview panel.

**How it works:**
- Uses a **10-minute rolling window** with 10-second buckets
- Only tracks requests within the last 10 minutes from current time
- Shows trend indicators: **↑** (green, >5% increase), **↓** (red, >5% decrease), **→** (yellow, stable)

**Testing with sample logs:**

The sample logs have been updated with recent timestamps for testing:

```bash
# Run with sample logs (shows rate metrics)
./bin/tailnginx -log ./sample_logs/access.log
```

**Generate live logs for testing:**

```bash
# Terminal 1: Generate continuous logs
./generate_live_logs.sh

# Terminal 2: Monitor with rate tracking
./bin/tailnginx -log ./sample_logs/access.log
```

**Why rate metrics might not appear:**
- Log file contains entries older than 10 minutes
- No recent activity in the last 10 minutes
- Timestamps are in the future

**Restore original sample logs:**
```bash
cp sample_logs/access.log.backup sample_logs/access.log
```

## Development

```bash
# Run tests
make test

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...

# Build
make build

# Clean dependencies
make tidy

# Run linter
golangci-lint run
```

### Test Coverage

tailnginx has comprehensive test coverage:

| Package | Coverage |
|---------|----------|
| pkg/parser | 91.7% |
| pkg/geoip | 88.9% |
| pkg/tailer | 84.6% |
| pkg/metrics | 100.0% |
| pkg/detector | 60.5% |
| **Overall** | **70.8%** |

## Log Format

tailnginx expects nginx logs in the **combined** format:

```nginx
log_format combined '$remote_addr - $remote_user [$time_local] '
                    '"$request" $status $body_bytes_sent '
                    '"$http_referer" "$http_user_agent"';
```

Sample logs for testing are provided in `sample_logs/access.log`.

## Architecture

- **cmd/tailnginx** - Main entry point with path validation and auto-detection
- **pkg/parser** - Nginx combined log format parser with comprehensive tests
- **pkg/tailer** - File tailing with reopen support and buffer limits
- **pkg/detector** - Auto-detection of nginx log files from config
- **pkg/geoip** - IP geolocation with embedded database and caching (phuslu/iploc)
- **pkg/metrics** - Request rate tracking with circular buffer and trend analysis
- **ui** - tview TUI implementation with responsive layouts
- **internal/config** - Configuration structures

### Security Features

- **Path validation** - Blocks reading sensitive files like `/etc/shadow`, `/etc/passwd`
- **Symlink resolution** - Detects and prevents path traversal attacks
- **Buffer limits** - Scanner limited to 1MB per line to prevent memory exhaustion
- **Input validation** - All user inputs are validated before use

## Screenshots

The dashboard displays:
- **Overview** - Total requests, request rate (req/s) with trend indicators, uptime, filters
- **HTTP Status Codes** - Visual bars showing status code distribution
- **Top Paths** - Most frequently accessed URLs
- **Top Visitors** - Most active IP addresses
- **Clients & Browsers** - User agent breakdown
- **HTTP Methods** - GET, POST, PUT, DELETE, PATCH distribution
- **Countries** - Geographic distribution of visitors (ISO country codes)
- **Top Referrers** - Traffic sources (Google, HackerNews, Twitter, etc.)
- **Recent Activity** - Live stream of incoming requests

## Performance

- **Request rate tracking** - Circular buffer with 10-second buckets over 10-minute window
- **GeoIP caching** - 10-50x speedup for repeated IP lookups using sync.Map
- **Batch processing** - Processes up to 100 log entries per batch
- **Buffered channels** - 1000-entry buffer for high-throughput logs
- **Efficient regex** - Pre-compiled patterns for fast parsing

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure `go test ./...` passes
5. Run `golangci-lint run` to check code quality
6. Submit a pull request

## Changelog

### v1.3.0 (2025-10-10)
- **Feature**: Added real-time request rate tracking (requests/second)
- **Feature**: Added trend indicators for rate changes (↑ increasing, ↓ decreasing, → stable)
- **Performance**: Implemented circular buffer for efficient rate calculation (10-minute window)
- **Architecture**: Migrated from archived hpcloud/tail to actively maintained nxadm/tail
- **Tests**: Added comprehensive metrics package tests (100% coverage)

### v1.2.0 (2025-10-10)
- **Security**: Added path validation to prevent reading sensitive files
- **Security**: Added buffer limits to prevent memory exhaustion
- **Tests**: Increased coverage from 13.1% to 70.8% (+441%)
- **Performance**: Implemented GeoIP caching (10-50x speedup)
- **Quality**: Added comprehensive godoc comments
- **CI/CD**: Added golangci-lint with 16 linters
- **CI/CD**: Added race detector and coverage tracking
- **Bug Fix**: Fixed tailer not reading from beginning
- **Bug Fix**: Fixed default time window filtering

### v1.1.0 (2025-10-09)
- Migrated from Bubble Tea to tview framework
- Added auto-detection of nginx log files
- Added GeoIP support with embedded database
- Added time window filtering
- Added status code filtering
- Added configurable refresh rate

### v1.0.0 (2025-10-08)
- Initial release with basic monitoring features

## License

MIT
