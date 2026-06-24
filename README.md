# ycair.online

Cross-platform P2P VPN with encrypted mesh networking. Connect multiple devices into a virtual private network over the internet.

## Architecture

```
┌─────────────────────────────────┐
│  Frontend (Next.js 15 + React)  │  ← Tauri desktop shell
├─────────────────────────────────┤
│  Tauri (Rust)                   │  ← Sidecar process manager
├─────────────────────────────────┤
│  Go Core (ycair-core)           │  ← VPN engine
│  ├── signaling  (WebSocket client)
│  ├── p2p        (NAT traversal + hole punching)
│  ├── crypto     (Noise NN + ChaChaPoly)
│  ├── tun        (Virtual network interface)
│  └── mesh       (Packet routing + forwarding)
├─────────────────────────────────┤
│  Signaling Server (Go)          │  ← Peer discovery + room mgmt
└─────────────────────────────────┘
```

- **Signaling**: WebSocket-based room/peer discovery with password-protected rooms
- **NAT Traversal**: STUN (Google) for public address discovery + UDP hole punching
- **Encryption**: Noise NN handshake → ChaChaPoly AEAD per peer
- **Virtual Network**: TUN interface with `10.99.0.0/24` subnet, up to 253 peers per room

## Prerequisites

- **Go** 1.25+ (for core and signaling server)
- **Node.js** 22+ (for frontend)
- **Rust** 1.77+ (for Tauri desktop build)
- **Tauri CLI** (`cargo install tauri-cli`)
- **cloudflared** (optional, for public signaling tunnel)

## Quick Start

### 1. Install dependencies

```bash
npm install
```

### 2. Run signaling server (local)

```bash
cd signaling-server && go run .
# Signaling server on ws://localhost:9090/ws
```

### 3. Run the Go core (standalone)

```bash
cd core && go run . <room_code> <password> localhost:9090
```

### 4. Run the desktop app

```bash
npm run tauri dev
```

### 5. Public tunnel (optional)

```bash
# Quick tunnel via Cloudflare
bash scripts/quick-tunnel.sh

# Or setup a permanent tunnel
bash scripts/setup-tunnel.sh
```

## Build

### Cross-platform sidecars

```bash
bash scripts/build-sidecars.sh
```

Builds `ycair-core` for:
- macOS ARM64 / x86_64
- Linux x86_64
- Windows x86_64 (MSVC)

Plus the signaling server.

### Desktop app

```bash
npm run tauri build
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Next.js 15, React 19, Tailwind CSS 4, shadcn/ui |
| Desktop | Tauri 2 |
| VPN Core | Go, water (TUN), noise (crypto), pion/stun (NAT) |
| Signaling | Go, gorilla/websocket |
| Tunnel | Cloudflare Tunnel / cloudflared |

## Project Structure

```
.
├── app/                  # Next.js App Router
├── components/           # React components + shadcn/ui
├── core/                 # Go VPN engine
│   ├── crypto/           #   Noise encryption
│   ├── mesh/             #   Packet routing
│   ├── p2p/              #   NAT traversal + peer connections
│   ├── signaling/        #   WebSocket client
│   └── tun/              #   TUN interface (macOS/Linux/Windows)
├── signaling-server/     # Go WebSocket signaling server
├── src-tauri/            # Tauri desktop app (Rust)
├── scripts/              # Build + tunnel helper scripts
├── config/               # Cloudflare tunnel config
├── note/                 # Planning notes (markdown)
└── public/               # Static assets
```

## Security

- Room access gated by SHA256 password hash
- P2P traffic encrypted via Noise NN + ChaChaPoly
- Signaling server can be exposed via WSS (Cloudflare Tunnel)
- No CA certificates required — trust anchored in shared room password

## Platform Support

| Platform | TUN Support | Status |
|----------|-------------|--------|
| macOS | ✅ `ifconfig` + `route` | Full |
| Linux | ✅ `ip addr` + `ip link` | Full |
| Windows | ⚠️ Requires [Wintun](https://www.wintun.net/) | Partial |
