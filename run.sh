#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

BIN="$ROOT_DIR/socialpilot"

HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-8080}"

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  cat <<'USAGE'
Usage:
  ./run.sh
    Build and start web UI on 127.0.0.1:8080

  HOST=0.0.0.0 PORT=18080 ./run.sh
    Build and start web UI on custom host/port

  ./run.sh <socialpilot args...>
    Build and run socialpilot with provided args
    Example: ./run.sh contact add --name "林月" --gender female --tags "客户"
USAGE
  exit 0
fi

echo "[1/2] Building socialpilot..."
go build -o "$BIN" .

build_webui() {
  if [[ ! -d "$ROOT_DIR/webui" ]]; then
    return
  fi
  if ! command -v npm >/dev/null 2>&1; then
    echo "[WARN] npm not found, skip webui build."
    return
  fi
  echo "[2/3] Building webui..."
  (
    cd "$ROOT_DIR/webui"
    if [[ ! -d node_modules ]]; then
      npm install
    fi
    npm run build
  )
}

if [[ $# -eq 0 ]]; then
  build_webui
  echo "[3/3] Starting..."
  echo "Web UI: http://${HOST}:${PORT}"
  exec "$BIN" web --host "$HOST" --port "$PORT"
fi

if [[ "${1:-}" == "web" ]]; then
  build_webui
fi

echo "[2/2] Starting..."
exec "$BIN" "$@"
