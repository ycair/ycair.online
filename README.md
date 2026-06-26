# ycair.online

> 跨平台 P2P 加密網狀 VPN — 輸入房號密碼即連線
> Cross-platform P2P encrypted mesh VPN — room code + password, done.

---

## 📥 下載 / Download

| 平台 | 檔案 | 說明 |
|------|------|------|
| macOS | [ycair.online_0.1.0_aarch64.dmg](https://github.com/ycair/ycair.online/releases/latest) | 桌面應用 |
| Windows | [ycair-core-x86_64-pc-windows-msvc.exe](https://github.com/ycair/ycair.online/releases/latest) | CLI（桌面應用需在 Windows 建置） |
| Linux | [ycair-core-x86_64-unknown-linux-gnu](https://github.com/ycair/ycair.online/releases/latest) | CLI |

## 🚀 使用方式（macOS 桌面應用）

```
1. 下載 .dmg → 拖入 Applications → 開啟
2. 輸入 Room Code（例如 "mc-world"）
3. 輸入 Password
4. 點 Connect
5. macOS 彈出管理員密碼 → 輸入允許
6. VPN IP 顯示 → 完成
```

**就是這樣。** 沒有 Host/Join、沒有信令伺服器設定、沒有 port forwarding。全部自動處理。

### 🎮 Minecraft 互聯

同房間內的所有人：

| 誰 | 做什麼 |
|----|--------|
| 開世界的人 | Minecraft → Singleplayer → Esc → Open to LAN |
| 其他人 | Multiplayer → Direct Connect → `開世界那人的 VPN IP:25565` |

VPN IP 在 app 的 UI 上直接顯示（例如 `10.99.0.2`）。

| 版本 | 協定 | Direct Connect 格式 |
|------|------|---------------------|
| Java Edition | TCP | `10.99.0.2`（port 25565 預設） |
| Bedrock Edition | UDP | `10.99.0.2:19132` |

## 🖥 Windows / Linux CLI

```bash
# macOS/Linux
./ycair-core-x86_64-unknown-linux-gnu <room> <password> signal.ycair.space

# Windows (系統管理員執行)
ycair-core-x86_64-pc-windows-msvc.exe <room> <password> signal.ycair.space
```

## 🏗 架構

```
你 ──WSS──→ signal.ycair.space ──→ 其他人
                (信令伺服器)
                    │
              交換 peer 資訊
                    │
          你 ←──UDP P2P──→ 其他人
         (Noise NN + ChaChaPoly 加密)
```

- **信令**：公網 `signal.ycair.space`，WebSocket Secure (WSS)
- **NAT 穿透**：Google STUN + UDP Hole Punching
- **加密**：Noise NN → ChaChaPoly AEAD（端對端）
- **虛擬網卡**：TUN，`10.99.0.0/24` 子網

## 🔐 安全性

- 房間密碼 SHA256，伺服器不存明文
- P2P 傳輸 Noise NN + ChaChaPoly 端對端加密
- 信令通道 WSS 加密（Cloudflare Tunnel）
- 信令伺服器只做 peer 介紹，無法解密傳輸內容

## 📋 支援協定

只要是 IPv4，全部透傳：TCP（Minecraft Java、HTTP、SSH）、UDP（Minecraft Bedrock、DNS）、ICMP（ping）。

## 📁 專案結構

```
├── app/                  # Next.js 前端
├── components/           # React 元件
├── core/                 # Go VPN 引擎（crypto/mesh/p2p/signaling/tun）
├── signaling-server/     # Go 信令伺服器
├── src-tauri/            # Tauri 桌面應用 (Rust)
└── scripts/              # 建置腳本
```

## 🔧 從原始碼建置

```bash
git clone https://github.com/ycair/ycair.online.git
cd ycair.online && npm install
bash scripts/build-sidecars.sh
npm run tauri build
```
