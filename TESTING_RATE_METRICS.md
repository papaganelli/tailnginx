# Testing Rate Metrics Feature

The request rate tracking feature displays real-time statistics in the overview panel.

## Understanding Rate Tracking

The RateTracker uses a **10-minute rolling window** with 10-second buckets. It only tracks requests whose timestamps fall within the last 10 minutes from the current time.

### Why you might not see rate metrics:

1. **Old log files** - If your log file contains entries older than 10 minutes, they won't be tracked
2. **No recent activity** - The rate tracker needs requests within the last 10 minutes
3. **Sample logs** - By default, sample logs may have old timestamps

## Testing with Sample Logs

The sample logs have been updated to include timestamps from the last 5 minutes. To test:

```bash
# Run with sample logs
./bin/tailnginx -log ./sample_logs/access.log
```

You should see in the overview panel:
- **Rate: X.X req/s** followed by a trend indicator
- **↑** (green) = traffic increasing >5%
- **↓** (red) = traffic decreasing >5%
- **→** (yellow) = traffic stable

## Generating Live Logs

For continuous testing, use the log generator script:

```bash
# Terminal 1: Generate live logs
./generate_live_logs.sh

# Terminal 2: Monitor the logs
./bin/tailnginx -log ./sample_logs/access.log
```

The generator creates realistic log entries with current timestamps at random intervals.

## Real-World Usage

For production nginx logs, the rate tracking works automatically:

```bash
# Monitor live nginx access log
./bin/tailnginx -log /var/log/nginx/access.log

# Or use auto-detection
./bin/tailnginx
```

The rate metrics will update in real-time as new requests come in!

## Restoring Original Sample Logs

If you want to restore the original sample logs:

```bash
cp sample_logs/access.log.backup sample_logs/access.log
```
