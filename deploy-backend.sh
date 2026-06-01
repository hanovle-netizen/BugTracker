#!/bin/bash
# Запускать на сервере (zolbrain@blacksalfetka)
set -euo pipefail

APP_DIR="/home/zolbrain/BugTracker"
BACKEND_DIR="$APP_DIR/backend"
COMPOSE_FILE="$BACKEND_DIR/docker/docker-compose.prod.yml"
ENV_FILE="$BACKEND_DIR/docker/.env"

log()  { echo "[$(date '+%H:%M:%S')] $*"; }
fail() { echo "ERROR: $*" >&2; exit 1; }

log "=== [1/4] Pulling latest code ==="
cd "$APP_DIR"
git pull --ff-only origin master

log "=== [2/4] Checking .env ==="
if [ ! -f "$ENV_FILE" ]; then
  fail ".env not found at $ENV_FILE — copy from .env.example and fill in secrets"
fi

log "=== [3/4] Rebuilding & restarting backend ==="
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" build app
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d

log "=== [4/4] Health check ==="
TRIES=0
until curl -sf "http://127.0.0.1:9191/api/healthz" >/dev/null 2>&1; do
  TRIES=$((TRIES + 1))
  if [ "$TRIES" -ge 30 ]; then
    log "Last backend logs:"
    docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" logs --tail=50 app
    fail "Backend did not become healthy after 60s"
  fi
  sleep 2
done

log ""
log "✓ Backend deployed and healthy at http://127.0.0.1:9191"
