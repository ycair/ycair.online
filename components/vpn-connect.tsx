"use client"

import { invoke } from "@tauri-apps/api/core"
import { useState, useEffect, useCallback } from "react"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Badge } from "@/components/ui/badge"
import { Wifi, WifiOff, Loader2, Shield, Lock, Users, Home, LogIn } from "lucide-react"

type ConnectionState = "disconnected" | "connecting" | "connected" | "error"

interface StatusPeer {
  id: string
  ip: string
}

interface CoreStatus {
  type: string
  assigned_ip: string
  peer_id: string
  peers: StatusPeer[]
  tun: string
  connected: boolean
}

export function VPNConnect() {
  const [connectionState, setConnectionState] = useState<ConnectionState>("disconnected")
  const [roomCode, setRoomCode] = useState("")
  const [password, setPassword] = useState("")
  const [progress, setProgress] = useState(0)
  const [errorMessage, setErrorMessage] = useState("")
  const [uptime, setUptime] = useState(0)
  const [vpnIP, setVpnIP] = useState("")
  const [peerList, setPeerList] = useState<StatusPeer[]>([])
  const [mode, setMode] = useState<"host" | "join">("host")
  const [signalingAddr, setSignalingAddr] = useState("")

  const handleConnect = useCallback(async () => {
    if (!roomCode.trim()) {
      setErrorMessage("Please enter a room code")
      setConnectionState("error")
      return
    }

    setErrorMessage("")
    setConnectionState("connecting")
    setProgress(0)

    const interval = setInterval(() => {
      setProgress((prev) => {
        if (prev >= 90) {
          clearInterval(interval)
          return 90
        }
        return prev + Math.random() * 15 + 5
      })
    }, 200)

    try {
      const msg = await invoke("start_connection", {
        mode: mode,
        room: roomCode,
        pass: password,
        signalingAddr: mode === "join" ? signalingAddr : null,
      })
      console.log("Backend response:", msg)

      clearInterval(interval)
      setProgress(100)
      setConnectionState("connected")
      setUptime(0)
    } catch (err) {
      console.error("Connection failed:", err)
      clearInterval(interval)
      setErrorMessage(typeof err === "string" ? err : "Connection failed. Please try again.")
      setConnectionState("error")
      setProgress(0)
    }
  }, [roomCode, password])

  const handleDisconnect = useCallback(async () => {
    setConnectionState("disconnected")
    setProgress(0)
    setUptime(0)

    try {
      await invoke("stop_connection")
    } catch {
      // stop_connection not yet implemented in Rust — non-fatal
    }
  }, [])

  useEffect(() => {
    if (connectionState !== "connected") return

    const interval = setInterval(() => {
      setUptime((prev) => prev + 1)
    }, 1000)

    return () => clearInterval(interval)
  }, [connectionState])

  useEffect(() => {
    if (connectionState !== "connected") return

    const poll = setInterval(async () => {
      try {
        const status = await invoke<CoreStatus | null>("get_status")
        if (status) {
          setVpnIP(status.assigned_ip)
          setPeerList(status.peers)
        }
      } catch {
      }
    }, 2000)

    return () => clearInterval(poll)
  }, [connectionState])

  const formatUptime = (seconds: number) => {
    const hrs = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60
    return `${hrs.toString().padStart(2, "0")}:${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`
  }

  const isInputDisabled = connectionState === "connecting" || connectionState === "connected"

  return (
    <div className="w-full max-w-[800px] min-h-[600px] flex flex-col bg-[oklch(0.12_0.005_260)] rounded-2xl border border-white/5 overflow-hidden shadow-2xl">
      <header className="flex items-center justify-between px-6 py-4 border-b border-white/5">
        <div className="flex items-center gap-3">
          <Shield className="size-5 text-[oklch(0.72_0.19_145)]" />
          <h1 className="text-lg font-semibold text-white tracking-tight">
            ycair<span className="text-[oklch(0.72_0.19_145)]">.online</span>
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-white/50">Secure Cross-Platform VPN</span>
          <StatusDot state={connectionState} />
        </div>
      </header>

      <main className="flex-1 flex flex-col items-center justify-center p-8 gap-6">
        <Card className="w-full max-w-md bg-[oklch(0.16_0.005_260)]/80 backdrop-blur-xl border-white/10 shadow-xl">
          <CardContent className="pt-6 space-y-5">
            <div className="space-y-4">
              <div className="flex gap-1 p-1 bg-white/5 rounded-lg">
                <button
                  onClick={() => setMode("host")}
                  disabled={isInputDisabled}
                  className={`flex-1 flex items-center justify-center gap-2 py-2 rounded-md text-sm font-medium transition-all ${
                    mode === "host"
                      ? "bg-[oklch(0.72_0.19_145)]/20 text-[oklch(0.72_0.19_145)]"
                      : "text-white/40 hover:text-white/70"
                  }`}
                >
                  <Home className="size-3.5" />
                  Host
                </button>
                <button
                  onClick={() => setMode("join")}
                  disabled={isInputDisabled}
                  className={`flex-1 flex items-center justify-center gap-2 py-2 rounded-md text-sm font-medium transition-all ${
                    mode === "join"
                      ? "bg-[oklch(0.72_0.19_145)]/20 text-[oklch(0.72_0.19_145)]"
                      : "text-white/40 hover:text-white/70"
                  }`}
                >
                  <LogIn className="size-3.5" />
                  Join
                </button>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-medium text-white/60 uppercase tracking-wider flex items-center gap-2">
                  <Lock className="size-3" />
                  Room Code
                </label>
                <Input
                  value={roomCode}
                  onChange={(e) => setRoomCode(e.target.value)}
                  placeholder="••••-••••-••••"
                  disabled={isInputDisabled}
                  className="font-mono text-center tracking-[0.3em] bg-[oklch(0.10_0.005_260)] border-white/10 text-white placeholder:text-white/20 h-12 text-lg focus-visible:border-[oklch(0.72_0.19_145)] focus-visible:ring-[oklch(0.72_0.19_145)]/20"
                />
              </div>

              <div className="space-y-2">
                <label className="text-xs font-medium text-white/60 uppercase tracking-wider flex items-center gap-2">
                  <Lock className="size-3" />
                  Password
                </label>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Enter password"
                  disabled={isInputDisabled}
                  className="bg-[oklch(0.10_0.005_260)] border-white/10 text-white placeholder:text-white/20 h-12 focus-visible:border-[oklch(0.72_0.19_145)] focus-visible:ring-[oklch(0.72_0.19_145)]/20"
                />
              </div>

              {mode === "join" && (
                <div className="space-y-2">
                  <label className="text-xs font-medium text-white/60 uppercase tracking-wider flex items-center gap-2">
                    <Wifi className="size-3" />
                    Signaling Server
                  </label>
                  <Input
                    value={signalingAddr}
                    onChange={(e) => setSignalingAddr(e.target.value)}
                    placeholder="host-ip:9090"
                    disabled={isInputDisabled}
                    className="bg-[oklch(0.10_0.005_260)] border-white/10 text-white placeholder:text-white/20 h-10 focus-visible:border-[oklch(0.72_0.19_145)] focus-visible:ring-[oklch(0.72_0.19_145)]/20"
                  />
                </div>
              )}
            </div>

            {connectionState === "connecting" && (
              <div className="space-y-3 animate-in fade-in duration-300">
                <Progress
                  value={Math.min(progress, 100)}
                  className="h-2 bg-white/10 [&>[data-slot=progress-indicator]]:bg-[oklch(0.72_0.19_145)]"
                />
                <div className="flex items-center justify-center gap-2 text-sm text-white/70">
                  <Loader2 className="size-4 animate-spin text-[oklch(0.72_0.19_145)]" />
                  <span>Vibe Checking... {Math.min(Math.round(progress), 100)}%</span>
                </div>
              </div>
            )}

            {connectionState === "error" && errorMessage && (
              <p className="text-sm text-[oklch(0.65_0.2_25)] text-center animate-in fade-in duration-300">
                {errorMessage}
              </p>
            )}

            {connectionState === "connected" ? (
              <Button
                onClick={handleDisconnect}
                className="w-full h-12 text-base font-medium bg-[oklch(0.45_0.2_25)] hover:bg-[oklch(0.5_0.22_25)] text-white border-0 transition-all duration-200"
              >
                <WifiOff className="size-5 mr-2" />
                Disconnect
              </Button>
            ) : (
              <Button
                onClick={handleConnect}
                disabled={connectionState === "connecting"}
                className="w-full h-12 text-base font-medium bg-[oklch(0.72_0.19_145)] hover:bg-[oklch(0.75_0.2_145)] text-[oklch(0.15_0.01_145)] border-0 transition-all duration-200 shadow-[0_0_20px_oklch(0.72_0.19_145_/_0.3)] hover:shadow-[0_0_30px_oklch(0.72_0.19_145_/_0.5)] disabled:shadow-none"
              >
                {connectionState === "connecting" ? (
                  <>
                    <Loader2 className="size-5 mr-2 animate-spin" />
                    Connecting...
                  </>
                ) : (
                  <>
                    <Wifi className="size-5 mr-2" />
                    Start Vibe
                  </>
                )}
              </Button>
            )}
          </CardContent>
        </Card>

        <Card className="w-full max-w-md bg-[oklch(0.16_0.005_260)]/80 backdrop-blur-xl border-white/10">
          <CardContent className="pt-6">
            <div className="grid grid-cols-3 gap-4 text-center">
              <div className="space-y-1">
                <p className="text-xs text-white/50 uppercase tracking-wider">VPN IP</p>
                <p className="font-mono text-sm text-[oklch(0.72_0.19_145)]">
                  {vpnIP || "--"}
                </p>
              </div>
              <div className="space-y-1">
                <p className="text-xs text-white/50 uppercase tracking-wider">Status</p>
                <div className="flex items-center justify-center gap-2">
                  <StatusDot state={connectionState} size="sm" />
                  <span className="text-sm text-white/90 capitalize">
                    {connectionState === "error" ? "disconnected" : connectionState}
                  </span>
                </div>
              </div>
              <div className="space-y-1">
                <p className="text-xs text-white/50 uppercase tracking-wider">Uptime</p>
                <p className="font-mono text-sm text-white/90">
                  {connectionState === "connected" ? formatUptime(uptime) : "--:--:--"}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {connectionState === "connected" && peerList.length > 0 && (
          <Card className="w-full max-w-md bg-[oklch(0.16_0.005_260)]/80 backdrop-blur-xl border-white/10">
            <CardContent className="pt-4 pb-3">
              <div className="flex items-center gap-2 mb-3">
                <Users className="size-3.5 text-white/50" />
                <p className="text-xs text-white/50 uppercase tracking-wider">
                  Peers ({peerList.length})
                </p>
              </div>
              <div className="space-y-1.5">
                {peerList.map((peer) => (
                  <div
                    key={peer.id}
                    className="flex items-center justify-between px-3 py-2 rounded-lg bg-white/5"
                  >
                    <span className="font-mono text-xs text-white/60 truncate max-w-[140px]">
                      {peer.id}
                    </span>
                    <span className="font-mono text-xs text-[oklch(0.72_0.19_145)]">
                      {peer.ip}
                    </span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}
      </main>

      <footer className="flex items-center justify-center py-4 border-t border-white/5">
        <Badge
          variant="outline"
          className={`text-xs border-white/10 bg-transparent transition-colors duration-300 ${
            connectionState === "connected"
              ? "text-[oklch(0.72_0.19_145)]"
              : "text-white/40"
          }`}
        >
          <span
            className={`size-1.5 rounded-full mr-2 transition-colors duration-300 ${
              connectionState === "connected"
                ? "bg-[oklch(0.72_0.19_145)]"
                : "bg-white/30"
            }`}
          />
          P2P Encrypted Tunnel {connectionState === "connected" ? "Active" : "Inactive"}
        </Badge>
      </footer>
    </div>
  )
}

function StatusDot({
  state,
  size = "md",
}: {
  state: ConnectionState
  size?: "sm" | "md"
}) {
  const sizeClass = size === "sm" ? "size-2" : "size-2.5"

  const getColorClass = () => {
    switch (state) {
      case "connected":
        return "bg-[oklch(0.72_0.19_145)]"
      case "connecting":
        return "bg-[oklch(0.8_0.15_85)]"
      case "error":
      case "disconnected":
      default:
        return "bg-white/40"
    }
  }

  return (
    <span className="relative flex">
      <span
        className={`${sizeClass} rounded-full ${getColorClass()} transition-colors duration-300`}
      />
      {state === "connecting" && (
        <span
          className={`absolute inset-0 ${sizeClass} rounded-full bg-[oklch(0.8_0.15_85)] animate-ping opacity-75`}
        />
      )}
      {state === "connected" && (
        <span
          className={`absolute inset-0 ${sizeClass} rounded-full bg-[oklch(0.72_0.19_145)] animate-pulse opacity-50`}
        />
      )}
    </span>
  )
}
