#!/bin/sh
set -e

LOCK_FILE="/root/setup_done.lock"

if [ ! -f "$LOCK_FILE" ]; then
  go run ./seed/seeder.go
  go run ./data-updater/updater.go
  go run ./embed/embedder.go
  touch "$LOCK_FILE"
fi