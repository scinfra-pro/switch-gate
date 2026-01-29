# Configuration

switch-gate uses YAML configuration with environment variable expansion.

## Configuration File

Default location: `/etc/switch-gate/config.yaml`

Override with: `switch-gate -config /path/to/config.yaml`

## Full Configuration Reference

```yaml
# Server configuration
server:
  # SOCKS5 proxy listen address
  listen: "0.0.0.0:18388"
  
  # Transparent proxy listen address (Linux only, optional)
  transparent: "0.0.0.0:18389"
  
  # HTTP API listen address
  api: "127.0.0.1:9090"

# Routing modes configuration
modes:
  # Direct mode - uses default routing
  direct:
    # Network interface name (optional, usually empty)
    interface: ""
    
    # Bind to specific local IP (optional)
    # Useful to bypass tunnel routing when connecting to upstream proxy
    local_ip: ""
  
  # Tunnel mode - routes through tunnel interface
  warp:
    # Tunnel interface name (e.g., warp0, WARP, wg0)
    interface: "WARP"
  
  # Home mode - routes through upstream SOCKS5 proxy
  home:
    # Proxy type (currently only socks5 is supported)
    type: "socks5"
    
    # Proxy host
    host: "proxy.example.com"
    
    # Proxy port
    port: 7000
    
    # Proxy username (optional)
    username: "your_username"
    
    # Proxy password (supports environment variable expansion)
    password: "${PROXY_PASSWORD}"

# Traffic limits
limits:
  home:
    # Maximum traffic in MB (0 = unlimited)
    max_mb: 100
    
    # Mode to switch to when limit is reached
    auto_switch_to: "warp"

# Webhook notifications (optional)
webhooks:
  # Enable webhook notifications
  enabled: false
  
  # Webhook receiver URL
  url: "http://${WEBHOOK_HOST}/webhook/switch-gate"
  
  # Shared secret for authentication
  secret: "${WEBHOOK_SECRET}"
  
  # VPS identifier included in event payload
  source: "my-vps"
  
  # Event filtering (which events to send)
  events:
    # Send when mode changes (via API or auto-switch)
    mode_changed: false
    
    # Send when home proxy limit is exhausted
    limit_reached: true

# Logging configuration
logging:
  # Log level: debug, info, warn, error
  level: "info"
  
  # Log format: json, text
  format: "json"
```

## Environment Variables

Configuration values can reference environment variables using `${VAR_NAME}` syntax:

```yaml
home:
  password: "${PROXY_PASSWORD}"
```

Set the environment variable before running:

```bash
export PROXY_PASSWORD="your-secret-password"
switch-gate -config config.yaml
```

## Minimal Configuration

```yaml
server:
  listen: "0.0.0.0:18388"
  api: "127.0.0.1:9090"

modes:
  direct:
    interface: ""
```

This enables only direct mode with SOCKS5 proxy and API.

## Mode-Specific Configuration

### Direct Mode with IP Binding

Bind to a specific local IP to bypass tunnel routing:

```yaml
modes:
  direct:
    local_ip: "203.0.113.10"
```

### Tunnel Mode

Requires a tunnel interface to be configured on the system:

```yaml
modes:
  warp:
    interface: "warp0"
```

The interface must exist and have policy routing configured.

### Upstream Proxy Mode

Route traffic through an external SOCKS5 proxy:

```yaml
modes:
  home:
    type: "socks5"
    host: "proxy.example.com"
    port: 7000
    username: "user"
    password: "${PROXY_PASSWORD}"
```

If `modes.direct.local_ip` is set, connections to the upstream proxy will use that IP to bypass tunnel routing.

## Traffic Limits

Set a traffic limit for home mode:

```yaml
limits:
  home:
    max_mb: 100
    auto_switch_to: "warp"
```

When the limit is reached:
1. Router switches to the specified fallback mode
2. New connections to home mode are rejected until restart

## Security Considerations

1. **API binding:** Bind API to localhost only (`127.0.0.1:9090`) for security
2. **Passwords:** Use environment variables for sensitive values
3. **File permissions:** Restrict config file permissions (`chmod 600`)
