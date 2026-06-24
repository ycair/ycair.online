#!/bin/bash
set -euo pipefail

echo "=== ycair.online Cloudflare Tunnel Setup ==="
echo ""
echo "This script sets up a Cloudflare Tunnel to expose"
echo "the signaling server at signal.ycair.online"
echo ""

if ! command -v cloudflared &>/dev/null; then
    echo "Installing cloudflared..."
    if [[ "$(uname -m)" == "arm64" ]]; then
        curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-amd64.tgz | tar xz -C /tmp
    else
        curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-amd64.tgz | tar xz -C /tmp
    fi
    sudo mv /tmp/cloudflared /usr/local/bin/cloudflared
    sudo chmod +x /usr/local/bin/cloudflared
    echo "cloudflared installed."
else
    echo "cloudflared already installed: $(cloudflared version)"
fi

echo ""
echo "Step 1: Login to Cloudflare"
echo "---------------------------"
cloudflared tunnel login

echo ""
echo "Step 2: Create tunnel"
echo "--------------------"
cloudflared tunnel create ycair-signal

echo ""
echo "Step 3: Configure DNS"
echo "---------------------"
cloudflared tunnel route dns ycair-signal signal.ycair.online

echo ""
echo "Step 4: Start tunnel"
echo "--------------------"
echo "Run this command to start the tunnel:"
echo ""
echo "  cloudflared tunnel run ycair-signal"
echo ""
echo "Or use the config file:"
echo ""
echo "  cloudflared tunnel --config config/cloudflared-config.yml run"
echo ""
echo "=== Setup Complete ==="
echo ""
echo "The signaling server will be available at:"
echo "  wss://signal.ycair.online/ws"
echo ""
echo "Update the Go core with:"
echo "  ycair-core <room> <pass> signal.ycair.online:443"
