Write-Host "=== Current containers, including exited ==="
docker compose ps -a

Write-Host "`n=== Checking port 8080 owner ==="
$portInfo = netstat -ano | findstr ":8080"
if ($portInfo) {
  Write-Host $portInfo
  Write-Host "If this is not Docker/kyc-api, either stop that process or run with API_PORT=8088."
} else {
  Write-Host "Port 8080 appears free."
}

Write-Host "`n=== Rebuilding and starting API ==="
docker compose up -d --build api

Write-Host "`n=== API container status ==="
docker compose ps -a api

Write-Host "`n=== API logs ==="
docker compose logs --tail=120 api

Write-Host "`n=== Health check on 8080 ==="
try {
  curl.exe http://localhost:8080/health
} catch {
  Write-Host "8080 failed. Trying alternate host port 8088..."
  $env:API_PORT="8088"
  docker compose up -d --build api
  curl.exe http://localhost:8088/health
}
