#!/bin/sh

curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
rm get-docker.sh

APP_DIR="/opt/kappalib"
mkdir -p "$APP_DIR"
git clone --depth 1 https://github.com/ch1kulya/kappalib.git "$APP_DIR"
