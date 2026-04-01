#!/usr/bin/env sh

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

exec docker compose -f "$SCRIPT_DIR/docker-compose.yml" "$@"
