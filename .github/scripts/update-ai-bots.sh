#!/usr/bin/env bash
set -euo pipefail

CADDY_SRC="https://raw.githubusercontent.com/ai-robots-txt/ai.robots.txt/main/Caddyfile"
ROBOTS_SRC="https://raw.githubusercontent.com/ai-robots-txt/ai.robots.txt/main/robots.txt"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "Fetching latest upstream AI bot lists..."
curl -fsSL "${CADDY_SRC}" -o "${TMP_DIR}/upstream_caddy"
curl -fsSL "${ROBOTS_SRC}" -o "${TMP_DIR}/upstream_robots"

echo "Updating ai-bots.caddy..."
{
	cat "${TMP_DIR}/upstream_caddy"
	printf "\nhandle @aibots {\n\tabort\n}\n"
} >ai-bots.caddy

echo "Updating internal/templates/robots.txt.tmpl..."
{
	cat "${TMP_DIR}/upstream_robots"
	printf "\n\nUser-agent: *\nDisallow: /*/chapter/*\nDisallow: /catalog\nDisallow: /history\nDisallow: /updates\nDisallow: /comments\n\nSitemap: {{.Domain}}/sitemap.xml\n"
} >internal/templates/robots.txt.tmpl

echo "AI bot lists updated successfully."
