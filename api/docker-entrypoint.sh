#!/bin/sh
set -eu
mkdir -p /data
chown -R appuser:appuser /data 2>/dev/null || true
exec su-exec appuser /app/app
