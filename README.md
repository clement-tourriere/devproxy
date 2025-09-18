# DevProxy 🚀

**Zero-config HTTPS proxy for Docker containers**

DevProxy automatically provides HTTPS access to all your Docker containers at predictable URLs, just like Orbstack's container domains feature. No configuration required!

## ✨ Features

- 🔧 **Zero Configuration**: No labels, env vars, or setup needed
- 🌐 **Automatic HTTPS**: Local CA with automatic certificate generation
- 📋 **Predictable URLs**: `https://container_name.localhost` and `https://service.project_name.localhost`
- 🔄 **Real-time Discovery**: Automatically detects container start/stop events
- 🎯 **Smart Port Detection**: Uses exposed ports or common web ports (80, 8080, 3000, 8000, 5000)
- 🛡️ **Container IP Support**: Direct routing without port mapping
- ⚙️ **Optional Overrides**: Custom domains and ports when needed

## 🚀 Quick Start

### Prerequisites

- Docker and Docker Compose

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/clement-tourriere/devproxy
   cd devproxy
   ```

2. **Start DevProxy**
   ```bash
   docker compose up -d
   ```

3. **Test it works**
   ```bash
   # Create a test container
   docker run -d --name test-nginx nginx:alpine

   # Visit https://test-nginx.localhost in your browser
   # You should see the nginx welcome page with HTTPS!
   ```

4. **Install trusted certificates (optional but recommended)**
   ```bash
   # Run the certificate installer
   ./trust-cert.sh

   # Restart your browser
   # Now visit https://test-nginx.localhost - no more security warnings!
   ```

That's it! 🎉

> **📱 macOS DNS Note**: `*.localhost` domains work in browsers (Chrome, Firefox, Safari) and curl, but may not work in all CLI tools or applications. For broader compatibility, consider adding entries to `/etc/hosts`.

## 🔒 HTTPS Certificate Setup

DevProxy uses Caddy's local Certificate Authority (CA) to provide automatic HTTPS. By default, browsers will show security warnings because the CA isn't trusted by your system.

### Automatic Certificate Installation

Run the included script to install DevProxy's root certificate in your system trust store:

```bash
./trust-cert.sh
```

This script works on:
- **macOS**: Installs certificate in System Keychain
- **Linux**: Supports Debian/Ubuntu (`update-ca-certificates`), RHEL/Fedora (`update-ca-trust`), and p11-kit
- **Windows**: Uses `certutil` (run from Administrator Command Prompt)

After installation:
1. Restart your browser
2. Visit any DevProxy domain - no more security warnings!
3. The dashboard will automatically use HTTPS links

### Manual Certificate Installation

If the automatic script doesn't work for your system:

1. **Export the certificate**:
   ```bash
   docker exec devproxy-caddy cat /data/caddy/pki/authorities/local/root.crt > caddy-root.crt
   ```

2. **Install in your system** (varies by OS - consult your system documentation)

3. **Restart your browser**

### HTTP Fallback

If you prefer not to install certificates, DevProxy automatically detects when certificates aren't trusted and provides HTTP links in the dashboard instead.

## 🆚 Why DevProxy?

DevProxy brings OrbStack's beloved container domains feature to any Docker setup:

| Feature | DevProxy | Manual nginx/traefik | Port Mapping |
|---------|----------|---------------------|--------------|
| **Setup** | Zero config | Complex configuration | Manual port assignment |
| **URLs** | `app.localhost` | Custom domain setup | `localhost:8080` |
| **HTTPS** | Automatic with local CA | Manual cert management | None |
| **Discovery** | Real-time container events | Manual service registration | Manual |
| **Multi-project** | Automatic namespacing | Complex routing rules | Port conflicts |

## 📖 How It Works

DevProxy consists of two main components:

1. **DevProxy Manager**: Monitors Docker events and manages container discovery
2. **Caddy Proxy**: Handles HTTPS termination and reverse proxying

### Domain Mapping Rules

| Container Type | Domain Format | Example |
|---------------|---------------|---------|
| Standalone Container | `container_name.localhost` | `nginx.localhost` |
| Compose Service | `service.project_name.localhost` | `web.myapp.localhost` |
| Custom Override | `devproxy.domain` label | `api.example.localhost` |

### Port Detection Priority

1. `DEVPROXY_PORT` environment variable
2. `devproxy.port` label
3. First exposed port in container
4. Common web ports: 80, 8080, 3000, 8000, 5000

### Performance & Resource Usage

DevProxy is designed to be lightweight and efficient:
- **Minimal overhead**: Uses Docker's event stream for real-time updates
- **Low memory**: < 20MB RAM usage for both DevProxy and Caddy combined
- **Fast startup**: < 2 seconds to fully initialize and proxy containers
- **Container IP routing**: Direct connection to containers, no port mapping bottleneck

## 🛠️ Usage Examples

### Basic Usage

```bash
# Start any container - it's automatically available via HTTPS
docker run -d --name my-app nginx:alpine
# → Available at https://my-app.localhost
```

### Docker Compose

```yaml
# compose.yaml
services:
  web:
    image: nginx:alpine
  api:
    image: node:alpine
    command: node server.js

