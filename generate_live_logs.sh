#!/bin/bash
# Generate nginx log entries with current timestamps for testing rate tracking

LOG_FILE="sample_logs/access.log"

# Sample IPs
IPS=("192.168.1.100" "10.0.0.45" "172.16.0.1" "203.0.113.42" "198.51.100.7")

# Sample paths
PATHS=("/api/users" "/products" "/login" "/dashboard" "/api/status" "/search")

# Sample user agents
AGENTS=(
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15"
    "Mozilla/5.0 (X11; Linux x86_64; rv:89.0) Firefox/89.0"
    "curl/7.68.0"
)

# Sample status codes
STATUSES=(200 200 200 201 304 400 404 500)

echo "Generating live log entries to $LOG_FILE..."
echo "Press Ctrl+C to stop"

# Clear the log file
> "$LOG_FILE"

# Generate logs continuously
while true; do
    # Random values
    IP=${IPS[$RANDOM % ${#IPS[@]}]}
    PATH=${PATHS[$RANDOM % ${#PATHS[@]}]}
    STATUS=${STATUSES[$RANDOM % ${#STATUSES[@]}]}
    AGENT=${AGENTS[$RANDOM % ${#AGENTS[@]}]}
    BYTES=$((RANDOM % 10000 + 100))

    # Current timestamp in nginx format
    TIMESTAMP=$(date +"%d/%b/%Y:%H:%M:%S %z")

    # Generate log line
    echo "$IP - - [$TIMESTAMP] \"GET $PATH HTTP/1.1\" $STATUS $BYTES \"-\" \"$AGENT\"" >> "$LOG_FILE"

    # Random delay between 0.1 and 2 seconds
    sleep 0.$((RANDOM % 10 + 1))
done
