# 跨平台優化計畫 / Cross-Platform Optimization Plan

## 目標 / Goal

macOS (Sequoia+) 與 Windows 11 雙平台完整支援，確保兩平台可互聯互通。

## 現狀分析 / Current State

| 項目 | macOS | Linux | Windows |
|------|-------|-------|---------|
| TUN 驅動 | ✅ water (utun) | ✅ water (/dev/tun) | ❌ stub only |
| IP 設定 | ✅ ifconfig + route | ✅ ip addr + ip link | ❌ netsh (not implemented) |
| 路由 | ✅ 10.99.0.0/24 | ✅ auto | ❌ none |
| 封包格式 | IPv4 raw | IPv4 raw | ❌ N/A |

## 技術決策 / Technical Decisions

### TUN 後端選擇 / TUN Backend Choice

- **macOS/Linux**: 保持 `songgao/water`（穩定、輕量、功能完整）
- **Windows**: 改用 `golang.zx2c4.com/wireguard/tun`（Wintun 驅動，原生 Layer 3 TUN）

理由：`water` 在 Windows 僅支援 TAP（Layer 2 乙太網框），需額外處理 MAC 標頭移除/添加，增加複雜度且與 macOS/Linux 行為不一致。`wireguard-go/tun` 提供真正的 Layer 3 TUN，與 macOS/Linux 的封包格式完全一致。

### 架構重構 / Architecture Refactor

```
core/tun/
├── tun.go           # 共用常數 (MTU) + 輔助函數
├── tun_darwin.go    # macOS: water + ifconfig/route
├── tun_linux.go     # Linux: water + ip addr/link
└── tun_windows.go   # Windows: wireguard-go/tun + netsh
```

每個平台檔案定義完整的 `Interface` struct 並匯出相同方法集。

## 實作步驟 / Implementation Steps

1. [ ] 重構 tun.go 為共用層（移除 water 依賴）
2. [ ] 更新 tun_darwin.go（保持 water，加入 build tag）
3. [ ] 更新 tun_linux.go（保持 water，加入 build tag）
4. [ ] 改寫 tun_windows.go（wireguard-go/tun + Wintun + netsh）
5. [ ] 更新 core/go.mod（加入 wireguard-go）
6. [ ] 更新 build-sidecars.sh（確保 Windows CGO 設定正確）
7. [ ] 測試跨平台互通性

## 互通性驗證清單 / Interoperability Checklist

- [ ] macOS ↔ macOS P2P 連線
- [ ] Windows ↔ Windows P2P 連線
- [ ] macOS ↔ Windows P2P 連線（關鍵）
- [ ] 加密通道協商正確
- [ ] IP 封包正確路由
- [ ] 信令伺服器跨平台註冊
