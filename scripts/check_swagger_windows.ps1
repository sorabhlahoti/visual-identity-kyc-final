Write-Host "Checking API health..."
curl.exe -s http://localhost:8080/health
Write-Host "`nChecking OpenAPI YAML..."
curl.exe -s -o NUL -w "OpenAPI HTTP status: %{http_code}`n" http://localhost:8080/openapi.yaml
Write-Host "Checking Swagger UI container..."
docker compose ps swagger-ui
Write-Host "`nOpen this in browser: http://localhost:8081"
Write-Host "If AUTH_REQUIRED=true, get a token: curl.exe -X POST http://localhost:8080/auth/token"
