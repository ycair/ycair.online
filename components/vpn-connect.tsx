"use client"

import { invoke } from "@tauri-apps/api/core"
import { useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Wifi, WifiOff, Loader2 } from "lucide-react"

export function VPNConnect() {
  // 狀態定義
  const [roomCode, setRoomCode] = useState("")
  const [password, setPassword] = useState("")
  const [status, setStatus] = useState<"disconnected" | "connecting" | "connected">("disconnected")
  const [progress, setProgress] = useState(0)
  const [localIP] = useState("192.168.1.100")

  // 處理連線邏輯
  const handleConnect = async () => {
    if (!roomCode || !password) return

    setStatus("connecting")
    setProgress(0)

    try {
      // 1. 呼叫 Rust 後端指令 (對應 src-tauri/src/lib.rs 中的 start_connection)
      const msg = await invoke("start_connection", {
        room: roomCode,
        pass: password,
      })
      console.log("Backend response:", msg)

      // 2. 模擬進度條跑動動畫
      const interval = setInterval(() => {
        setProgress((prev) => {
          if (prev >= 100) {
            clearInterval(interval)
            setStatus("connected")
            return 100
          }
          return prev + 20
        })
      }, 150)
    } catch (err) {
      console.error("連線過程發生錯誤:", err)
      setStatus("disconnected")
      setProgress(0)
    }
  }

  // 處理斷線邏輯
  const handleDisconnect = () => {
    setStatus("disconnected")
    setProgress(0)
  }

  const isConnected = status === "connected"
  const isConnecting = status === "connecting"

  return (
    <div className="w-full max-w-md">
      {/* Logo 區域 */}
      <div className="text-center mb-12">
        <h1 className="text-3xl font-bold tracking-tight text-foreground">
          ycair<span className="text-primary">.online</span>
        </h1>
        <p className="text-muted-foreground text-sm mt-2 font-medium tracking-wide">
          Secure Cross-Platform VPN
        </p>
      </div>

      {/* 輸入表單 */}
      <div className="space-y-4 mb-8">
        <div className="relative">
          <Input
            type="text"
            placeholder="Room Code"
            value={roomCode}
            onChange={(e) => setRoomCode(e.target.value)}
            disabled={isConnected || isConnecting}
            className="h-14 bg-card/40 border-border/50 text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:ring-primary/20 transition-all rounded-xl"
          />
        </div>
        <div className="relative">
          <Input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={isConnected || isConnecting}
            className="h-14 bg-card/40 border-border/50 text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:ring-primary/20 transition-all rounded-xl"
          />
        </div>
      </div>

      {/* 連線按鈕 */}
      <Button
        onClick={isConnected ? handleDisconnect : handleConnect}
        disabled={isConnecting || (!isConnected && (!roomCode || !password))}
        className={`
          w-full h-16 text-lg font-bold rounded-2xl transition-all duration-500
          ${
            isConnected
              ? "bg-destructive/10 text-destructive hover:bg-destructive/20 border border-destructive/20 shadow-none"
              : "bg-primary text-primary-foreground hover:bg-primary/90 shadow-[0_0_25px_rgba(74,222,128,0.2)] hover:shadow-[0_0_35px_rgba(74,222,128,0.4)]"
          }
        `}
      >
        {isConnecting ? (
          <span className="flex items-center gap-3">
            <Loader2 className="h-5 w-5 animate-spin" />
            Vibe Checking... {progress}%
          </span>
        ) : isConnected ? (
          <span className="flex items-center gap-2">
            <WifiOff className="h-5 w-5" />
            Disconnect
          </span>
        ) : (
          <span className="flex items-center gap-2">
            <Wifi className="h-5 w-5" />
            Start Vibe
          </span>
        )}
      </Button>

      {/* 狀態資訊區塊 */}
      <div className="mt-10 p-6 rounded-2xl bg-card/30 border border-border/20 backdrop-blur-sm">
        <div className="grid grid-cols-2 gap-6">
          <div>
            <p className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground/70 mb-2">Local IP</p>
            <p className="text-sm font-mono text-foreground/90">{localIP}</p>
          </div>
          <div>
            <p className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground/70 mb-2">Network Status</p>
            <div className="flex items-center gap-2">
              <span
                className={`h-2 w-2 rounded-full transition-all duration-500 ${
                  isConnected
                    ? "bg-primary shadow-[0_0_8px_#4ade80]"
                    : isConnecting
                    ? "bg-yellow-400 animate-pulse"
                    : "bg-muted-foreground/40"
                }`}
              />
              <span
                className={`text-sm font-semibold tracking-tight ${
                  isConnected ? "text-primary" : isConnecting ? "text-yellow-400" : "text-muted-foreground/60"
                }`}
              >
                {isConnecting ? "Connecting" : isConnected ? "Connected" : "Disconnected"}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* 底部說明 */}
      <div className="flex flex-col items-center gap-2 mt-10">
        <div className="flex items-center gap-2 px-3 py-1 rounded-full bg-muted/30 border border-border/20">
          <div className="h-1 w-1 rounded-full bg-primary" />
          <p className="text-[10px] text-muted-foreground/60 font-medium uppercase tracking-tighter">
            P2P Encrypted Tunnel Active
          </p>
        </div>
      </div>
    </div>
  )
}