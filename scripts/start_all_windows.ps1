$ErrorActionPreference = "Stop"
Write-Host "Starting Visual KYC final stack with Docker Compose..."
docker compose up -d --build
Write-Host "Services: API http://localhost:8080 | Inference http://localhost:8001 | Qdrant http://localhost:6333 | Kafka REST http://localhost:18082 or http://localhost:8082 | Grafana http://localhost:3000"
Write-Host "Tip: run 'docker compose ps' until kyc-api and kyc-worker are Up. Redpanda may take 30-90 seconds on Windows."
