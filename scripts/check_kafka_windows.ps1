$ErrorActionPreference = "Stop"
Write-Host "Checking Redpanda containers..."
docker compose ps redpanda redpanda-init

Write-Host "Checking Redpanda REST proxy from Windows host..."
$headers = @("-H", "Accept: application/vnd.kafka.v2+json")
try {
  curl.exe -sS @headers http://localhost:18082/topics
} catch {
  Write-Host "External port 18082 failed; trying 8082..."
  curl.exe -sS @headers http://localhost:8082/topics
}
Write-Host ""

Write-Host "Checking topics from inside the Docker network..."
docker compose exec redpanda rpk topic list -X brokers=redpanda:9092

Write-Host "Checking API container..."
docker compose ps api
