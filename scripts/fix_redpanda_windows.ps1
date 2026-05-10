$ErrorActionPreference = "Continue"
Write-Host "Stopping old containers..."
docker compose down --remove-orphans

Write-Host "Removing old Redpanda containers if they exist..."
docker rm -f kyc-redpanda kyc-redpanda-init 2>$null

Write-Host "Removing old Redpanda volumes that may contain stale broker metadata..."
$vols = docker volume ls --format "{{.Name}}" | Select-String "redpanda"
foreach ($v in $vols) {
  $name = $v.ToString()
  Write-Host "Removing volume $name"
  docker volume rm $name 2>$null
}

Write-Host "Pulling Redpanda image..."
docker pull docker.redpanda.com/redpandadata/redpanda:v26.1.6

Write-Host "Starting Redpanda only..."
docker compose up -d redpanda

Write-Host "Waiting for Redpanda Kafka API on 9092..."
$ready = $false
for ($i = 1; $i -le 90; $i++) {
  docker compose exec -T redpanda rpk topic list -X brokers=redpanda:9092 *> $null
  if ($LASTEXITCODE -eq 0) {
    $ready = $true
    break
  }
  Start-Sleep -Seconds 2
}

if (-not $ready) {
  Write-Host "Redpanda did not become ready. Showing logs:" -ForegroundColor Red
  docker compose logs --tail=160 redpanda
  exit 1
}

Write-Host "Redpanda is ready. Starting full stack..."
docker compose up -d --build

Write-Host "Checking topics..."
powershell -ExecutionPolicy Bypass -File .\scripts\check_kafka_windows.ps1
