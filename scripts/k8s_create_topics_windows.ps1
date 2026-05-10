$ErrorActionPreference = "Continue"

Write-Host "Creating/checking Redpanda topics..." -ForegroundColor Cyan
kubectl exec deploy/redpanda -- rpk topic create kyc_enroll -p 3 -r 1 -X brokers=redpanda:9092
kubectl exec deploy/redpanda -- rpk topic create kyc_verify -p 3 -r 1 -X brokers=redpanda:9092
Write-Host "Topics now:" -ForegroundColor Cyan
kubectl exec deploy/redpanda -- rpk topic list -X brokers=redpanda:9092
Write-Host "If a topic already exists, that error is safe to ignore for local setup." -ForegroundColor Yellow
