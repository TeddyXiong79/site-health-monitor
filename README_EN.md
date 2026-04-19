# Site Health Monitor

English | [中文](README.md)

A real-time visualization Dashboard for monitoring OpenClash proxy node health status, with multi-region node classification, latency statistics, and fault alerting.

<p align="center">
  <img src="https://img.shields.io/github/v/release/TeddyXiong79/site-health-monitor?style=flat-square" alt="Release">
  <img src="https://img.shields.io/github/license/TeddyXiong79/site-health-monitor?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Platform-amd64%20%7C%20arm64-blue?style=flat-square" alt="Platform">
</p>

## ✨ Features

- **Node Health Monitoring** — Auto-fetches OpenClash API data, displays real-time latency status for all proxy nodes
- **Smart Classification** — Auto-groups by region (Hong Kong/Singapore/Taiwan/Japan/US etc.) and latency level
- **Real-time Statistics** — Health percentage, fast/normal/slow/fault node counts at a glance
- **Cyberpunk Dashboard** — Matrix Rain background effects, wave digit animations, auto-optimized for mobile
- **Proxy Switching** — Click any node name to switch the current node of the `🔰国外流量` (Foreign Traffic) selector group (group name is hardcoded, see Configuration below)
- **Auto Refresh** — Configurable refresh interval (default **300s**), with manual delay check trigger
- **External API** — RESTful API with Bearer Token authentication for integration with Claw/AI and other systems
- **Data Caching** — 10-second TTL cache layer to prevent redundant upstream requests under high concurrency
- **Config Persistence** — Docker Volume auto-persistence, configuration survives container restarts
- **Containerized Deployment** — Multi-platform Docker images (amd64/arm64), scratch-based ultra-minimal

## 🏗️ Architecture

```
┌──────────────┐    ┌──────────────────────────────┐    ┌───────────────┐
│   Browser    │───▶│     Go HTTP Server            │───▶│  OpenClash    │
│  Dashboard   │◀───│     (gorilla/mux)             │◀───│  Router API   │
└──────────────┘    │                              │    └───────────────┘
                    │  ┌─────────┐ ┌────────────┐  │
┌──────────────┐    │  │ Cache   │ │ Rate       │  │
│  Claw/AI     │───▶│  │ 10s TTL │ │ Limiter    │  │
│  MCP Client  │◀───│  └─────────┘ └────────────┘  │
└──────────────┘    │                              │
                    │  ┌──────────────────────────┐ │
                    │  │ /data/config.json        │ │
                    │  │ Docker Volume Persistence │ │
                    │  └──────────────────────────┘ │
                    └──────────────────────────────┘
```

### Tech Stack

