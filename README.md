# tailnginx

A beautiful Go TUI application that monitors nginx access logs in real-time using tview.

## Features

- 🚀 Real-time nginx access log monitoring
- 🔍 **Auto-Detection** - Automatically finds nginx log files on your system
- ⏱️ **Time Windows** - View last 5/30min, 1/3/12h, 1/7/30 days, or all time (press `t` to toggle)
- 📊 Live statistics: requests, unique visitors, uptime
- 📡 Status code distribution with color-coded bars (2xx=green, 3xx=blue, 4xx=yellow, 5xx=red)
- 🔥 Top visited paths
- 👥 Most active visitors by IP
- 🌐 Browser/client detection (Chrome, Firefox, Safari, curl, bots, etc.)
- 🌍 **IP Geolocation** - See visitor countries with embedded GeoIP database (no external files needed)
- 🔗 **Top Referrers** - Track where your traffic comes from (search engines, social media, etc.)
- ⚡ **Configurable refresh rate** - Adjust update speed from 100ms to 10s
- ⏸️ **Pause/Resume** - Press space to pause/resume monitoring
- 🔍 **Filtering** - Filter by HTTP status codes (press `2`-`5`)
- 📝 Recent request stream with timestamps
- 🎨 Professional TUI built with tview - elegant tables and automatic layouts

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

By default, tailnginx shows data from the **last 5 minutes**. Press `t` to cycle through:
- **5m** - Last 5 minutes (default)
- **30m** - Last 30 minutes
- **1h** - Last 1 hour
- **3h** - Last 3 hours
- **12h** - Last 12 hours
- **1d** - Last 1 day
- **7d** - Last 7 days
- **30d** - Last 30 days
- **All time** - All data since app start

This makes the app actually useful for monitoring current traffic without being overwhelmed by historical data!

## Development

```bash
# Run tests
make test

# Build
make build

# Clean dependencies
make tidy
```

## Log Format

tailnginx expects nginx logs in the **combined** format:

```nginx
log_format combined '$remote_addr - $remote_user [$time_local] '
                    '"$request" $status $body_bytes_sent '
                    '"$http_referer" "$http_user_agent"';
```

Sample logs for testing are provided in `sample_logs/access.log`.

## Architecture

- **cmd/tailnginx** - Main entry point
- **pkg/parser** - Nginx combined log format parser
- **pkg/tailer** - File tailing with reopen support
- **pkg/geoip** - IP geolocation with embedded database (phuslu/iploc)
- **ui** - Bubble Tea TUI implementation
- **internal/config** - Configuration structures

## Screenshots

The dashboard displays:
- **Overview** - Total requests, unique visitors, average bytes per request
- **HTTP Status Codes** - Visual bars showing status code distribution
- **Top Paths** - Most frequently accessed URLs
- **Top Visitors** - Most active IP addresses
- **Clients & Browsers** - User agent breakdown
- **HTTP Methods** - GET, POST, PUT, DELETE, PATCH distribution
- **Countries** - Geographic distribution of visitors (ISO country codes)
- **Top Referrers** - Traffic sources (Google, HackerNews, Twitter, etc.)
- **Recent Activity** - Live stream of incoming requests

## License

MIT
