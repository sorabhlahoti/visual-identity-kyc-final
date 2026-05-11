param(
  [string]$InferenceMode = "arcface",
  [switch]$BuildApiWorker
)
$ErrorActionPreference = "Stop"
Write-Host "== Visual KYC hard reset pipeline =="
kubectl config use-context kind-visual-kyc
if ($BuildApiWorker) {
  Write-Host "Building API and worker images..."
  docker build -t visual-kyc-api:latest --build-arg TARGET=server .\api
  docker build -t visual-kyc-worker:latest --build-arg TARGET=worker .\api
  kind load docker-image visual-kyc-api:latest --name visual-kyc
  kind load docker-image visual-kyc-worker:latest --name visual-kyc
}
Write-Host "Upgrading Helm..."
helm upgrade --install visual-kyc .\helm\visual-kyc `
  --set env.embeddingMode=$InferenceMode `
  --set env.livenessRequired=true `
  --set replicaCount.api=1 `
  --set replicaCount.worker=0 `
  --set replicaCount.inference=1
Write-Host "Waiting for API/inference/redpanda..."
kubectl rollout status deployment/api --timeout=180s
kubectl rollout status deployment/inference --timeout=240s
kubectl rollout status deployment/redpanda --timeout=180s
Write-Host "Checking inference endpoint..."
kubectl get endpoints inference -o wide
Write-Host "Resetting Kafka command topics..."
kubectl exec deploy/redpanda -- rpk topic delete kyc_enroll -X brokers=redpanda:9092 2>$null | Out-Host
kubectl exec deploy/redpanda -- rpk topic delete kyc_verify -X brokers=redpanda:9092 2>$null | Out-Host
Start-Sleep -Seconds 3
kubectl exec deploy/redpanda -- rpk topic create kyc_enroll -p 3 -r 1 -X brokers=redpanda:9092
kubectl exec deploy/redpanda -- rpk topic create kyc_verify -p 3 -r 1 -X brokers=redpanda:9092
kubectl exec deploy/redpanda -- rpk topic list -X brokers=redpanda:9092
Write-Host "Starting worker..."
kubectl scale deployment worker --replicas=1
kubectl rollout status deployment/worker --timeout=180s
Write-Host "Pods:"
kubectl get pods
Write-Host "Worker logs:"
kubectl logs deployment/worker --tail=80
Write-Host "Done. Now submit a fresh request. Use a valid DOB like 1990-01-21, not 1990-21-01."
