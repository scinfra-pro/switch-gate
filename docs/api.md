# API Reference

switch-gate provides a RESTful HTTP API for mode switching and monitoring.

## Base URL

Default: `http://127.0.0.1:9090`

## Endpoints

### GET /status

Returns current status and metrics.

**Query parameters:**

| Name | Type | Description |
|------|------|-------------|
| check | bool | Optional. If `true`, performs health check of current mode |

**Response:**

```json
{
  "mode": "direct",
  "uptime": "2h34m56s",
  "connections": 12,
  "traffic": {
    "direct_mb": 150.5,
    "warp_mb": 2340.2,
    "home_mb": 45.3,
    "total_mb": 2536.0
  },
  "home": {
    "limit_mb": 100,
    "used_mb": 45.3,
    "remaining_mb": 54.7,
    "cost_usd": 0.16
  },
  "available_modes": ["direct", "warp", "home"]
}
```

**Response with `?check=true` (mode healthy):**

```json
{
  "mode": "warp",
  "mode_healthy": true,
  "uptime": "2h34m56s",
  ...
}
```

**Response with `?check=true` (mode unhealthy):**

```json
{
  "mode": "warp",
  "mode_healthy": false,
  "mode_error": "warp_unreachable",
  "uptime": "2h34m56s",
  ...
}
```

**Health check fields (only with `?check=true`):**

| Field | Type | Description |
|-------|------|-------------|
| `mode_healthy` | bool | Whether current mode is working |
| `mode_error` | string | Error code (only if `mode_healthy` is false) |

**Mode error codes:**

| Code | Description |
|------|-------------|
| `warp_unreachable` | WARP tunnel not responding |
| `warp_timeout` | Connection timeout through WARP |
| `home_unreachable` | Home proxy not responding |
| `home_timeout` | Connection timeout through Home |

**Examples:**

```bash
# Quick status (no health check)
curl http://localhost:9090/status

# Status with health check (~5 sec longer)
curl "http://localhost:9090/status?check=true"
```

---

### POST /mode/{mode}

Switch routing mode.

**Parameters:**

| Name | Type | Description |
|------|------|-------------|
| mode | path | Target mode: `direct`, `warp`, or `home` |

**Response fields:**

| Field | Type | Description |
|-------|------|-------------|
| `success` | bool | Whether the requested mode was activated |
| `requested` | string | The mode that was requested |
| `mode` | string | The current active mode |
| `error` | string | Error code (only if `success` is false) |
| `status` | string | `"ok"` for backward compatibility (only if success) |

**Response (success):**

```json
{
  "success": true,
  "requested": "warp",
  "mode": "warp",
  "status": "ok"
}
```

**Response (failure — mode not available):**

```json
{
  "success": false,
  "requested": "warp",
  "mode": "direct",
  "error": "mode_not_configured"
}
```

**Response (failure — home limit reached):**

```json
{
  "success": false,
  "requested": "home",
  "mode": "warp",
  "error": "home_limit_reached"
}
```

**Error codes:**

| Code | Description |
|------|-------------|
| `mode_invalid` | Unknown mode (not direct/warp/home) |
| `mode_not_configured` | Mode is valid but not configured |
| `home_limit_reached` | Home proxy traffic limit exhausted |
| `internal_error` | Unexpected internal error |

**Examples:**

```bash
# Switch to direct mode
curl -X POST http://localhost:9090/mode/direct

# Switch to tunnel mode
curl -X POST http://localhost:9090/mode/warp

# Switch to upstream proxy mode
curl -X POST http://localhost:9090/mode/home
```

---

### GET /metrics

Returns Prometheus-compatible metrics.

**Response:**

```
# HELP switch_gate_bytes_total Total bytes transferred
# TYPE switch_gate_bytes_total counter
switch_gate_bytes_total{mode="direct"} 157810688
switch_gate_bytes_total{mode="warp"} 2453299200
switch_gate_bytes_total{mode="home"} 47500288

# HELP switch_gate_connections_active Active connections
# TYPE switch_gate_connections_active gauge
switch_gate_connections_active 12

# HELP switch_gate_connections_total Total connections
# TYPE switch_gate_connections_total counter
switch_gate_connections_total 1847

# HELP switch_gate_uptime_seconds Uptime in seconds
# TYPE switch_gate_uptime_seconds gauge
switch_gate_uptime_seconds 9296
```

**Example:**

```bash
curl http://localhost:9090/metrics
```

---

### GET /health

Health check endpoint.

**Response:**

```json
{
  "status": "healthy"
}
```

**Example:**

```bash
curl http://localhost:9090/health
```

---

### POST /limit/home

Set traffic limit for home mode.

**Request body:**

```json
{
  "limit_mb": 200
}
```

**Response:**

```json
{
  "status": "ok",
  "limit_mb": 200
}
```

**Example:**

```bash
curl -X POST http://localhost:9090/limit/home \
  -H "Content-Type: application/json" \
  -d '{"limit_mb": 200}'
```

---

## Error Responses

Most endpoints return errors in this format:

```json
{
  "error": "error message"
}
```

**Exception:** `POST /mode/{mode}` always returns HTTP 200 with structured response.
Check `success` field to determine if mode switch succeeded.

**HTTP Status Codes:**

| Code | Description |
|------|-------------|
| 200 | Success (or mode switch with fallback) |
| 400 | Bad request (invalid JSON, etc.) |
| 404 | Endpoint not found |
| 500 | Internal server error |

---

## Integration Examples

### Shell Script

```bash
#!/bin/bash

API="http://localhost:9090"

# Get current mode
current_mode=$(curl -s "$API/status" | jq -r .mode)
echo "Current mode: $current_mode"

# Switch mode and check result
response=$(curl -s -X POST "$API/mode/warp")
success=$(echo "$response" | jq -r .success)
mode=$(echo "$response" | jq -r .mode)

if [ "$success" = "true" ]; then
    echo "Switched to: $mode"
else
    error=$(echo "$response" | jq -r .error)
    echo "Failed to switch, staying on: $mode (error: $error)"
fi
```

### Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: 'switch-gate'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/metrics'
```

### Health Check (Docker/Kubernetes)

```yaml
# Docker Compose
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:9090/health"]
  interval: 30s
  timeout: 10s
  retries: 3
```

```yaml
# Kubernetes
livenessProbe:
  httpGet:
    path: /health
    port: 9090
  initialDelaySeconds: 5
  periodSeconds: 10
```
