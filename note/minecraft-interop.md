# Minecraft 跨平台互通性驗證 / Cross-Platform Minecraft Interoperability

## 協定分析 / Protocol Analysis

### Minecraft Java Edition (TCP)

| 項目 | 說明 |
|------|------|
| 傳輸層 | TCP (協議號 6) |
| 預設埠 | 25565 |
| 連線模式 | Client-Server，TCP 三向握手 |
| LAN 發現 | UDP multicast 224.0.2.60:4445 |

### Minecraft Bedrock Edition (UDP)

| 項目 | 說明 |
|------|------|
| 傳輸層 | UDP (協議號 17)，上層 RakNet |
| 預設埠 | 19132 |
| 連線模式 | Client-Server over RakNet |
| LAN 發現 | UDP broadcast |

## Mesh 層 TCP/UDP 穿透驗證 / Mesh TCP/UDP Penetration Verification

### 封包路徑 / Packet Path

```
Player A (10.99.0.2, Host)                    Player B (10.99.0.3, Client)
────────────────────────────────────────────────────────────────────────
Minecraft listens 0.0.0.0:25565               Minecraft connects to 10.99.0.2:25565
         │                                              │
         │                                      kernel: TCP SYN → TUN
         │                                      dst=10.99.0.2, proto=TCP(6)
         │                                              │
         │                                      mesh.forwardLoop()
         │                                      extractDstIP → 10.99.0.2
         │                                      lookupPeer → peer A
         │                                      encrypt → UDP → A
         │                                              │
 kernel ← TUN ← decrypt ← UDP                 mesh.receiveLoop() ← UDP
 dst=10.99.0.2, proto=TCP                           │
         │                                      decrypt → plaintext
 kernel: TCP SYN-ACK → TUN                            │
 dst=10.99.0.3                                   kernel ← TUN
         │                                              │
 mesh.forwardLoop()                               ... ACK ...
 encrypt → UDP → B                                        │
         │                                      ✅ TCP connection established
    ... data flow ...
```

### 關鍵程式碼路徑 / Critical Code Paths

1. **`extractDstIP()`** (`mesh/mesh.go:144-156`): 只檢查 IPv4 版本號 (`version != 4`)，不檢查協定欄位。TCP/UDP/ICMP 全部透傳。

2. **`forwardLoop()`** (`mesh/mesh.go:41-73`): 從 TUN 讀取 → 查路由 → 加密 → UDP 發送。阻塞讀取，無封包丟失。

3. **`receiveLoop()`** (`mesh/mesh.go:75-111`): UDP 接收 → 解密 → TUN 寫入。200ms 超時輪詢。

4. **Bidirectional**: forwardLoop 和 receiveLoop 在獨立 goroutine 中並行運行 → 全雙工。

### 已確認限制 / Confirmed Limitations

| 限制 | 影響 | 解決方案 |
|------|------|----------|
| Multicast/Broadcast 不穿透 | LAN 發現無效 | Direct Connect 輸入 VPN IP |
| TCP over UDP 可能亂序 | 極低延遲環境下無影響 | TCP 自身重排序 |
| 加密開銷 ~32 bytes | 有效 MTU = 1468 | Path MTU Discovery 自動調整 |

## 使用步驟 / Usage Steps

### 主機端 (Host)

1. 啟動 ycair.online，建立/加入房間
2. 記下 VPN IP（見 UI 面板的 "VPN IP"）
3. 開啟 Minecraft → Singleplayer → Open to LAN
4. 遊戲會監聽 0.0.0.0:25565 (Java) 或 0.0.0.0:19132 (Bedrock)
5. 告知客戶端你的 VPN IP

### 客戶端 (Client)

1. 啟動 ycair.online，加入同一房間
2. Java Edition: Multiplayer → Direct Connect → 輸入 `主機VPN IP:25565`
3. Bedrock Edition: Friends → Add Server → 輸入主機 VPN IP，Port 19132
4. 連線成功後即可共同遊玩

### 跨平台驗證矩陣

| Host \ Client | macOS Java | macOS Bedrock | Windows Java | Windows Bedrock |
|---------------|------------|---------------|--------------|-----------------|
| macOS Java | ✅ TCP | ❌ 不同版 | ✅ TCP | ❌ 不同版 |
| macOS Bedrock | ❌ 不同版 | ✅ UDP | ❌ 不同版 | ✅ UDP |
| Windows Java | ✅ TCP | ❌ 不同版 | ✅ TCP | ❌ 不同版 |
| Windows Bedrock | ❌ 不同版 | ✅ UDP | ❌ 不同版 | ✅ UDP |

> 注意：Java Edition 與 Bedrock Edition 之間**無法互聯**（不同遊戲引擎）。但 macOS Java ↔ Windows Java 和 macOS Bedrock ↔ Windows Bedrock 均可透過 ycair 互聯。
