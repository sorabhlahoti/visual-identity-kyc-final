$ErrorActionPreference = "Stop"
Set-Location "$PSScriptRoot\..\api"
$env:QDRANT_URL = "http://localhost:6333"
$env:EMBEDDER_URL = "http://localhost:8001"
$env:METADATA_PATH = ".\data\metadata.json"
$env:EVENT_LOG_PATH = ".\data\events.log"
if (-not $env:HASH_PEPPER) { $env:HASH_PEPPER = "dev-change-this-long-random-secret" }
if (-not $env:JWT_SECRET) { $env:JWT_SECRET = "dev-jwt-secret-change-this" }
go run .\cmd\server
