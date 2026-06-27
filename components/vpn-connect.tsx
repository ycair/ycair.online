"use client"

import { invoke } from "@tauri-apps/api/core"
import { useState, useEffect, useCallback } from "react"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Badge } from "@/components/ui/badge"
import { Wifi, WifiOff, Loader2, Shield, Lock, Users } from "lucide-react"

type ConnectionState = "disconnected" | "connecting" | "connected" | "error"

interface StatusPeer { id: string; ip: string }
interface CoreStatus {
  type: string; assigned_ip: string; peer_id: string
  peers: StatusPeer[]; tun: string; tun_error: string; connected: boolean; public_ip: string
}

export function VPNConnect() {
  const [connectionState, setConnectionState] = useState<ConnectionState>("disconnected")
  const [roomCode, setRoomCode] = useState("")
  const [password, setPassword] = useState("")
  const [progress, setProgress] = useState(0)
  const [errorMessage, setErrorMessage] = useState("")
  const [uptime, setUptime] = useState(0)
  const [vpnIP, setVpnIP] = useState("")
  const [tunName, setTunName] = useState("")
  const [tunError, setTunError] = useState("")
  const [peerList, setPeerList] = useState<StatusPeer[]>([])

  const handleConnect = useCallback(async () => {
    if (!roomCode.trim() || !password.trim()) { setErrorMessage("Enter room code and password"); setConnectionState("error"); return }
    setErrorMessage(""); setConnectionState("connecting"); setProgress(0)
    const interval = setInterval(() => { setProgress((p) => p >= 90 ? 90 : p + Math.random() * 15 + 5) }, 200)
    try {
      await invoke("start_connection", { room: roomCode, pass: password })
      clearInterval(interval); setProgress(100); setConnectionState("connected"); setUptime(0)
    } catch (err) {
      clearInterval(interval); setErrorMessage(typeof err === "string" ? err : "Connection failed"); setConnectionState("error"); setProgress(0)
    }
  }, [roomCode, password])

  const handleDisconnect = useCallback(async () => { setConnectionState("disconnected"); setProgress(0); setUptime(0); try { await invoke("stop_connection") } catch { /* ok */ } }, [])

  useEffect(() => { if (connectionState !== "connected") return; const i = setInterval(() => setUptime((p) => p + 1), 1000); return () => clearInterval(i) }, [connectionState])

  useEffect(() => {
    if (connectionState !== "connected") return
    const poll = setInterval(async () => {
      try { const s = await invoke<CoreStatus | null>("get_status"); if (s) { setVpnIP(s.assigned_ip); setTunName(s.tun); setTunError(s.tun_error || ""); setPeerList(s.peers) } } catch { /* ok */ }
    }, 2000)
    return () => clearInterval(poll)
  }, [connectionState])

  const formatUptime = (s: number) => `${String(Math.floor(s/3600)).padStart(2,"0")}:${String(Math.floor((s%3600)/60)).padStart(2,"0")}:${String(s%60).padStart(2,"0")}`
  const isInputDisabled = connectionState === "connecting" || connectionState === "connected"

  return (
    <div className="w-full max-w-[800px] min-h-[600px] flex flex-col bg-[oklch(0.12_0.005_260)] rounded-2xl border border-white/5 overflow-hidden shadow-2xl">
      <header className="flex items-center justify-between px-6 py-4 border-b border-white/5">
        <div className="flex items-center gap-3"><Shield className="size-5 text-[oklch(0.72_0.19_145)]"/><h1 className="text-lg font-semibold text-white tracking-tight">ycair<span className="text-[oklch(0.72_0.19_145)]">.online</span></h1></div>
        <div className="flex items-center gap-2"><span className="text-xs text-white/50">P2P VPN</span><StatusDot state={connectionState}/></div>
      </header>

      <main className="flex-1 flex flex-col items-center justify-center p-8 gap-6">
        <Card className="w-full max-w-md bg-[oklch(0.16_0.005_260)]/80 backdrop-blur-xl border-white/10 shadow-xl">
          <CardContent className="pt-6 space-y-5">
            <div className="space-y-2">
              <label className="text-xs font-medium text-white/60 uppercase tracking-wider flex items-center gap-2"><Lock className="size-3"/>Room Code</label>
              <Input value={roomCode} onChange={(e) => setRoomCode(e.target.value)} placeholder="any-name" disabled={isInputDisabled} className="font-mono text-center tracking-[0.3em] bg-[oklch(0.10_0.005_260)] border-white/10 text-white placeholder:text-white/20 h-12 text-lg focus-visible:border-[oklch(0.72_0.19_145)]"/>
            </div>
            <div className="space-y-2">
              <label className="text-xs font-medium text-white/60 uppercase tracking-wider flex items-center gap-2"><Lock className="size-3"/>Password</label>
              <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Enter password" disabled={isInputDisabled} className="bg-[oklch(0.10_0.005_260)] border-white/10 text-white placeholder:text-white/20 h-12 focus-visible:border-[oklch(0.72_0.19_145)]"/>
            </div>

            {connectionState === "connecting" && (
              <div className="space-y-3">
                <Progress value={Math.min(progress,100)} className="h-2 bg-white/10 [&>[data-slot=progress-indicator]]:bg-[oklch(0.72_0.19_145)]"/>
                <div className="flex items-center justify-center gap-2 text-sm text-white/70"><Loader2 className="size-4 animate-spin text-[oklch(0.72_0.19_145)]"/>Connecting via signal.ycair.space...</div>
              </div>
            )}
            {connectionState === "error" && <p className="text-sm text-red-400 text-center">{errorMessage}</p>}

            {connectionState === "connected" ? (
              <Button onClick={handleDisconnect} className="w-full h-12 text-base font-medium bg-red-600 hover:bg-red-700 text-white"><WifiOff className="size-5 mr-2"/>Disconnect</Button>
            ) : (
              <Button onClick={handleConnect} disabled={connectionState==="connecting"} className="w-full h-12 text-base font-medium bg-[oklch(0.72_0.19_145)] hover:bg-[oklch(0.75_0.2_145)] text-black"><Wifi className="size-5 mr-2"/>Connect</Button>
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
              {tunName ? (
                <div className="mt-3 pt-3 border-t border-white/5 flex items-center justify-center gap-2 text-xs text-green-400">
                  <span className="size-1.5 rounded-full bg-green-400"/>Adapter: {tunName}
                </div>
              ) : (
                <div className="mt-3 pt-3 border-t border-white/5 text-xs text-yellow-400 text-center">
                  <span className="size-1.5 rounded-full bg-yellow-400 inline-block mr-1"/>VPN adapter not created — Run as Administrator
                  {tunError && <p className="mt-1 text-white/40 font-mono text-[10px] break-all">{tunError}</p>}
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {connectionState === "connected" && peerList.length > 0 && (
          <Card className="w-full max-w-md bg-[oklch(0.16_0.005_260)]/80 backdrop-blur-xl border-white/10">
            <CardContent className="pt-4 pb-3">
              <div className="flex items-center gap-2 mb-3"><Users className="size-3.5 text-white/50"/><p className="text-xs text-white/50 uppercase">Peers ({peerList.length})</p></div>
              {peerList.map((p) => (<div key={p.id} className="flex items-center justify-between px-3 py-2 rounded-lg bg-white/5"><span className="font-mono text-xs text-white/60 truncate max-w-[140px]">{p.id}</span><span className="font-mono text-xs text-[oklch(0.72_0.19_145)]">{p.ip}</span></div>))}
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
