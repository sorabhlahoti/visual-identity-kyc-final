Write-Host "=== Docker Compose services, including exited containers ==="
docker compose ps -a

Write-Host "`n=== Local port 8080 owner/process ==="
$portInfo = netstat -ano | findstr ":8080"
if ($portInfo) {
  Write-Host $portInfo
  Write-Host "If kyc-api is not running, some other process may be owning port 8080."
} else {
  Write-Host "No local process currently owns 8080."
}

Write-Host "`n=== API health from Windows host ==="
try {
  curl.exe -v http://localhost:8080/health
} catch {
  Write-Host $_
}

Write-Host "`n=== API logs ==="
docker compose logs --tail=160 api

Write-Host "`n=== Worker logs ==="
docker compose logs --tail=120 worker

Write-Host "`n=== Redpanda init logs ==="
docker compose logs --tail=120 redpanda-init

Write-Host "`n=== Redpanda topics from container ==="
docker compose exec redpanda rpk topic list -X brokers=redpanda:9092

Write-Host "`n=== Redpanda REST topics from Windows host ==="
curl.exe -sS -H "Accept: application/vnd.kafka.v2+json" http://localhost:18082/topics

Write-Host "`n=== Image file check in current folder ==="
Get-ChildItem -Force | Where-Object { $_.Name -match "(?i)self.*image|\.jpg$|\.jpeg$|\.png$" } | Select-Object Name,Length,FullName

Write-Host "`n=== Suggested next command if api is exited ==="
Write-Host "docker compose up -d --build api"
