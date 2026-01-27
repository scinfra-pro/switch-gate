# switch-gate

Outbound traffic router with dynamic mode switching via API.

## Features

- **Three routing modes:**
  - `direct` — Route through server's default interface
  - `warp` — Route through tunnel interface (e.g., WireGuard, WARP)
  - `home` — Route through upstream SOCKS5 proxy

- **HTTP API** for runtime mode switching
- **Prometheus metrics** for monitoring
- **Traffic limits** with automatic mode switching
- **SOCKS5 proxy** interface for clients
- **Transparent proxy** support (Linux, iptables REDIRECT)

## Quick Start

### Build

```bash
# Build for current OS
make build

# Build for Linux
make build-linux

# Build for all platforms
make build-all
```

### Run

```bash
# Create configuration
cp configs/switch-gate.example.yaml configs/switch-gate.yaml
# Edit configs/switch-gate.yaml with your settings

# Run
make run
```

### Docker

```bash
# Build image
make docker-build

# Run container
docker run -p 18388:18388 -p 9090:9090 \
  -v /path/to/config.yaml:/etc/switch-gate/config.yaml \
  switch-gate:latest
```

## Configuration

See [docs/configuration.md](docs/configuration.md) for full configuration reference.

Basic example:

```yaml
server:
  listen: "0.0.0.0:18388"   # SOCKS5 proxy
  api: "127.0.0.1:9090"     # HTTP API

modes:
  direct:
    local_ip: ""            # Optional: bind to specific IP
  warp:
    interface: "warp0"      # Tunnel interface name
  home:
    type: "socks5"
    host: "proxy.example.com"
    port: 7000
    username: "user"
    password: "${PROXY_PASSWORD}"

limits:
  home:
    max_mb: 100
    auto_switch_to: "warp"
```

## API

See [docs/api.md](docs/api.md) for full API reference.

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/status` | Current status and metrics |
| POST | `/mode/{mode}` | Switch routing mode |
| GET | `/metrics` | Prometheus metrics |
| GET | `/health` | Health check |
| POST | `/limit/home` | Set home mode traffic limit |

### Examples

```bash
# Get status
curl http://localhost:9090/status

# Switch to direct mode
curl -X POST http://localhost:9090/mode/direct

# Switch to tunnel mode
curl -X POST http://localhost:9090/mode/warp

# Switch to upstream proxy mode
curl -X POST http://localhost:9090/mode/home
```

## Development

### Linting

Install [golangci-lint](https://golangci-lint.run/):

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

Run linter:

```bash
make lint
```

### Testing

```bash
make test          # Run tests
make test-cover    # Run tests with coverage
```

## Documentation

- [Architecture](docs/architecture.md) — How it works
- [Configuration](docs/configuration.md) — Configuration reference
- [API Reference](docs/api.md) — HTTP API documentation
- [Monitoring](docs/monitoring.md) — Metrics, events, Prometheus
- [Deployment](docs/deployment.md) — Deployment guide

## License

MIT License. See [LICENSE](LICENSE) for details.
