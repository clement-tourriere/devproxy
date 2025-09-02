# DevProxy - Zero-Config Container Domain Mapper

## Overview
DevProxy is a tool that automatically provides HTTPS access to all Docker containers at predictable URLs, replicating Orbstack's container domain functionality.

### Features
- **Zero configuration**: No labels or configuration needed for containers
- **Automatic domain mapping**:
  - Standalone containers: `https://container_name.localhost`
  - Compose services: `https://service.project_name.localhost`
- **Automatic HTTPS** with local certificates
- **Container IP support**: Direct routing to container IPs (no port mapping needed)
- **Optional overrides**: Environment variables or labels for advanced users

## Architecture

### Components
1. **devproxy**: Go service monitoring Docker events and generating Caddy configuration
2. **Caddy**: Reverse proxy handling HTTPS and routing
3. **Docker API integration**: Automatic container discovery

### Domain Mapping Rules
- Container name: `container_name.localhost`
- Compose service: `service.project_name.localhost`
- Compose project: `project_name.localhost`

### Port Detection
1. Custom port via `DEVPROXY_PORT` environment variable
2. Custom port via `devproxy.port` label
3. First exposed port in container
4. Common web ports (80, 8080, 3000, 8000, 5000)

### Optional Configuration
- `devproxy.enabled=false` - Disable proxy for container
- `devproxy.domain=custom.localhost` - Custom domain
- `devproxy.port=3000` - Custom port
- `DEVPROXY_PORT=3000` - Environment variable for port

## Usage

### Development Setup
```bash
# Use Go 1.23 with mise
mise use go@1.23

# Build and run
go build -o devproxy ./cmd/devproxy
docker compose up -d
```

### Running
```bash
# Start DevProxy stack (includes dashboard)
docker compose up -d

# Containers will automatically be available at:
# - https://container_name.localhost
# - https://service.project_name.localhost (for compose services)
```

### Dashboard Access
DevProxy includes a web dashboard to view all active containers:

**🌐 Dashboard URL**: https://devproxy-dashboard.localhost

Features:
- View all active containers and their domains
- Container status, IPs, and ports
- Quick links to access services
- Real-time updates (auto-refresh every 30s)

### Testing
Create a test container:
```bash
docker run -d --name nginx nginx:alpine
# Available at: https://nginx.localhost
```

Create a compose service:
```yaml
# compose.yaml
services:
  web:
    image: nginx:alpine

# Run: docker compose up -d
# Available at: https://web.myproject.localhost
```

## File Structure
```
devproxy/
├── cmd/
│   ├── devproxy/main.go         # Main DevProxy application
│   └── dashboard/main.go        # Dashboard application
├── internal/
│   ├── docker/
│   │   ├── monitor.go           # Docker event monitoring
│   │   └── discovery.go         # Container discovery logic
│   ├── caddy/
│   │   ├── config.go            # Caddy config generation
│   │   └── api.go               # Caddy admin API client
│   ├── proxy/
│   │   └── manager.go           # Main orchestration logic
│   └── dashboard/
│       └── server.go            # Web dashboard server
├── compose.yaml                 # Docker Compose setup
├── Caddyfile                    # Initial Caddy configuration
├── Dockerfile                   # DevProxy container image
├── Dockerfile.dashboard         # Dashboard container image
├── go.mod                       # Go dependencies
├── README.md                    # User documentation
├── INSTALL.md                   # Installation guide
└── CLAUDE.md                    # Development documentation
```

## Development Notes
- Uses Go 1.23+ for `log/slog` package
- Requires Docker socket access for container monitoring
- Caddy admin API runs on port 2019
- HTTP/HTTPS traffic on ports 80/443
- Dashboard runs on port 8080
- Uses `devproxy` Docker network for isolation

## Services in Docker Compose
- `caddy`: Reverse proxy handling HTTPS and routing
- `devproxy`: Main manager monitoring Docker events
- `dashboard`: Web interface showing active containers
