# Webhooks

switch-gate can send HTTP webhook notifications when events occur, such as mode changes or traffic limit exhaustion.

## Configuration

Enable webhooks in the configuration file:

```yaml
webhooks:
  # Enable webhook notifications
  enabled: true
  
  # Webhook receiver URL
  url: "http://${WEBHOOK_HOST}/webhook/switch-gate"
  
  # Shared secret for authentication
  secret: "${WEBHOOK_SECRET}"
  
  # VPS identifier included in event payload
  source: "my-vps"
  
  # Event filtering
  events:
    mode_changed: false    # Disable if using Telegram inline buttons
    limit_reached: true    # Important automatic event
```

## Event Filtering

Filter which events to send at the source to avoid unnecessary traffic:

```yaml
webhooks:
  events:
    mode_changed: false    # Don't send (user sees inline button state)
    limit_reached: true    # Send (automatic event, user may not know)
```

### Recommended Settings

| Event | Recommended | Reason |
|-------|-------------|--------|
| `mode_changed` | `false` | User switches via Telegram buttons and sees the result immediately |
| `limit_reached` | `true` | Automatic event; user should be notified about the switch |

### When to Enable mode_changed

- Multiple users managing the same VPS
- Automated mode switches via cron jobs
- External API calls (not via Telegram bot)
- Debugging and monitoring

---

## Events

### mode.changed

Sent when the routing mode changes, either manually via API or automatically due to limit exhaustion.

**Payload:**

```json
{
  "event": "mode.changed",
  "timestamp": "2026-01-28T15:30:00Z",
  "source": "my-vps",
  "payload": {
    "from": "direct",
    "to": "warp",
    "trigger": "manual"
  }
}
```

**Trigger values:**

| Value | Description |
|-------|-------------|
| `manual` | Mode changed via API (`POST /mode/{mode}`) |
| `limit_reached` | Mode changed automatically due to traffic limit |

---

### limit.reached

Sent when the home mode traffic limit is exhausted.

**Payload:**

```json
{
  "event": "limit.reached",
  "timestamp": "2026-01-28T16:00:00Z",
  "source": "my-vps",
  "payload": {
    "mode": "home",
    "used_mb": 100,
    "limit_mb": 100,
    "switched_to": "warp"
  }
}
```

**Note:** When `limit.reached` is sent, a `mode.changed` event is also sent with `trigger: "limit_reached"` (if `events.mode_changed` is enabled).

---

## Request Format

All webhook requests are HTTP POST with the following headers:

| Header | Value |
|--------|-------|
| `Content-Type` | `application/json` |
| `X-Webhook-Secret` | Configured secret value |

## Retry Behavior

Webhooks are sent asynchronously with automatic retry:

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 1 second |
| 3 | 3 seconds |

- Request timeout: 5 seconds per attempt
- If all attempts fail, an error is logged
- Failed webhooks do not affect switch-gate operation

## Security

### Authentication

Receivers should validate the `X-Webhook-Secret` header:

```go
if r.Header.Get("X-Webhook-Secret") != expectedSecret {
    http.Error(w, "Unauthorized", 401)
    return
}
```

### Network Isolation

For security, send webhooks over a private network (e.g., WireGuard VPN) rather than the public internet:

```bash
# Set WEBHOOK_HOST to a private IP
export WEBHOOK_HOST="10.0.5.10:8080"
```

## Example Receiver

Simple Go receiver that logs events:

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
)

type Event struct {
    Name      string                 `json:"event"`
    Timestamp string                 `json:"timestamp"`
    Source    string                 `json:"source"`
    Payload   map[string]interface{} `json:"payload"`
}

func main() {
    secret := "your-webhook-secret"
    
    http.HandleFunc("/webhook/switch-gate", func(w http.ResponseWriter, r *http.Request) {
        // Validate secret
        if r.Header.Get("X-Webhook-Secret") != secret {
            http.Error(w, "Unauthorized", 401)
            return
        }
        
        // Parse event
        var event Event
        if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
            http.Error(w, "Bad Request", 400)
            return
        }
        
        log.Printf("Event: %s from %s - %v", event.Name, event.Source, event.Payload)
        w.WriteHeader(http.StatusOK)
    })
    
    log.Println("Listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Disabling Webhooks

To disable webhooks, set `enabled: false` or remove the webhooks section:

```yaml
webhooks:
  enabled: false
```

When disabled, no webhook-related code is executed.
