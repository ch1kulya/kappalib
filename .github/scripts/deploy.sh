#!/usr/bin/env bash
set -euo pipefail

TARGET_DIR="/opt/kappalib"

if [ ! -d "${TARGET_DIR}" ]; then
	echo "::error::Directory ${TARGET_DIR} does not exist on VPS"
	exit 1
fi

cd "${TARGET_DIR}"

echo "Updating repository to latest main..."
git fetch origin main
git reset --hard origin/main

echo "Pulling docker images..."
docker compose pull

if [ ! -f "GeoLite2-Country.mmdb" ] || [ -n "$(find GeoLite2-Country.mmdb -mtime +7 2>/dev/null)" ]; then
	echo "Downloading GeoLite2-Country.mmdb..."
	curl -fsSL -o GeoLite2-Country.mmdb.tmp https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-Country.mmdb
	mv GeoLite2-Country.mmdb.tmp GeoLite2-Country.mmdb
fi

echo "Starting containers..."
docker compose up -d --build --remove-orphans

echo "Waiting for all containers to become healthy..."
TIMEOUT=120
ELAPSED=0
INTERVAL=5

while [ $ELAPSED -lt $TIMEOUT ]; do
	PENDING=0
	CONTAINER_IDS=$(docker compose ps -q)

	if [ -z "$CONTAINER_IDS" ]; then
		echo "::error::No containers found running under docker compose"
		exit 1
	fi

	for cid in $CONTAINER_IDS; do
		STATUS=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$cid")
		NAME=$(docker inspect --format '{{.Name}}' "$cid" | sed 's/^\///')

		if [ "$STATUS" = "unhealthy" ] || [ "$STATUS" = "dead" ] || [ "$STATUS" = "exited" ]; then
			echo "::error::Container $NAME is in failed state: $STATUS"
			docker compose logs --tail=50 "$NAME"
			exit 1
		fi

		if [ "$STATUS" = "starting" ]; then
			PENDING=1
		fi
	done

	if [ $PENDING -eq 0 ]; then
		echo "All containers are healthy and running."
		docker compose ps
		break
	fi

	sleep $INTERVAL
	ELAPSED=$((ELAPSED + INTERVAL))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
	echo "::error::Timeout waiting for containers to become healthy after ${TIMEOUT}s"
	docker compose ps
	docker compose logs --tail=50
	exit 1
fi
