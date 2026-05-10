param(
  [string]$ClusterName = "visual-kyc",
  [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"
Write-Host "== Visual KYC: fix verify worker consumer ==" -ForegroundColor Cyan
Write-Host "This script rebuilds/reloads the worker image, resets local Kafka command topics, and restarts one worker." -ForegroundColor Yellow

kubectl config use-context "kind-$ClusterName"

if (-not $SkipBuild) {
  Write-Host "\n[1/6] Building worker image..." -ForegroundColor Cyan
  docker build -t visual-kyc-worker:latest --build-arg TARGET=worker .\api

  Write-Host "\n[2/6] Loading worker image into kind..." -ForegroundColor Cyan
  kind load docker-image visual-kyc-worker:latest --name $ClusterName
} else {
  Write-Host "\n[1/6] Skipping worker image build/load." -ForegroundColor Yellow
}

Write-Host "\n[3/6] Upgrading Helm chart..." -ForegroundColor Cyan
helm upgrade --install visual-kyc .\helm\visual-kyc `
  --set replicaCount.worker=1 `
  --set replicaCount.api=1 `
  --set replicaCount.inference=1

Write-Host "\n[4/6] Stopping worker and resetting local Kafka command topics..." -ForegroundColor Cyan
kubectl scale deployment worker --replicas=0
Start-Sleep -Seconds 8
kubectl delete pod -l app=worker --force --grace-period=0 --ignore-not-found | Out-Host

kubectl exec deploy/redpanda -- rpk topic delete kyc_enroll -X brokers=redpanda:9092 2>$null | Out-Host
kubectl exec deploy/redpanda -- rpk topic delete kyc_verify -X brokers=redpanda:9092 2>$null | Out-Host
Start-Sleep -Seconds 3
kubectl exec deploy/redpanda -- rpk topic create kyc_enroll -p 3 -r 1 -X brokers=redpanda:9092 | Out-Host
kubectl exec deploy/redpanda -- rpk topic create kyc_verify -p 3 -r 1 -X brokers=redpanda:9092 | Out-Host

Write-Host "\n[5/6] Starting worker..." -ForegroundColor Cyan
kubectl scale deployment worker --replicas=1
kubectl rollout restart deployment/worker
kubectl rollout status deployment/worker --timeout=180s

Write-Host "\n[6/6] Verification checks..." -ForegroundColor Cyan
kubectl get pods
Write-Host "\nKafka topics:" -ForegroundColor Cyan
kubectl exec deploy/redpanda -- rpk topic list -X brokers=redpanda:9092
Write-Host "\nWorker logs:" -ForegroundColor Cyan
kubectl logs deployment/worker --tail=80

Write-Host "\nDone. Submit a NEW /kyc/verify request. Old ACCEPTED transactions may remain old." -ForegroundColor Green
