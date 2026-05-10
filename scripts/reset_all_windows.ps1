$ErrorActionPreference = "Stop"
Write-Host "This deletes local Qdrant, Redis, Redpanda and Grafana volumes."
docker compose down -v
