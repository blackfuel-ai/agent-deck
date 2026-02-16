# Simple Daily Log File Plan for Slack/Telegram Message History

## Overview
Store all Slack/Telegram messages in simple daily log files using JSON Lines format.

## Configuration

Add to `~/.agent-deck/config.toml` under `[conductor]`:

```toml
[conductor]
enabled = true
message_history_enabled = true  # Enable/disable message history logging (default: true)
heartbeat_interval = 15

[conductor.telegram]
token = "your-bot-token"
user_id = 12345678
```

Set `message_history_enabled = false` to disable message logging completely.

## Benefits
- **Simple**: No database, just append to files
- **Debugging**: Easy to `tail -f` or `grep` for troubleshooting
- **Context**: Conductors can read recent history
- **Audit trail**: Permanent records by day
- **Analytics**: Can process with standard Unix tools (jq, awk, grep)

## File Structure

```
~/.agent-deck/conductor/logs/
├── 2026-02-16.log
├── 2026-02-17.log
└── 2026-02-18.log
```

## Log Format (JSON Lines)

Each line is a complete JSON object:

```json
{"timestamp":"2026-02-16T14:30:45.123Z","platform":"telegram","direction":"incoming","sender":"12345678","recipient":"conductor-work","profile":"work","conductor":"conductor-work","message":"check the frontend session","response":"Frontend session is waiting. Last error: npm test failed...","response_time_ms":2340,"status":"completed","message_id":"999","metadata":{"username":"user123","first_name":"John"}}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | ISO 8601 string | When message was received |
| `platform` | string | "telegram" or "slack" |
| `direction` | string | "incoming" or "outgoing" |
| `sender` | string | User ID or conductor name |
| `recipient` | string | Conductor session name or user |
| `profile` | string | Agent-deck profile |
| `conductor` | string | Conductor handling message |
| `message` | string | User's message text |
| `response` | string | Conductor's response (null for incomplete) |
| `response_time_ms` | int | Response time in ms (null for incomplete) |
| `status` | string | "pending", "completed", "error", "timeout" |
| `message_id` | string | Platform-specific message ID (optional) |
| `thread_id` | string | Platform-specific thread ID (optional) |
| `metadata` | object | Additional platform-specific data (optional) |

## Implementation

### 1. Helper Functions in bridge.py

```python
from datetime import datetime, timezone
import json
from pathlib import Path

LOGS_DIR = CONDUCTOR_DIR / "logs"

def get_daily_log_path() -> Path:
    """Get path for today's log file."""
    today = datetime.now(timezone.utc).strftime("%Y-%m-%d")
    LOGS_DIR.mkdir(exist_ok=True)
    return LOGS_DIR / f"{today}.log"

def log_message(platform, direction, sender, message, **kwargs):
    """Append a message to today's log file."""
    entry = {
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "platform": platform,
        "direction": direction,
        "sender": sender,
        "message": message,
        "status": "pending" if direction == "incoming" else "sent",
        **kwargs
    }

    with open(get_daily_log_path(), "a", encoding="utf-8") as f:
        f.write(json.dumps(entry, ensure_ascii=False) + "\n")

    return entry["timestamp"]  # Use as correlation ID

def update_message_response(msg_timestamp, response, response_time_ms, status="completed"):
    """Add a completion entry for a message."""
    entry = {
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "message_timestamp": msg_timestamp,
        "response": response,
        "response_time_ms": response_time_ms,
        "status": status,
    }

    with open(get_daily_log_path(), "a", encoding="utf-8") as f:
        f.write(json.dumps(entry, ensure_ascii=False) + "\n")
```

### 2. Integration Points in bridge.py

**In `handle_message()` function:**

```python
# Before sending to conductor
msg_ts = log_message(
    platform="telegram",
    direction="incoming",
    sender=str(message.from_user.id),
    recipient=session_title,
    profile=target_profile,
    conductor=session_title,
    message=cleaned_msg,
    message_id=str(message.message_id),
    metadata={
        "username": message.from_user.username,
        "first_name": message.from_user.first_name,
    }
)

# After getting response
start_time = time.time()
response = await wait_for_response(...)
response_time_ms = int((time.time() - start_time) * 1000)

