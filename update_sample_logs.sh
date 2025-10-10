#!/bin/bash
# Update sample logs with current timestamps for rate metrics testing
# Run this script before testing if sample logs are more than 10 minutes old

cd "$(dirname "$0")"

if [ ! -f "sample_logs/access.log.backup" ]; then
    echo "Error: sample_logs/access.log.backup not found"
    exit 1
fi

echo "Updating sample logs with current timestamps..."

python3 << 'PYTHON'
import re
from datetime import datetime, timedelta

# Read the backup file
with open('sample_logs/access.log.backup', 'r') as f:
    lines = f.readlines()

# Get current time and spread entries over last 5 minutes
now = datetime.now()
time_delta = timedelta(minutes=5)
entry_count = len(lines)

with open('sample_logs/access.log', 'w') as f:
    for i, line in enumerate(lines):
        # Calculate timestamp for this entry (spread over last 5 minutes)
        entry_time = now - time_delta + (time_delta * i / entry_count)

        # Format timestamp in nginx format
        new_timestamp = entry_time.strftime("%d/%b/%Y:%H:%M:%S +0200")

        # Replace the timestamp in the log line
        new_line = re.sub(r'\[.*?\]', f'[{new_timestamp}]', line)
        f.write(new_line)

print(f"✓ Updated {entry_count} log entries")
print(f"  Time range: {(now - time_delta).strftime('%H:%M:%S')} to {now.strftime('%H:%M:%S')}")
print(f"\nNow run: ./bin/tailnginx -log ./sample_logs/access.log")
PYTHON
