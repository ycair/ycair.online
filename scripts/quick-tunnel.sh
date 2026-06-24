#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$PROJECT_ROOT/src-tauri/bin"

echo "=== ycair.online Quick Tunnel ==="
echo ""

if ! command -v cloudflared &>/dev/null; then
    echo "cloudflared not installed."
    echo "Install: brew install cloudflared"
    exit 1
fi

SIGNAL_BIN="$BIN_DIR/signaling-server"
if [[ ! -f "$SIGNAL_BIN" ]]; then
    cd "$PROJECT_ROOT/signaling-server" && go build -o "$SIGNAL_BIN" .
fi

echo "Starting signaling server on localhost:9090..."
"$SIGNAL_BIN" -port 9090 &
SIGNAL_PID=$!
sleep 1

echo "Starting Cloudflare quick tunnel..."
echo ""
cloudflared tunnel --url http://localhost:9090 2>&1 | while read line; do
    echo "$line"
    if echo "$line" | grep -q "trycloudflare.com"; then
        TUNNEL_URL=$(echo "$line" | grep -o 'https://[^ ]*trycloudflare\.com')
        echo ""
        echo "=== Tunnel Ready ==="
        echo "Signaling server available at:"
        echo "  wss://${TUNNEL_URL#https://}/ws"
        echo ""
        echo "Use with ycair-core:"
        echo "  ycair-core <room> <pass> ${TUNNEL_URL#https://}"
    fi
done &
TUNNEL_PID=$!

trap "kill $SIGNAL_PID $TUNNEL_PID 2>/dev/null; exit" INT TERM
wait
