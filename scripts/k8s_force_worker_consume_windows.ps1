$ErrorActionPreference = "Stop"

Write-Host "Step 1: Ensure Kafka topics exist"
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_create_topics_windows.ps1

Write-Host "`nStep 2: Force a fresh worker consumer group so old offsets/stale consumers cannot block processing"
$group = "kyc-workers-local-" + (Get-Date -Format "yyyyMMddHHmmss")
Write-Host "Using consumer group: $group"
kubectl set env deployment/worker KAFKA_CONSUMER_GROUP=$group --overwrite

Write-Host "`nStep 3: Restart worker"
kubectl rollout restart deployment/worker
kubectl rollout status deployment/worker --timeout=180s

Write-Host "`nStep 4: Current pods"
kubectl get pods

Write-Host "`nStep 5: Worker logs"
kubectl logs deployment/worker --tail=120

Write-Host "`nNow submit a new enroll/verify request, then run:"
Write-Host "kubectl logs deployment/worker --tail=160"
