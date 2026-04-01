"use client"

import { useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Wifi, WifiOff } from "lucide-react"

export function VPNConnect() {
  const [roomCode, setRoomCode] = useState("")
  const [password, setPassword] = useState("")
  const [status, setStatus] = useState<"disconnected" | "connecting" | "connected">("disconnected")
  const [localIP] = useState("192.168.1.100")

  const handleConnect = () => {
    if (!roomCode || !password) return
    
    setStatus("connecting")
    // Simulate connection
    setTimeout(() => {
      setStatus("connected")
    }, 2000)
  }

  const handleDisconnect = () => {
    setStatus("disconnected")
  }

  const isConnected = status === "connected"
  const isConnecting = status === "connecting"

  return (
    <div className="w-full max-w-md">
      {/* Logo */}
      <div className="text-center mb-12">
        <h1 className="text-3xl font-bold tracking-tight text-foreground">
          ycair<span className="text-primary">.online</span>
        </h1>
        <p className="text-muted-foreground text-sm mt-2">Secure Cross-Platform VPN</p>
      </div>

      {/* Connection Form */}
      <div className="space-y-4 mb-8">
        <div className="relative">
          <Input
            type="text"
            placeholder="Room Code"
            value={roomCode}
            onChange={(e) => setRoomCode(e.target.value)}
            disabled={isConnected || isConnecting}
            className="h-14 bg-input border-border/50 text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:ring-primary/20 transition-all"
          />
        </div>
        <div className="relative">
          <Input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={isConnected || isConnecting}
            className="h-14 bg-input border-border/50 text-foreground placeholder:text-muted-foreground focus:border-primary/50 focus:ring-primary/20 transition-all"
          />
        </div>
      </div>

      {/* Connect Button */}
      <Button
        onClick={isConnected ? handleDisconnect : handleConnect}
        disabled={isConnecting || (!isConnected && (!roomCode || !password))}
        className={`
          w-full h-16 text-lg font-semibold rounded-xl transition-all duration-300
          ${isConnected 
            ? "bg-destructive/20 text-destructive hover:bg-destructive/30 border border-destructive/30" 
            : "bg-primary text-primary-foreground hover:bg-primary/90 shadow-[0_0_30px_rgba(74,222,128,0.3)] hover:shadow-[0_0_40px_rgba(74,222,128,0.5)]"
          }
          ${isConnecting ? "animate-pulse" : ""}
        `}
      >
        {isConnecting ? (
          <span className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-current animate-ping" />
            Connecting...
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

      {/* Status Area */}
      <div className="mt-10 p-5 rounded-xl bg-card/50 border border-border/30">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <p className="text-xs uppercase tracking-wider text-muted-foreground mb-1">Local IP</p>
            <p className="text-sm font-mono text-foreground">{localIP}</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-wider text-muted-foreground mb-1">Status</p>
            <div className="flex items-center gap-2">
              <span 
                className={`h-2 w-2 rounded-full ${
                  isConnected 
                    ? "bg-primary animate-pulse" 
                    : isConnecting 
                      ? "bg-yellow-500 animate-pulse" 
                      : "bg-muted-foreground"
                }`} 
              />
              <span className={`text-sm font-medium ${
                isConnected 
                  ? "text-primary" 
                  : isConnecting 
                    ? "text-yellow-500" 
                    : "text-muted-foreground"
              }`}>
                {isConnecting ? "Connecting" : isConnected ? "Connected" : "Disconnected"}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Subtle branding */}
      <p className="text-center text-xs text-muted-foreground/50 mt-8">
        Encrypted peer-to-peer connection
      </p>
    </div>
  )
}
