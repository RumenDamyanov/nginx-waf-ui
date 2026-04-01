# nginx-waf-ui

Web management interface for nginx-waf.

## Overview

nginx-waf-ui is a web-based management interface for nginx-waf. It provides a
dashboard for managing IP lists, viewing block statistics, and configuring WAF
settings through a browser.

## Features

- Dashboard with block statistics and trends
- IP list management (add, remove, search)
- Tag-based list organization
- Block log viewer
- Feed status monitoring
- User management with roles
- Audit log

## Architecture

```
Browser ──> nginx-waf-ui ──> nginx-waf-api ──> nginx-waf
             (Go + htmx)     (REST API)       (C module)
```

## Quick Start

```bash
# Build
make build

# Configure
cp conf/config.example.yaml /etc/nginx-waf-ui/config.yaml
# Edit configuration...

# Run
./nginx-waf-ui -config /etc/nginx-waf-ui/config.yaml
```

## Prerequisites

- nginx-waf-api running and accessible

## Installation

### From OBS packages (recommended)

Available for Fedora, openSUSE, Debian, and Ubuntu via
[OBS](https://build.opensuse.org/package/show/home:rumenx/nginx-waf-ui).

### From source

```bash
make build
sudo make install
sudo cp conf/config.example.yaml /etc/nginx-waf-ui/config.yaml
sudo cp dist/nginx-waf-ui.service /etc/systemd/system/
sudo systemctl enable --now nginx-waf-ui
```

## Related Projects

### nginx-waf Ecosystem

- [nginx-waf](https://github.com/RumenDamyanov/nginx-waf) - Core nginx module (required)
- [nginx-waf-api](https://github.com/RumenDamyanov/nginx-waf-api) - REST API (required)
- [nginx-waf-feeds](https://github.com/RumenDamyanov/nginx-waf-feeds) - Threat feed updater
- [nginx-waf-lua](https://github.com/RumenDamyanov/nginx-waf-lua) - OpenResty Lua integration

### Other Nginx Modules

- [nginx-torblocker](https://github.com/RumenDamyanov/nginx-torblocker) - Control access from Tor exit nodes
- [nginx-cf-realip](https://github.com/RumenDamyanov/nginx-cf-realip) - Automatic Cloudflare IP list fetcher for real client IP restoration
- [nginx-gone](https://github.com/RumenDamyanov/nginx-gone) - Return HTTP 410 Gone for permanently removed URIs

## License

BSD 3-Clause License - see [LICENSE.md](LICENSE.md) for details.
