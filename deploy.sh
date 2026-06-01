#!/bin/bash
# Запускать на ФРОНТЕНД-сервере
set -euo pipefail

# ──────────────────────────────────────────────
# Config
# ──────────────────────────────────────────────
DOMAIN="bug-tracker.sytes.net"
APP_DIR="/home/zolbrain/BugTracker"
WWW_DIR="/var/www/bugtracker"
FRONTEND_URL="https://bug-tracker.sytes.net"
BACKEND_HEALTHCHECK="https://bug-tracker.sytes.net/api/healthz"

# ──────────────────────────────────────────────
log() { echo "[$(date '+%H:%M:%S')] $*"; }
fail() { echo "ERROR: $*" >&2; exit 1; }

# ──────────────────────────────────────────────
# 1. Preflight
# ──────────────────────────────────────────────
log "=== [1/5] Preflight checks ==="
command -v git     >/dev/null 2>&1 || fail "git not found"
command -v npm     >/dev/null 2>&1 || fail "npm not found"
command -v nginx   >/dev/null 2>&1 || fail "nginx not found"
command -v certbot >/dev/null 2>&1 || fail "certbot not found (apt install certbot python3-certbot-nginx)"
[ -d "$APP_DIR" ]                   || fail "APP_DIR not found: $APP_DIR"

# SSL: получаем сертификат если его ещё нет
CERT_PATH="/etc/letsencrypt/live/$DOMAIN/fullchain.pem"
if [ ! -f "$CERT_PATH" ]; then
  log "SSL cert not found — requesting via certbot (nginx must be running with HTTP)"

  # Ставим временный HTTP-конфиг чтобы certbot мог пройти ACME-challenge
  cat > /etc/nginx/sites-available/bugtracker.conf <<EOF
server {
    listen 80;
    server_name $DOMAIN;
    root $WWW_DIR;
    location /.well-known/acme-challenge/ { root /var/www/html; }
    location / { return 444; }
}
EOF
  mkdir -p "$WWW_DIR"
  if [ ! -e /etc/nginx/sites-enabled/bugtracker.conf ]; then
    ln -s /etc/nginx/sites-available/bugtracker.conf \
          /etc/nginx/sites-enabled/bugtracker.conf
  fi
  nginx -t && systemctl reload nginx

  certbot certonly --webroot -w /var/www/html -d "$DOMAIN" --non-interactive --agree-tos \
    || fail "certbot failed — убедись что DNS $DOMAIN → этот сервер, и порт 80 открыт"
  log "SSL cert obtained."
else
  log "SSL cert found: $CERT_PATH"
fi

# Проверяем что бэкенд живой (предупреждение, не блокируем деплой)
log "Checking backend at $BACKEND_HEALTHCHECK ..."
if curl -sf "$BACKEND_HEALTHCHECK" >/dev/null 2>&1; then
  log "Backend is healthy."
else
  log "WARNING: Backend not responding at $BACKEND_HEALTHCHECK — deploying frontend anyway"
fi

# ──────────────────────────────────────────────
# 2. Pull latest code
# ──────────────────────────────────────────────
log "=== [2/5] Pulling latest code ==="
cd "$APP_DIR"
CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
log "Branch: $CURRENT_BRANCH"
git fetch origin "$CURRENT_BRANCH"
git pull --ff-only origin "$CURRENT_BRANCH"

# ──────────────────────────────────────────────
# 3. Build frontend
# ──────────────────────────────────────────────
log "=== [3/5] Building frontend ==="
cd "$APP_DIR/TaskTrackerFrontend"
npm ci --prefer-offline
VITE_API_URL="$FRONTEND_URL/api" npm run build

# ──────────────────────────────────────────────
# 4. Deploy static files (атомарная замена)
# ──────────────────────────────────────────────
log "=== [4/5] Deploying static files ==="
TEMP_DIR="$(mktemp -d)"
cp -r "$APP_DIR/TaskTrackerFrontend/dist/." "$TEMP_DIR/"
chmod -R 755 "$TEMP_DIR"
find "$TEMP_DIR" -type f -exec chmod 644 {} \;
rm -rf "$WWW_DIR"
mv "$TEMP_DIR" "$WWW_DIR"
log "Static files deployed to $WWW_DIR"

# ──────────────────────────────────────────────
# 5. Nginx
# ──────────────────────────────────────────────
log "=== [5/5] Updating nginx config ==="
cp "$APP_DIR/nginx/bugtracker.conf" /etc/nginx/sites-available/bugtracker

if [ ! -e /etc/nginx/sites-enabled/bugtracker ]; then
  ln -s /etc/nginx/sites-available/bugtracker \
        /etc/nginx/sites-enabled/bugtracker
  log "Created symlink in sites-enabled (first deploy)"
fi

nginx -t || fail "nginx config test failed — other sites unaffected, fix the config and retry"
systemctl reload nginx

# Авто-обновление сертификата (certbot renew + nginx reload)
# Добавляем в cron только если записи ещё нет
CRON_JOB="0 3 * * * certbot renew --quiet --deploy-hook 'systemctl reload nginx'"
crontab -l 2>/dev/null | grep -qF "certbot renew" \
  || (crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -
log "Certbot auto-renew cron: OK"

log ""
log "✓ Frontend deployed: https://$DOMAIN"