| Component | Technology | Description |
|-----------|-----------|-------------|
| Web Framework | [gorilla/mux](https://github.com/gorilla/mux) | HTTP routing |
| Config Management | [spf13/viper](https://github.com/spf13/viper) | JSON config read/write |
| Rate Limiting | [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) | Per-IP Token Bucket |
| Frontend | Vanilla HTML/CSS/JS | Cyberpunk style, zero dependencies |
| Container | scratch base image | Minimal: binary + templates + CA certs only |

### Data Flow

1. **Dashboard/API request** → Cache layer check (10s TTL)
2. **Cache miss** → HTTP request to OpenClash API for node data
3. **Processing** → Region classification, latency grading, sorting & statistics
4. **Response** → Dashboard rendering or JSON API response

## 🚀 Quick Start

### Docker (Recommended)

```bash
# One-command start (config auto-persists, no extra setup needed)
docker run -d \
  --name site-health-monitor \
  --restart always \
  -p 9099:9099 \
  ghcr.io/teddyxiong79/site-health-monitor:latest
```

After first launch, visit `http://localhost:9099/` and configure the data source at the bottom of the Dashboard. Config is automatically saved to a Docker Volume and survives container restarts.

```bash
# Completely remove container and config data
docker rm -v site-health-monitor
```

### Docker Compose (Recommended for Production)

```yaml
services:
  site-health-monitor:
    image: ghcr.io/teddyxiong79/site-health-monitor:latest
    container_name: site-health-monitor
    ports:
      - "9099:9099"
    volumes:
      - monitor-data:/data
    restart: unless-stopped

volumes:
  monitor-data:
```

```bash
# Start
docker compose up -d

# Complete cleanup (container + config data)
docker compose down -v
```

### Binary

```bash
chmod +x site-health-monitor
./site-health-monitor
```

Access: http://localhost:9099/

## ⚙️ Configuration

### 🚨 OpenClash Prerequisite (Important)

This program **requires** an OpenClash selector group literally named **`🔰国外流量`** (with the emoji prefix; the Chinese text means "Foreign Traffic"). The group name is **hardcoded** in `fetcher.go` and is used by:

- **Delay refresh** (`POST /api/refresh`) — triggers a latency test on all nodes in `🔰国外流量`
- **Proxy switching** (`POST /api/switch`) — changes the currently selected node in `🔰国外流量`

> ⚠️ If your OpenClash configuration does not have this exact group name (e.g. you use `🚀节点选择` or `Proxy`), both endpoints will return **404 group not found**. The node data dashboard still works, but the "click to switch" and "refresh delay" features will fail.
>
> To use a different group name, edit `fetcher.go:206` and `fetcher.go:256`, then rebuild.

### Dashboard Settings

Configure on the Dashboard after first launch:

| Parameter | Description | Default |
|-----------|-------------|---------|
| Data Source Address | OpenClash router IP | - |
| Data Source Port | OpenClash API port | `9090` |
| API Secret | OpenClash control panel secret | - |
| Refresh Interval | Dashboard auto-refresh interval (seconds) | `300` |

Config file `config.json`:

```json
{
  "api_address": "",
  "api_secret": "",
  "api_source_port": "9090",
  "port": "9099",
  "refresh_seconds": 300
}
```

> 💡 Leave the secret field empty when modifying config to keep the existing secret unchanged.

## 📡 API Reference

### Internal APIs (Dashboard same-origin calls)

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/` | Dashboard page | None |
| GET | `/api/data` | Get current node data and statistics | None |
| GET | `/api/health` | Health check | None |
| POST | `/api/config` | Save configuration | Bearer Token (when secret is set) |
| POST | `/api/test` | Test OpenClash connection | Bearer Token (when secret is set) |
| POST | `/api/refresh` | Trigger latency test for the `🔰国外流量` group (rate limited: 1/10s) | None |
| POST | `/api/switch` | Switch the current node in the `🔰国外流量` group (rate limited: 1/10s) | Bearer Token (when secret is set) |

### External APIs (Bearer Token auth + rate limited)

All external APIs require header: `Authorization: Bearer <API_SECRET>`

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/summary` | Get statistics summary |
| GET | `/api/nodes` | Get all node list |
| GET | `/api/regions` | Get all regions with nodes |
| GET | `/api/regions/{code}/nodes` | Get nodes for specific region |
| GET | `/api/nodes/filter?region=&category=` | Filter by region and category |
| GET | `/api/nodes/{name}` | Get specific node details |

### Response Example

```json
// GET /api/summary
{
  "total": 103,
  "fast": 20,
  "normal": 38,
  "high_latency": 21,
  "fault": 24,
  "healthy_count": 58,
  "healthy_pct": 56
}
```

## 📊 Latency Classification

| Level | Latency Range | Description |
|-------|---------------|-------------|
| 🚀 Fast | ≤ 150ms | Premium nodes |
| ✅ Normal | 151 ~ 240ms | Usable nodes |
| ⚠️ Slow | 241 ~ 500ms | High latency nodes |
| ❌ Fault | > 500ms or ≤ 0ms | Unavailable nodes (including negative latency) |

## 🌍 Region Classification

Five auto-detected regions: **Hong Kong, Singapore, Taiwan, Japan, United States**. Unmatched nodes fall into "Other".

## 🔒 Security Features

- **Bearer Token Auth** — External APIs use constant-time comparison to prevent timing attacks
- **SSRF Protection** — Validates data source addresses, blocks loopback and URL injection
- **Response Size Limit** — Upstream responses capped at 10MB to prevent OOM
- **Per-IP Rate Limiting** — Separate rate limiters for external APIs and operation endpoints, supports X-Forwarded-For
- **Secret Masking** — Dashboard never exposes full secrets, only masked versions
- **Config Isolation** — Docker image contains no sensitive config, persisted via Volume at runtime

## 🐳 Docker Images

| Image | Description |
|-------|-------------|
| `ghcr.io/teddyxiong79/site-health-monitor:latest` | Latest stable |
| `ghcr.io/teddyxiong79/site-health-monitor:v1.6.2` | Specific version |

Supported platforms: `linux/amd64`, `linux/arm64`.

## 🛠️ Building

```bash
# Local build
git clone https://github.com/TeddyXiong79/site-health-monitor.git
cd site-health-monitor
go build -o site-health-monitor .

# Docker multi-platform build
docker buildx build --platform=linux/amd64,linux/arm64 \
  -t ghcr.io/teddyxiong79/site-health-monitor:latest --push .
```

## 📝 Changelog

- **v1.6.2** — Bugfix: Auth response JSON + Token sync + Latency classification fix
  - 🐛 All 401 auth failure responses changed from plain text to JSON format (8 places in handlers.go); external API clients no longer crash when parsing
  - 🐛 Dashboard now syncs the hidden `raw-token` field after changing the secret, preventing subsequent operations from using the old token
  - 🐛 `categorizeDelay` correctly classifies negative latency (≤0) as `fault` instead of `normal`
  - 📦 Default refresh interval changed from 120s to **300s**
- **v1.6.1** — Security hardening + Comprehensive testing + Documentation sync
  - 🔒 `/api/config` and `/api/test` now require Bearer Token auth (first-time setup without secret is allowed)
  - 🔒 `maskToken()` fix: short secrets (≤4 chars) no longer exposed in plaintext, returns `***`
  - 🐛 `ValidateConfig` fix: allows empty API secret (compatible with first-time setup and no-secret OpenClash), added data source address validation
  - 📝 API docs (`docs/api.md`) fully synced: latency classification aligned with code, removed deprecated `/api/token` endpoint
  - ✅ Added 40+ unit tests with 100% coverage on core functions
  - 📦 Windows AMD64 executable available for download
- **v1.6.0** — Security hardening + Performance optimization + Config persistence
  - 🔒 Removed plaintext secret from Dashboard HTML, introduced SafeConfig masked rendering
  - 🔒 SSRF protection (address validation), response size limit (10MB)
  - 🔒 External API Bearer Token auth with constant-time comparison
  - 🔒 Removed /api/token endpoint (security risk)
  - ⚡ Added 10s TTL data cache layer to prevent redundant upstream requests
  - ⚡ Rate limiter fix: correct IP extraction (strip port), X-Forwarded-For/X-Real-IP support
  - ⚡ Rate limiting for /api/refresh and /api/switch (1 per 10s)
  - ⚡ Mobile auto-disables Matrix Rain animation to save CPU/battery
  - 🐛 Graceful shutdown via srv.Shutdown (waits for in-flight requests)
  - 🐛 Fixed "leave empty to keep" secret behavior mismatch between UI and backend
  - 🐛 Fixed fetchDataAndUpdate not checking HTTP status code causing data wipe
  - 🐛 Fixed countdown timer double-trigger and request storm risk
  - 🐛 Fixed internalLimiter goroutine leak
  - 🐛 Fixed error responses being cached causing delayed recovery
  - 🐛 Enhanced config validation (port range 1-65535, refresh interval 10-3600s)
  - 📦 Config persistence: Docker Volume auto-persistence, zero user action required
  - 📦 Dockerfile fixes: correct image version, no bundled secrets, CA certificates added
  - 📦 Default refresh interval changed to 120 seconds
  - 📦 Added .gitignore + config.json.example
- **v1.5.2** — Proxy node switching, 204 compatibility, default refresh 90s
- **v1.5.1** — Concurrency safety fixes, bug fixes, Docker build improvements
- **v1.5.0** — Initial release

## License

MIT
