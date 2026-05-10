$ErrorActionPreference = "Stop"
Write-Host "Creating Redpanda topics manually..."
docker compose exec redpanda rpk topic create kyc_enroll -X brokers=redpanda:9092 --partitions 3 --replicas 1 --if-not-exists
docker compose exec redpanda rpk topic create kyc_verify -X brokers=redpanda:9092 --partitions 3 --replicas 1 --if-not-exists
Write-Host "Topics now:"
docker compose exec redpanda rpk topic list -X brokers=redpanda:9092
Write-Host "REST proxy topics:"
curl.exe -sS http://localhost:18082/topics
