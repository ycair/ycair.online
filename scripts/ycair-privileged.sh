#!/bin/bash
# macOS: requests admin privileges and runs ycair-core.
# Usage: ycair-privileged.sh <room> <pass> [signaling_addr]

ROOM="${1:?room required}"
PASS="${2:?password required}"
ADDR="${3:-localhost:9090}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CORE="${SCRIPT_DIR}/../src-tauri/bin/ycair-core-aarch64-apple-darwin"
LOGFILE="/tmp/ycair-core-${ROOM}.log"

[[ -f "$CORE" ]] || { echo "YCAR_ERROR:core binary not found" >&2; exit 1; }

pkill -f "ycair-core.*${ROOM}" 2>/dev/null || true
rm -f "$LOGFILE"

# osascript shows native macOS admin dialog.
# Shell backgrounds ycair-core and exits immediately.
osascript -e "
do shell script \"nohup '$CORE' '$ROOM' '$PASS' '$ADDR' > '$LOGFILE' 2>&1 &\"
with administrator privileges
" 2>/dev/null

for i in $(seq 1 50); do
    if [[ -f "$LOGFILE" ]] && [[ -s "$LOGFILE" ]]; then
        PID=$(pgrep -f "ycair-core.*${ROOM}" | head -1)
        echo "YCAR_PID:${PID:-0}"
        cat "$LOGFILE"
        exit 0
    fi
    sleep 0.2
done

echo "YCAR_ERROR:core did not start (admin auth cancelled?)" >&2
exit 1
