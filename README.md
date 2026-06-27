# ycair.online

P2P VPN for playing Minecraft with friends across the internet. Room code + password, no config needed.

## Download

[Latest Release](https://github.com/ycair/ycair.online/releases/latest)

| Platform | File |
|----------|------|
| macOS | `ycair.online_*.dmg` |
| Windows | `ycair.online-windows-msi.zip` |
| Linux | `ycair-core-x86_64-unknown-linux-gnu` |

## How to Use

### macOS
Download `.dmg`, drag to Applications, open. Enter room code + password, click Connect. macOS asks for admin password (VPN adapter).

### Windows
Download `.msi.zip`, unzip, run installer. Open from Start Menu. Right-click → Run as Administrator. Enter room code + password, click Connect.

### Linux
```bash
./ycair-core-x86_64-unknown-linux-gnu <room> <password> signal.ycair.space
```

### Minecraft
1. Everyone connects to ycair with same room code + password
2. Host: Minecraft → Singleplayer → Open to LAN
3. Others: Multiplayer → Direct Connect → host's VPN IP

Java Edition: `10.99.0.2` (TCP port 25565). Bedrock: `10.99.0.2:19132` (UDP).

## How It Works

```
You ← WSS → signal.ycair.space ← WSS → Friend
                  |
           peer discovery
                  |
You ← UDP (encrypted) → Friend
```

- Signaling: `wss://signal.ycair.space/ws`
- NAT traversal: STUN + UDP hole punching
- Encryption: Noise NN + ChaChaPoly
- VPN subnet: `10.99.0.0/24`

## Security

- Password: HMAC-SHA256 salted per room, rate limited (5/10s per IP)
- P2P tunnel: Noise NN + ChaChaPoly
- Signaling channel: WSS via Cloudflare Tunnel
- Room isolation: different rooms cannot see each other

## Build

```bash
git clone https://github.com/ycair/ycair.online.git
cd ycair.online && npm install
bash scripts/build-sidecars.sh
npm run tauri build
```