# Run: docker compose up -d
# → web: https://web.myproject.localhost
# → api: https://api.myproject.localhost
# → project: https://myproject.localhost
```

### Custom Configuration

```bash
# Custom port via environment variable
docker run -d --name my-api -e DEVPROXY_PORT=3000 node:alpine

# Custom port via label
docker run -d --name my-api --label devproxy.port=3000 node:alpine

# Custom domain via label
docker run -d --name my-api --label devproxy.domain=api.mycompany.localhost node:alpine

# Disable proxy for a container
docker run -d --name utility --label devproxy.enabled=false alpine sleep 3600
```

## 🎛️ Dashboard

DevProxy includes a web dashboard to view all active container domains:

**📍 Dashboard URL**: `https://devproxy-dashboard.localhost`

The dashboard shows:
- All active containers and their domains
- Container status and ports
- Quick links to access services
- Real-time updates when containers start/stop

## 🔧 Development Setup

### Prerequisites

```bash
# Install mise for Go version management
curl https://mise.run | sh
```

```bash
# Initialize mise
# Trust mise
mise trust
mise install
```

### Build from Source

```bash

# Install dependencies
go mod tidy

# Build binary
go build -o devproxy ./cmd/devproxy

# Build Docker image
docker compose build
```

### Project Structure

```
devproxy/
├── cmd/
│   ├── devproxy/          # Main DevProxy application
│   └── dashboard/         # Dashboard application
├── internal/
│   ├── docker/            # Docker API integration & discovery
│   ├── caddy/             # Caddy configuration & API client
│   ├── proxy/             # Main orchestration logic
│   └── dashboard/         # Web dashboard server
├── compose.yaml           # Docker Compose setup
├── Dockerfile            # DevProxy container image
├── Dockerfile.dashboard   # Dashboard container image
├── Caddyfile             # Initial Caddy configuration
├── trust-cert.sh         # Certificate installation script
├── INSTALL.md            # Installation guide
├── CLAUDE.md             # Development documentation
└── README.md            # This file
```

## ⚙️ Configuration

DevProxy works perfectly without any configuration, but offers flexible customization options when needed.

### 🎛️ DevProxy Configuration

| Environment Variable | Description | Default | Example |
|---------------------|-------------|---------|---------|
| `DEVPROXY_LOG_LEVEL` | Logging verbosity (debug/info/warn/error) | `info` | `debug` |
| `DEVPROXY_DOMAIN_SUFFIX` | Domain suffix for containers | `localhost` | `dev.local` |
| `CADDY_ADMIN_URL` | Caddy admin API URL | `http://caddy:2019` | `http://localhost:2019` |

### 📊 Dashboard Configuration

| Environment Variable | Description | Default | Example |
|---------------------|-------------|---------|---------|
| `DEVPROXY_DASHBOARD_REFRESH` | Auto-refresh interval (seconds) | `30` | `10` |
| `DEVPROXY_DASHBOARD_EXCLUDE` | Projects to hide (comma-separated) | `devproxy` | `devproxy,test,staging` |
| `DEVPROXY_DASHBOARD_SHOW_ALL` | Show all containers including system ones | `false` | `true` |
| `DASHBOARD_ADDR` | Dashboard listening address | `:8080` | `:3000` |

### 🏷️ Container Labels

| Label | Description | Example |
|-------|-------------|---------|
| `devproxy.enabled` | Enable/disable proxy | `false` |
| `devproxy.domain` | Custom domain | `api.mycompany.localhost` |
| `devproxy.port` | Custom port | `3000` |

### 📝 Usage Examples

#### Default (Zero Configuration)
```bash
docker compose up -d
# Works perfectly with sensible defaults
```

#### Custom Dashboard Refresh Rate
```bash
# Dashboard updates every 10 seconds instead of 30
DEVPROXY_DASHBOARD_REFRESH=10 docker compose up -d
```

