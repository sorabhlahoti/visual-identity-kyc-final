Write-Host "Rebuilding API and worker containers..."
docker compose up -d --build api worker
Write-Host "`nChecking API health..."
curl.exe http://localhost:8080/health
Write-Host "`nRecent API logs..."
docker compose logs --tail=60 api
