"use client"

import { invoke } from "@tauri-apps/api/core"
import { useState, useEffect, useCallback } from "react"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Badge } from "@/components/ui/badge"
import { Wifi, WifiOff, Loader2, Shield, Lock, Users, Home, LogIn, Copy, Globe } from "lucide-react"

type ConnectionState = "disconnected" | "connecting" | "connected" | "error"

interface StatusPeer { id: string; ip: string }
interface CoreStatus {
  type: string; assigned_ip: string; peer_id: string
  peers: StatusPeer[]; tun: string; connected: boolean; public_ip: string
}

export function VPNConnect() {
  const [connectionState, setConnectionState] = useState<ConnectionState>("disconnected")
  const [roomCode, setRoomCode] = useState("")
  const [password, setPassword] = useState("")
  const [progress, setProgress] = useState(0)
  const [errorMessage, setErrorMessage] = useState("")
  const [uptime, setUptime] = useState(0)
  const [vpnIP, setVpnIP] = useState("")
  const [publicIP, setPublicIP] = useState("")
  const [peerList, setPeerList] = useState<StatusPeer[]>([])
  const [mode, setMode] = useState<"host" | "join">("host")
  const [signalingAddr, setSignalingAddr] = useState("")
  const [tunnelUrl, setTunnelUrl] = useState("")

  const handleConnect = useCallback(async () => {
    if (!roomCode.trim()) { setErrorMessage("Please enter a room code"); setConnectionState("error"); return }
    setErrorMessage(""); setConnectionState("connecting"); setProgress(0)
    const interval = setInterval(() => { setProgress((p) => p >= 90 ? 90 : p + Math.random() * 15 + 5) }, 200)
    try {
      await invoke("start_connection", { mode, room: roomCode, pass: password, signalingAddr: mode === "join" ? signalingAddr : null })
      clearInterval(interval); setProgress(100); setConnectionState("connected"); setUptime(0)
    } catch (err) {
      clearInterval(interval); setErrorMessage(typeof err === "string" ? err : "Connection failed"); setConnectionState("error"); setProgress(0)
    }
  }, [roomCode, password, mode, signalingAddr])

  const handleDisconnect = useCallback(async () => { setConnectionState("disconnected"); setProgress(0); setUptime(0); try { await invoke("stop_connection") } catch { /* ok */ } }, [])

  useEffect(() => { if (connectionState !== "connected") return; const i = setInterval(() => setUptime((p) => p + 1), 1000); return () => clearInterval(i) }, [connectionState])

  useEffect(() => {
    if (connectionState !== "connected") return
    const poll = setInterval(async () => {
      try { const s = await invoke<CoreStatus | null>("get_status"); if (s) { setVpnIP(s.assigned_ip); setPeerList(s.peers); setPublicIP(s.public_ip) } } catch { /* ok */ }
    }, 2000)
    return () => clearInterval(poll)
  }, [connectionState])

  useEffect(() => {
    if (mode !== "host") return
    const poll = setInterval(async () => {
      try { const u = await invoke<string | null>("get_tunnel_url"); if (u) setTunnelUrl(u) } catch { /* ok */ }
    }, 2000)
    return () => clearInterval(poll)
  }, [mode])

  const formatUptime = (s: number) => `${String(Math.floor(s/3600)).padStart(2,"0")}:${String(Math.floor((s%3600)/60)).padStart(2,"0")}:${String(s%60).padStart(2,"0")}`
  const isInputDisabled = connectionState === "connecting" || connectionState === "connected"
  const signalDisplay = tunnelUrl || (publicIP ? `${publicIP.split(":")[0]}:9090` : "auto")

  return (
    <div className="w-full max-w-[800px] min-h-[600px] flex flex-col bg-[oklch(0.12_0.005_260)] rounded-2xl border border-white/5 overflow-hidden shadow-2xl">
      <header className="flex items-center justify-between px-6 py-4 border-b border-white/5">
        <div className="flex items-center gap-3">
          <Shield className="size-5 text-[oklch(0.72_0.19_145)]" />
          <h1 className="text-lg font-semibold text-white tracking-tight">ycair<span className="text-[oklch(0.72_0.19_145)]">.online</span></h1>
        </div>
        <div className="flex items-center gap-2"><span className="text-xs text-white/50">P2P VPN</span><StatusDot state={connectionState} /></div>
      </header>

      <main className="flex-1 flex flex-col items-center justify-center p-8 gap-6">
        <Card className="w-full max-w-md bg-[oklch(0.16_0.005_260)]/80 backdrop-blur-xl border-white/10 shadow-xl">
          <CardContent className="pt-6 space-y-5">
            <div className="flex gap-1 p-1 bg-white/5 rounded-lg">
              <button onClick={() => setMode("host")} disabled={isInputDisabled} className={`flex-1 flex items-center justify-center gap-2 py-2 rounded-md text-sm font-medium transition-all ${mode==="host"?"bg-[oklch(0.72_0.19_145)]/20 text-[oklch(0.72_0.19_145)]":"text-white/40 hover:text-white/70"}`}><Home className="size-3.5"/>Host</button>
              <button onClick={() => setMode("join")} disabled={isInputDisabled} className={`flex-1 flex items-center justify-center gap-2 py-2 rounded-md text-sm font-medium transition-all ${mode==="join"?"bg-[oklch(0.72_0.19_145)]/20 text-[oklch(0.72_0.19_145)]":"text-white/40 hover:text-white/70"}`}><LogIn className="size-3.5"/>Join</button>
            </div>

            <div className="space-y-2">
              <label className="text-xs font-medium text-white/60 uppercase tracking-wider flex items-center gap-2"><Lock className="size-3"/>Room Code</label>
              <Input value={roomCode} onChange={(e) => setRoomCode(e.target.value)} placeholder="••••-••••-••••" disabled={isInputDisabled} className="font-mono text-center tracking-[0.3em] bg-[oklch(0.10_0.005_260)] border-white/10 text-white placeholder:text-white/20 h-12 text-lg focus-visible:border-[oklch(0.72_0.19_145)]" />
            </div>
            <div className="space-y-2">
              <label className="text-xs font-medium text-white/60 uppercase tracking-wider flex items-center gap-2"><Lock className="size-3"/>Password</label>
              <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Enter password" disabled={isInputDisabled} className="bg-[oklch(0.10_0.005_260)] border-white/10 text-white placeholder:text-white/20 h-12 focus-visible:border-[oklch(0.72_0.19_145)]" />
            </div>

            {mode === "host" && connectionState !== "connected" && (
              <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-white/5 text-xs text-white/50">
                <Globe className="size-3 shrink-0" />
                <span>Signaling server starts automatically. Share room code + password with Join.</span>
              </div>
            )}

            {mode === "join" && (
              <div className="space-y-2">
                <label className="text-xs font-medium text-white/60 uppercase tracking-wider flex items-center gap-2"><Wifi className="size-3"/>Host Address</label>
                <Input value={signalingAddr} onChange={(e) => setSignalingAddr(e.target.value)} placeholder="host-ip:9090 or tunnel-url" disabled={isInputDisabled} className="bg-[oklch(0.10_0.005_260)] border-white/10 text-white placeholder:text-white/20 h-10 focus-visible:border-[oklch(0.72_0.19_145)]" />
              </div>
            )}

            {connectionState === "connecting" && (
              <div className="space-y-3">
                <Progress value={Math.min(progress,100)} className="h-2 bg-white/10 [&>[data-slot=progress-indicator]]:bg-[oklch(0.72_0.19_145)]" />
                <div className="flex items-center justify-center gap-2 text-sm text-white/70"><Loader2 className="size-4 animate-spin text-[oklch(0.72_0.19_145)]"/>Connecting... {Math.round(progress)}%</div>
              </div>
            )}
            {connectionState === "error" && errorMessage && <p className="text-sm text-red-400 text-center">{errorMessage}</p>}

            {connectionState === "connected" ? (
              <Button onClick={handleDisconnect} className="w-full h-12 text-base font-medium bg-red-600 hover:bg-red-700 text-white"><WifiOff className="size-5 mr-2"/>Disconnect</Button>
            ) : (
              <Button onClick={handleConnect} disabled={connectionState==="connecting"} className="w-full h-12 text-base font-medium bg-[oklch(0.72_0.19_145)] hover:bg-[oklch(0.75_0.2_145)] text-black"><Wifi className="size-5 mr-2"/>Start Vibe</Button>
            )}
          </CardContent>
        </Card>

        {connectionState === "connected" && (
          <Card className="w-full max-w-md bg-[oklch(0.16_0.005_260)]/80 backdrop-blur-xl border-white/10">
            <CardContent className="pt-6">
              <div className="grid grid-cols-3 gap-4 text-center">
                <div className="space-y-1"><p className="text-xs text-white/50 uppercase">VPN IP</p><p className="font-mono text-sm text-[oklch(0.72_0.19_145)]">{vpnIP||"--"}</p></div>
                <div className="space-y-1"><p className="text-xs text-white/50 uppercase">Status</p><div className="flex items-center justify-center gap-2"><StatusDot state={connectionState} size="sm"/><span className="text-sm text-white/90">connected</span></div></div>
                <div className="space-y-1"><p className="text-xs text-white/50 uppercase">Uptime</p><p className="font-mono text-sm text-white/90">{formatUptime(uptime)}</p></div>
              </div>
              {mode === "host" && (
                <div className="mt-4 pt-4 border-t border-white/5 space-y-2">
                  <p className="text-xs text-white/50 uppercase tracking-wider">Join Info</p>
                  <div className="flex items-center justify-between px-3 py-2 rounded-lg bg-white/5">
                    <span className="text-xs text-white/60">Signaling</span>
                    <span className="font-mono text-xs text-[oklch(0.72_0.19_145)]">{signalDisplay}</span>
                  </div>
                  <p className="text-[10px] text-white/30">Share the room code + password. Join enters the signaling address above.</p>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {connectionState === "connected" && peerList.length > 0 && (
          <Card className="w-full max-w-md bg-[oklch(0.16_0.005_260)]/80 backdrop-blur-xl border-white/10">
            <CardContent className="pt-4 pb-3">
              <div className="flex items-center gap-2 mb-3"><Users className="size-3.5 text-white/50"/><p className="text-xs text-white/50 uppercase">Peers ({peerList.length})</p></div>
              {peerList.map((p) => (
                <div key={p.id} className="flex items-center justify-between px-3 py-2 rounded-lg bg-white/5">
                  <span className="font-mono text-xs text-white/60 truncate max-w-[140px]">{p.id}</span>
                  <span className="font-mono text-xs text-[oklch(0.72_0.19_145)]">{p.ip}</span>
                </div>
              ))}
            </CardContent>
          </Card>
        )}
      </main>

      <footer className="flex items-center justify-center py-4 border-t border-white/5">
        <Badge variant="outline" className={`text-xs border-white/10 bg-transparent ${connectionState==="connected"?"text-[oklch(0.72_0.19_145)]":"text-white/40"}`}>
          <span className={`size-1.5 rounded-full mr-2 ${connectionState==="connected"?"bg-[oklch(0.72_0.19_145)]":"bg-white/30"}`}/>P2P Encrypted Tunnel {connectionState==="connected"?"Active":"Inactive"}
        </Badge>
      </footer>
    </div>
  )
}

function StatusDot({ state, size = "md" }: { state: ConnectionState; size?: "sm" | "md" }) {
  const sz = size === "sm" ? "size-2" : "size-2.5"
  const color = state === "connected" ? "bg-[oklch(0.72_0.19_145)]" : state === "connecting" ? "bg-yellow-400" : "bg-white/40"
  return <span className="relative flex"><span className={`${sz} rounded-full ${color}`}/>{state==="connecting"&&<span className={`absolute inset-0 ${sz} rounded-full bg-yellow-400 animate-ping opacity-75`}/>}{state==="connected"&&<span className={`absolute inset-0 ${sz} rounded-full bg-[oklch(0.72_0.19_145)] animate-pulse opacity-50`}/>}</span>
}