#### Show All Containers in Dashboard
```bash
# Include DevProxy's own containers in dashboard
DEVPROXY_DASHBOARD_SHOW_ALL=true docker compose up -d
```

#### Hide Multiple Projects
```bash
# Hide development and test projects from dashboard
DEVPROXY_DASHBOARD_EXCLUDE=devproxy,test,staging docker compose up -d
```

#### Debug Logging
```bash
# Enable detailed debug logging
DEVPROXY_LOG_LEVEL=debug docker compose up -d
```

#### Using .env File (Recommended)
Create a `.env` file in your DevProxy directory:
```bash
# .env
DEVPROXY_LOG_LEVEL=debug
DEVPROXY_DASHBOARD_REFRESH=15
DEVPROXY_DASHBOARD_EXCLUDE=devproxy,internal
DEVPROXY_DASHBOARD_SHOW_ALL=false
```

Then start normally:
```bash
docker compose up -d
```

#### Per-Container Configuration
```bash
# Custom port via environment variable
docker run -d --name my-api -e DEVPROXY_PORT=3000 node:alpine

# Custom port via label
docker run -d --name my-api --label devproxy.port=3000 node:alpine

# Custom domain via label
docker run -d --name my-api --label devproxy.domain=api.mycompany.localhost node:alpine

# Disable proxy for a container
docker run -d --name utility --label devproxy.enabled=false alpine sleep 3600
```

## ⚠️ Important Limitations

### Container-to-Container Communication

**`.localhost` domains do NOT work for container-to-container communication** because they resolve to 127.0.0.1 inside containers.

**✅ What works:**
- **Browser/Host access**: `https://myapp.localhost` ✅
- **Container-to-container via container names**: `http://myapp` ✅  
- **Container-to-container via IPs**: `http://172.18.0.3` ✅

**❌ What doesn't work:**
- **Container-to-container via .localhost**: `http://myapp.localhost` ❌ (resolves to 127.0.0.1)

### Docker Environment Compatibility

**✅ Works with:**
- Docker Engine on Linux
- Docker Desktop on macOS (in most configurations)
- Any environment where containers get individual IP addresses

**❌ May not work with:**
- Docker Desktop with certain network configurations
- Environments where containers share the host network
- Systems where container IPs are not directly routable

### DNS Setup (Optional)

For `*.localhost` domains to work system-wide in all applications:

**macOS/Linux**:
```bash
# Add to /etc/hosts (requires sudo)
127.0.0.1 myapp.localhost
127.0.0.1 devproxy-dashboard.localhost

# Or use dnsmasq for wildcard support
brew install dnsmasq
echo 'address=/localhost/127.0.0.1' > /usr/local/etc/dnsmasq.conf
sudo brew services start dnsmasq
```

**Windows**:
```powershell
# Add to C:\Windows\System32\drivers\etc\hosts
127.0.0.1 myapp.localhost
127.0.0.1 devproxy-dashboard.localhost
```


## 🔍 Troubleshooting

### Common Issues

**Container not accessible**
```bash
# Check if DevProxy is running
docker compose ps

# Check logs
docker compose logs devproxy
docker compose logs caddy

# Verify container has correct IP
docker inspect <container_name> | grep IPAddress
```

**HTTPS Certificate Issues**
```bash
# Restart DevProxy to regenerate certificates
docker compose restart

# Check Caddy certificate status
docker compose exec caddy caddy trust
```

**HTTPS stopped working after removing volumes**
```bash
# If you ran: docker compose down -v
# This removes Caddy's certificate volumes, generating new certificates

# Solution: Re-trust the new certificates
./trust-cert.sh

# Then restart your browser to use the new trusted certificates
```

**Port Detection Issues**
```bash
# Override port manually
docker run -d --name myapp -e DEVPROXY_PORT=8080 myimage

# Or use label
docker run -d --name myapp --label devproxy.port=8080 myimage
```

### Logs and Debugging

```bash
# View all logs
docker compose logs

# Follow DevProxy manager logs
docker compose logs -f devproxy

# Follow Caddy proxy logs
docker compose logs -f caddy

# Debug specific container
docker logs <container_name>
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes
4. Add tests if applicable
5. Commit: `git commit -am 'Add feature'`
6. Push: `git push origin feature-name`
7. Create a Pull Request

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [Orbstack's](https://orbstack.dev) container domains feature
- Built with [Caddy](https://caddyserver.com/) for automatic HTTPS
- Uses Docker API for container discovery

---

**🌟 Star this repo if DevProxy helps you!**

Need help? Open an issue or join our discussions!
