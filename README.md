# ycair.online

> 跨平台 P2P 加密網狀 VPN — macOS / Windows 11 互聯互通
> Cross-platform P2P encrypted mesh VPN — macOS & Windows 11 interconnect seamlessly

---

## 📥 下載安裝 / Download & Install

### macOS

[![Download macOS](https://img.shields.io/badge/Download-macOS_.dmg-blue)](https://github.com/ycair/ycair.online/releases/latest)

1. 下載 `ycair.online_0.1.0_aarch64.dmg`
2. 雙擊開啟，將 `ycair.online` 拖入 `Applications`
3. 從 Applications 開啟 app
4. 首次連線時會要求管理員密碼（建立 VPN 虛擬網卡）

```
⚠️ macOS Gatekeeper: 若出現「無法驗證開發者」
→ 系統設定 → 隱私與安全性 → 強制開啟
```

### Windows 11

[![Download Windows](https://img.shields.io/badge/Download-Windows_x64-blue)](https://github.com/ycair/ycair.online/releases/latest)

Windows Tauri 桌面應用需在 Windows 上建置。目前提供 **Go 核心 CLI**（命令列直用）：

1. 下載 `ycair-core-x86_64-pc-windows-msvc.exe`
2. 下載 `signaling-server.exe`（或自建信令伺服器）
3. 以**系統管理員**開啟 terminal 執行：
```cmd
# 啟動信令伺服器（Host 端）
signaling-server.exe -port 9090

# 啟動 ycair-core
ycair-core-x86_64-pc-windows-msvc.exe <房間碼> <密碼> localhost:9090
```
4. Minecraft 中使用 `10.99.0.2` 作為伺服器 IP

> 💡 完整 Windows 桌面應用（`.msi`）即將推出，或參見下方 [從原始碼建置](#從原始碼建置--build-from-source)

---

## 🎮 使用方式 / How to Use

### Host（開房）

| 步驟 | 操作 |
|------|------|
| 1 | 選擇 **Host** 模式 |
| 2 | 輸入 Room Code（任意字串，如 `mc-world`） |
| 3 | 輸入 Password |
| 4 | 點 **Start Vibe** |
| 5 | macOS 彈出管理員密碼 → 輸入允許 |
| 6 | VPN IP 顯示（如 `10.99.0.2`）→ 告訴 Join 方 |

### Join（加入）

| 步驟 | 操作 |
|------|------|
| 1 | 選擇 **Join** 模式 |
| 2 | 輸入與 Host **相同的** Room Code + Password |
| 3 | Signaling Server 輸入 Host 的 IP`:9090` |
| 4 | 點 **Start Vibe** |
| 5 | Peers 列表出現 Host → 連線成功 |

### 🎮 Minecraft 互聯

**Host 端**：Minecraft → Singleplayer → Open to LAN
**Client 端**：Multiplayer → Direct Connect → `10.99.0.2:25565`

| 版本 | Host IP | Port |
|------|---------|------|
| Java Edition | `10.99.0.2`（Host 的 VPN IP） | 25565 |
| Bedrock Edition | `10.99.0.2` | 19132 |

---

## 🏗 架構 / Architecture

```
┌─────────────────────────────────┐
│  前端 / Frontend                 │  ← Tauri 桌面殼層 / desktop shell
│  (Next.js 15 + React 19)        │
├─────────────────────────────────┤
│  Tauri (Rust)                   │  ← Sidecar 進程管理 / process mgr
├─────────────────────────────────┤
│  Go 核心 / Go Core (ycair-core) │  ← VPN 引擎
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

| 組件 | 說明 |
|------|------|
| **信令** | WebSocket 房間/節點發現，SHA256 密碼保護 |
| **NAT 穿透** | Google STUN + UDP Hole Punching（50 次重試） |
| **加密** | Noise NN 握手 → ChaChaPoly AEAD，每節點獨立通道 |
| **虛擬網路** | TUN 介面，`10.99.0.0/24` 子網，最多 253 節點 |

---

## 📋 平台支援 / Platform Support

| 平台 | TUN 驅動 | 桌面應用 | CLI |
|------|---------|---------|-----|
| macOS (ARM64) | `water` (utun) | ✅ `.dmg` | ✅ |
| macOS (x86_64) | `water` (utun) | 🔧 需建置 | ✅ |
| Windows 11 | `wireguard-go` (Wintun) | 🔧 需建置 | ✅ |
| Linux | `water` (/dev/tun) | 🔧 需建置 | ✅ |

---

## 🔧 從原始碼建置 / Build from Source

### 環境需求

| 工具 | 版本 | 用途 |
|------|------|------|
| Go | 1.25+ | 核心 + 信令伺服器 |
| Node.js | 22+ | 前端建置 |
| Rust | 1.77+ | Tauri 桌面建置 |

### macOS

```bash
git clone https://github.com/ycair/ycair.online.git
cd ycair.online
npm install
bash scripts/build-sidecars.sh    # 建置 Go 核心
npm run tauri build                # 建置 .dmg
```

### Windows

```bash
git clone https://github.com/ycair/ycair.online.git
cd ycair.online
npm install
cd core && go build -o ../src-tauri/bin/ycair-core-x86_64-pc-windows-msvc.exe .
cd ../signaling-server && go build -o ../src-tauri/bin/signaling-server.exe .
cd .. && npm run tauri build        # 建置 .msi
```

---

## 📁 專案結構 / Project Structure

```
.
├── app/                  # Next.js App Router
├── components/           # React 元件 + shadcn/ui
├── core/                 # Go VPN 引擎
│   ├── crypto/           #   Noise NN + ChaChaPoly
│   ├── mesh/             #   封包路由與轉發
│   ├── p2p/              #   NAT 穿透 + 節點連線
│   ├── signaling/        #   WebSocket 信令客戶端
│   └── tun/              #   TUN 虛擬網卡（3 平台實作）
├── signaling-server/     # Go WebSocket 信令伺服器
├── src-tauri/            # Tauri 桌面應用 (Rust)
├── scripts/              # 建置與隧道腳本
├── note/                 # 規劃筆記 / planning notes
└── public/               # 靜態資源
```

---

## 🔐 安全性 / Security

- 房間密碼 SHA256 哈希，伺服器不存明文
- P2P 傳輸 Noise NN + ChaChaPoly AEAD 對稱加密
- 信令通道可選 WSS（Cloudflare Tunnel）
- 信任基於共享房間密碼，無需 CA 憑證

---

## 📝 License

MIT
