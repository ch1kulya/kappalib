#!/usr/bin/env bash
set -euo pipefail

TARGET_DIR="/opt/kappalib"

if [ ! -d "${TARGET_DIR}" ]; then
	echo "::error::Directory ${TARGET_DIR} does not exist on VPS"
	exit 1
fi

cd "${TARGET_DIR}"

echo "Downloading GeoLite2-Country.mmdb..."
curl -fsSL -o GeoLite2-Country.mmdb.tmp https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-Country.mmdb
test -s GeoLite2-Country.mmdb.tmp
mv GeoLite2-Country.mmdb.tmp GeoLite2-Country.mmdb

echo "Reloading Caddy..."
docker compose exec -T caddy caddy reload --config /etc/caddy/Caddyfile || docker compose restart caddy

echo "GeoIP database updated successfully."
