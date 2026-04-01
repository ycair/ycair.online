"use client"

import { useState } from "react"
import { VPNConnect } from "@/components/vpn-connect"

export default function Home() {
  return (
    <main className="min-h-screen flex items-center justify-center bg-background p-4">
      <VPNConnect />
    </main>
  )
}