update_message_response(
    msg_ts,
    response=response,
    response_time_ms=response_time_ms,
    status="completed"  # or "error", "timeout"
)
```

**In `heartbeat_loop()` function:**

```python
# Log heartbeat messages
log_message(
    platform="heartbeat",
    direction="outgoing",
    sender="bridge",
    recipient=session_title,
    profile=profile,
    conductor=session_title,
    message=heartbeat_msg
)
```

### 3. Slack Integration (Same Pattern)

Add similar logging in Slack message handlers when implemented.

## Query Tools

### Simple Shell Scripts

**View today's messages:**
```bash
#!/bin/bash
# logs/view-today.sh
cat ~/.agent-deck/conductor/logs/$(date -u +%Y-%m-%d).log | jq .
```

**Search by keyword:**
```bash
#!/bin/bash
# logs/search.sh KEYWORD
grep -i "$1" ~/.agent-deck/conductor/logs/*.log | jq -r '[.timestamp, .platform, .message] | @tsv'
```

**Analytics - response times:**
```bash
#!/bin/bash
# logs/avg-response-time.sh
cat ~/.agent-deck/conductor/logs/*.log | \
  jq -s 'map(select(.response_time_ms)) |
         {avg: (map(.response_time_ms) | add / length),
          min: (map(.response_time_ms) | min),
          max: (map(.response_time_ms) | max)}'
```

**Count by status:**
```bash
#!/bin/bash
# logs/status-breakdown.sh
cat ~/.agent-deck/conductor/logs/*.log | \
  jq -s 'group_by(.status) | map({status: .[0].status, count: length})'
```

### Python CLI Tool (Optional)

```bash
# Query recent messages
python3 conductor/query_logs.py --recent 50

# Search
python3 conductor/query_logs.py --search "error"

# Analytics
python3 conductor/query_logs.py --analytics --days 7

# Export to JSON
python3 conductor/query_logs.py --export output.json --days 30
```

## Conductor Access to History

**Add to conductor CLAUDE.md:**

```markdown
## Message History

You can access recent message history:

```bash
# View today's messages
tail -50 ~/.agent-deck/conductor/logs/$(date -u +%Y-%m-%d).log | jq .

# Search for specific messages
grep -i "keyword" ~/.agent-deck/conductor/logs/*.log | jq .

# Get recent messages (last 20 lines)
tail -20 ~/.agent-deck/conductor/logs/$(date -u +%Y-%m-%d).log | jq -r '.message'
```

Use this to understand context before responding.
```

## Log Rotation & Cleanup

**Add to conductor setup:**

```bash
# Optional: Keep only last 90 days
find ~/.agent-deck/conductor/logs -name "*.log" -mtime +90 -delete
```

**Or in Python (bridge.py startup):**

```python
def cleanup_old_logs(days=90):
    """Delete log files older than specified days."""
    cutoff = datetime.now(timezone.utc) - timedelta(days=days)

    for log_file in LOGS_DIR.glob("*.log"):
        try:
            # Parse date from filename YYYY-MM-DD.log
            date_str = log_file.stem
            file_date = datetime.strptime(date_str, "%Y-%m-%d").replace(tzinfo=timezone.utc)
            if file_date < cutoff:
                log_file.unlink()
                log.info(f"Deleted old log: {log_file}")
        except (ValueError, OSError) as e:
            log.warning(f"Failed to process {log_file}: {e}")
```

## Advantages of This Approach

1. **No dependencies**: No SQLite, just standard library
2. **Human readable**: Can open in any text editor
3. **Standard tools**: grep, jq, awk all work
4. **Automatic rotation**: New file each day
5. **No migration needed**: JSON format is self-describing
6. **Easy backup**: Just copy the logs directory
7. **Easy to debug**: `tail -f` to watch live
8. **Easy to extend**: Add fields without schema changes

## Future Enhancements (Optional)

1. **Compression**: gzip old logs automatically
2. **Structured querying**: Simple Python script for common queries
3. **Web UI**: Simple Flask app to browse logs
4. **Metrics**: Export to Prometheus/Grafana
5. **Alerts**: Monitor for error patterns

## Implementation Checklist

- [ ] Create `logs/` directory structure
- [ ] Add helper functions to bridge.py
- [ ] Integrate logging in `handle_message()`
- [ ] Integrate logging in `heartbeat_loop()`
- [ ] Add Slack message logging (when implemented)
- [ ] Create shell scripts for common queries
- [ ] Update conductor CLAUDE.md with history access info
- [ ] Add log cleanup on bridge startup
- [ ] Test with actual Telegram messages
- [ ] Document in README
