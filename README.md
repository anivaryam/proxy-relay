# proxy-relay

A lightweight authenticated SOCKS5 + HTTP CONNECT proxy server. Deploy on Railway (or any container host) to route your traffic through a different region.

```
Your Machine → proxy-relay-server (Singapore) → Internet
```

## Features

- **SOCKS5 proxy** with username/password auth
- **HTTP CONNECT proxy** with Bearer token auth
- **Single port** — protocol auto-detected per connection
- **Token authentication** — no open relay
- **Lightweight** — pure Go stdlib server, no third-party proxy libs

## Install (Client)

```sh
# From source
git clone https://github.com/anivaryam/proxy-relay.git
cd proxy-relay
make install
```

## Quick Start

### 1. Deploy the server

Deploy to Railway, Docker, or any host:

```sh
# Docker
docker build -t proxy-relay .
docker run -p 8080:8080 -e PROXY_AUTH_TOKEN=your-secret-token proxy-relay

# Or from source
make build-server
PROXY_AUTH_TOKEN=your-secret-token ./bin/proxy-relay-server
```

### 2. Configure the client

```sh
proxy-relay config set-server socks5h://your-server:port
proxy-relay config set-token your-secret-token
```

### 3. Enable the proxy

```sh
proxy-relay on
source ~/.proxy-relay/proxy.env   # for terminal apps
```

### 4. Disable when done

```sh
proxy-relay off
unset http_proxy https_proxy all_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY
```

## Deploy on Railway

1. Push this repo to GitHub
2. Connect it to [Railway](https://railway.app)
3. Railway auto-detects the Dockerfile
4. Set environment variable: `PROXY_AUTH_TOKEN=your-secret-token`
5. Under Networking, enable **TCP Proxy** to get a public `host:port`
6. Use that `host:port` as your server address

> **Important:** Enable TCP Proxy in Railway networking settings (not just the default HTTP). SOCKS5 needs raw TCP access.

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PROXY_AUTH_TOKEN` | Yes | Auth token for client connections |
| `PORT` | No | Listen port (default `8080`) |

## CLI Usage

```
proxy-relay on                         Enable system proxy
proxy-relay off                        Disable system proxy
proxy-relay status                     Show proxy status
proxy-relay config set-server <addr>   Set server address
proxy-relay config set-token <token>   Set auth token
proxy-relay config show                Show config
```

## Browser Setup

After `proxy-relay on`, configure your browser to use the SOCKS5 proxy:

**Firefox:** Settings → Network → Manual proxy → SOCKS Host: `<host>`, Port: `<port>`, SOCKS v5, check "Proxy DNS"

**Chrome:** Use a proxy extension like SwitchyOmega, or launch with:
```sh
google-chrome --proxy-server="socks5://<host>:<port>"
```

## How It Works

The server listens on a single TCP port. On each connection, it peeks at the first byte to detect the protocol:
- `0x05` → SOCKS5 (username/password auth, password = token)
- Otherwise → HTTP CONNECT (Proxy-Authorization: Bearer token)

Both protocols authenticate against the same `PROXY_AUTH_TOKEN`, then relay traffic bidirectionally to the target.

## Configuration

Client config is stored at `~/.proxy-relay/config.yml`:

```yaml
server_addr: socks5h://your-server:port
auth_token: your-secret-token
```

## License

MIT
