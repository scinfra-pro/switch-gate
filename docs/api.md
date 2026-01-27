# API Reference

switch-gate provides a RESTful HTTP API for mode switching and monitoring.

## Base URL

Default: `http://127.0.0.1:9090`

## Endpoints

### GET /status

Returns current status and metrics.

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

**Example:**

```bash
curl http://localhost:9090/status
```

---

### POST /mode/{mode}

Switch routing mode.

**Parameters:**

| Name | Type | Description |
|------|------|-------------|
| mode | path | Target mode: `direct`, `warp`, or `home` |

**Response (success):**

```json
{
  "status": "ok",
  "mode": "warp"
}
```

**Response (error):**

```json
{
  "error": "mode warp is not available"
}
```

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

All error responses have the format:

```json
{
  "error": "error message"
}
```

**HTTP Status Codes:**

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad request (invalid mode, invalid JSON, etc.) |
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

# Switch mode based on condition
if [ "$current_mode" = "direct" ]; then
    curl -s -X POST "$API/mode/warp" | jq .
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
