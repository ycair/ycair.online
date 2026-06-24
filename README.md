# ycair.online

> 跨平台 P2P 加密網狀 VPN —— macOS 與 Windows 11 可互聯互通
> Cross-platform P2P encrypted mesh VPN — macOS and Windows 11 interconnect seamlessly

---

## 架構 / Architecture

```
┌─────────────────────────────────┐
│  前端 / Frontend                 │  ← Tauri 桌面殼層 / desktop shell
│  (Next.js 15 + React 19)        │
├─────────────────────────────────┤
│  Tauri (Rust)                   │  ← Sidecar 進程管理器 / process mgr
├─────────────────────────────────┤
│  Go 核心 / Go Core (ycair-core) │  ← VPN 引擎 / VPN engine
│  ├── signaling  信令客戶端        │
│  ├── p2p        NAT 穿透 + 打洞   │
│  ├── crypto     Noise NN + ChaChaPoly
│  ├── tun        虛擬網卡           │
│  └── mesh       封包路由與轉發     │
├─────────────────────────────────┤
│  信令伺服器 / Signaling Server   │  ← 節點發現 + 房間管理
│  (Go + WebSocket)               │
└─────────────────────────────────┘
```

| 組件 / Component | 說明 / Description |
|---|---|
| **信令 / Signaling** | WebSocket 房間/節點發現，密碼保護 / password-protected room & peer discovery |
| **NAT 穿透 / NAT Traversal** | Google STUN 公網位址發現 + UDP Hole Punching |
| **加密 / Encryption** | Noise NN 握手 → ChaChaPoly AEAD，每節點獨立通道 / per-peer channel |
| **虛擬網路 / Virtual Net** | TUN 介面，`10.99.0.0/24` 子網，每房間最多 253 節點 |

---

## 平台支援 / Platform Support

| 平台 / Platform | TUN 驅動 / Driver | 狀態 / Status |
|---|---|---|
| macOS (Sequoia+) | `water` (utun) | ✅ 完整支援 / Full |
| Linux | `water` (/dev/tun) | ✅ 完整支援 / Full |
| Windows 11 | `wireguard-go` (Wintun) | ✅ 完整支援 / Full |

> **跨平台互聯 / Cross-Platform Interconnect**: macOS 與 Windows 節點使用相同的 Noise NN + ChaChaPoly 加密協定、相同的信令協議和相同的 IP 路由表（`10.99.0.0/24`），**100% 互通**。唯一差異是底層 TUN 驅動實現，對上層完全透明。
>
> macOS and Windows peers share the same Noise NN + ChaChaPoly encryption, signaling protocol, and IP routing table (`10.99.0.0/24`) — **100% interoperable**. Only the low-level TUN driver differs, transparent to upper layers.

---

## 環境需求 / Prerequisites

| 工具 / Tool | 版本 / Version | 用途 / Purpose |
|---|---|---|
| Go | 1.25+ | 核心與信令伺服器 / core & signaling |
| Node.js | 22+ | 前端建置 / frontend build |
| Rust | 1.77+ | Tauri 桌面建置 / desktop build |
| Tauri CLI | 2.x | `cargo install tauri-cli` |
| cloudflared | latest | 可選：公網信令隧道 / optional public tunnel |

### Windows 11 額外需求 / Windows 11 Extra

- **Wintun 驅動**：由 `wireguard-go` 內嵌，無需手動安裝 / embedded in `wireguard-go`, no manual install
- **管理員權限**：首次建立 TUN 介面需要 / admin rights required for first TUN creation

### macOS 額外需求 / macOS Extra

- **系統擴展**：macOS 會提示授權網路擴展 / system may prompt for network extension permission
- `utun` 介面由系統自動管理 / utun interfaces managed automatically by the OS

---

## 快速開始 / Quick Start

### 1. 安裝依賴 / Install Dependencies

```bash
npm install
```

### 2. 啟動信令伺服器 / Run Signaling Server

```bash
cd signaling-server && go run .
# 信令伺服器運行於 / Signaling server on ws://localhost:9090/ws
```

### 3. 執行 Go 核心 / Run Go Core (standalone)

```bash
cd core && go run . <房間碼/room_code> <密碼/password> localhost:9090
```

### 4. 啟動桌面應用 / Run Desktop App

```bash
npm run tauri dev
```

