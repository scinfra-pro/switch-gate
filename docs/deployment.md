# Deployment

## Requirements

- Linux (recommended for transparent proxy support)
- Go 1.21+ (for building from source)
- Network interfaces configured (for tunnel mode)

## Installation

### From Binary

Download the latest release from GitHub Releases:

```bash
# Download
curl -LO https://github.com/scinfra-pro/switch-gate/releases/latest/download/switch-gate-linux-amd64

# Make executable
chmod +x switch-gate-linux-amd64

# Move to PATH
sudo mv switch-gate-linux-amd64 /usr/local/bin/switch-gate
```

### From Source

```bash
# Clone repository
git clone https://github.com/scinfra-pro/switch-gate.git
cd switch-gate

# Build
make build-linux

# Install
sudo cp bin/switch-gate-linux-amd64 /usr/local/bin/switch-gate
```

## Configuration

Create configuration file:

```bash
sudo mkdir -p /etc/switch-gate
sudo cp configs/switch-gate.example.yaml /etc/switch-gate/config.yaml
sudo chmod 600 /etc/switch-gate/config.yaml
```

Edit configuration:

```bash
sudo nano /etc/switch-gate/config.yaml
```

## Systemd Service

Create service file:

```bash
sudo tee /etc/systemd/system/switch-gate.service << 'EOF'
[Unit]
Description=switch-gate traffic router
Documentation=https://github.com/scinfra-pro/switch-gate
After=network.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/switch-gate -config /etc/switch-gate/config.yaml
Restart=always
RestartSec=5

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/etc/switch-gate

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=switch-gate

[Install]
WantedBy=multi-user.target
EOF
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable switch-gate
sudo systemctl start switch-gate
```

Check status:

```bash
sudo systemctl status switch-gate
sudo journalctl -u switch-gate -f
```

## Docker

### Build Image

```bash
docker build -t switch-gate:latest .
```

### Run Container

```bash
docker run -d \
  --name switch-gate \
  --restart unless-stopped \
  -p 18388:18388 \
  -p 9090:9090 \
  -v /path/to/config.yaml:/etc/switch-gate/config.yaml:ro \
  -e PROXY_PASSWORD="your-password" \
  switch-gate:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  switch-gate:
    image: switch-gate:latest
    container_name: switch-gate
    restart: unless-stopped
    ports:
      - "18388:18388"
      - "9090:9090"
    volumes:
      - ./config.yaml:/etc/switch-gate/config.yaml:ro
    environment:
      - PROXY_PASSWORD=${PROXY_PASSWORD}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9090/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Verification

### Check Service

```bash
# Service status
systemctl status switch-gate

# API health
curl http://127.0.0.1:9090/health

# Current status
curl http://127.0.0.1:9090/status
```

### Test SOCKS5 Proxy

```bash
# Test connection through proxy
curl -x socks5h://127.0.0.1:18388 https://ifconfig.me
```

### Test Mode Switching

```bash
# Switch to each mode and verify
curl -X POST http://127.0.0.1:9090/mode/direct
curl -x socks5h://127.0.0.1:18388 https://ifconfig.me

curl -X POST http://127.0.0.1:9090/mode/warp
curl -x socks5h://127.0.0.1:18388 https://ifconfig.me
```

## Transparent Proxy (Linux)

For transparent proxy with iptables REDIRECT:

1. Enable transparent proxy in config:

```yaml
server:
  listen: "0.0.0.0:18388"
  transparent: "0.0.0.0:18389"
  api: "127.0.0.1:9090"
```

2. Configure iptables:

```bash
# Redirect TCP traffic to transparent proxy
iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 18389
iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-port 18389
```

## Monitoring

### Prometheus

Add to Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: 'switch-gate'
    static_configs:
      - targets: ['switch-gate-host:9090']
    metrics_path: '/metrics'
```

### Available Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `switch_gate_bytes_total{mode}` | counter | Total bytes per mode |
| `switch_gate_connections_active` | gauge | Active connections |
| `switch_gate_connections_total` | counter | Total connections |
| `switch_gate_uptime_seconds` | gauge | Uptime in seconds |

## Troubleshooting

### Service Won't Start

```bash
# Check logs
journalctl -u switch-gate -n 50

# Verify config syntax
switch-gate -config /etc/switch-gate/config.yaml
```

### Mode Not Available

```bash
# Check available modes
curl http://127.0.0.1:9090/status | jq .available_modes

# For warp mode, verify interface exists
ip link show warp0
```

### Connection Errors

```bash
# Check if ports are listening
ss -tlnp | grep -E '18388|9090'

# Test local connectivity
curl -x socks5h://127.0.0.1:18388 https://example.com
```