### 5. 公網隧道（可選）/ Public Tunnel (optional)

```bash
# Cloudflare 快速隧道 / quick tunnel
bash scripts/quick-tunnel.sh

# 或設定永久隧道 / or permanent tunnel
bash scripts/setup-tunnel.sh
```

---

## 建置 / Build

### 跨平台 Sidecar / Cross-Platform Sidecars

```bash
bash scripts/build-sidecars.sh
```

產出 / Produces `ycair-core` for:

| 平台 / Platform | 架構 / Arch | TUN 後端 / Backend |
|---|---|---|
| macOS | ARM64, x86_64 | `water` (utun) |
| Linux | x86_64 | `water` (/dev/tun) |
| Windows | x86_64 | `wireguard-go` (Wintun) |

另含信令伺服器 / Plus signaling server binary.

### 桌面應用 / Desktop App

```bash
npm run tauri build
```

---

## 技術棧 / Tech Stack

| 層 / Layer | 技術 / Technology |
|---|---|
| 前端 / Frontend | Next.js 15, React 19, Tailwind CSS 4, shadcn/ui |
| 桌面 / Desktop | Tauri 2 |
| VPN 核心 (macOS/Linux) | Go, `water` (TUN), `flynn/noise`, `pion/stun` |
| VPN 核心 (Windows) | Go, `wireguard-go/tun` (Wintun), `flynn/noise`, `pion/stun` |
| 信令 / Signaling | Go, `gorilla/websocket` |
| 公網隧道 / Tunnel | Cloudflare Tunnel / cloudflared |

---

## 專案結構 / Project Structure

```
.
├── app/                  # Next.js App Router
├── components/           # React 元件 + shadcn/ui
├── core/                 # Go VPN 引擎 / engine
│   ├── crypto/           #   Noise NN + ChaChaPoly 加密
│   ├── mesh/             #   封包路由與轉發 / packet routing & forwarding
│   ├── p2p/              #   NAT 穿透 + 節點連線管理
│   ├── signaling/        #   WebSocket 信令客戶端
│   └── tun/              #   TUN 虛擬網卡（macOS/Linux/Windows 三平台實作）
├── signaling-server/     # Go WebSocket 信令伺服器
├── src-tauri/            # Tauri 桌面應用 (Rust)
├── scripts/              # 建置與隧道輔助腳本
├── config/               # Cloudflare 隧道設定
├── note/                 # 規劃筆記 (markdown)
└── public/               # 靜態資源
```

---

## 安全性 / Security

| 機制 / Mechanism | 說明 / Description |
|---|---|
| 房間密碼 / Room Password | SHA256 哈希驗證，伺服器不存明文 / hashed verification, server never stores plaintext |
| P2P 傳輸 / P2P Transport | Noise NN 握手 → ChaChaPoly AEAD 對稱加密 |
| 信令通道 / Signaling Channel | 可選 WSS 加密（Cloudflare Tunnel）/ optional WSS via Cloudflare |
| 信任模型 / Trust Model | 基於共享房間密碼，無需 CA 憑證 / anchored in shared room password, no CA needed |

---

## 跨平台設計細節 / Cross-Platform Design Notes

### TUN 抽象層 / TUN Abstraction

```
core/tun/
├── tun.go           # 共用常數、MTU 定義 / shared constants
├── tun_darwin.go    # macOS: water 庫 + utun + ifconfig/route
├── tun_linux.go     # Linux: water 庫 + /dev/tun + ip addr/link
└── tun_windows.go   # Windows: wireguard-go/tun + Wintun + netsh
```

三個平台檔案匯出完全相同的 `Interface` 型別與方法（`Create`, `Read`, `Write`, `Close`, `Name`, `IP`），mesh 層無需感知平台差異。

All three platform files export an identical `Interface` type and method set. The mesh layer is platform-agnostic.

### 互通性保證 / Interoperability Guarantee

- 封包格式：IPv4 over TUN，所有平台一致 / packet format: IPv4 over TUN, identical across platforms
- 加密協定：Noise NN + ChaChaPoly，完全跨平台 / encryption: fully cross-platform
- 信令協議：JSON over WebSocket，完全跨平台 / signaling: fully cross-platform
- IP 分配：由信令伺服器統一分配 `10.99.0.x`，與客戶端平台無關 / IP assignment: server-managed, platform-independent
