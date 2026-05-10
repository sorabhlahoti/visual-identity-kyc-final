Write-Host "=== Which local process owns port 8080? ==="
Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue | Select-Object LocalAddress,LocalPort,State,OwningProcess | Format-Table

Write-Host "`n=== Kubernetes pods ==="
kubectl get pods -o wide

Write-Host "`n=== API health through Kubernetes service, not localhost ==="
kubectl run curl-api-check --rm -i --restart=Never --image=curlimages/curl:8.10.1 -- `
  curl -sS http://api:8080/health

Write-Host "`n=== Redpanda topics ==="
kubectl exec deploy/redpanda -- rpk -X brokers=redpanda:9092 topic list

Write-Host "`n=== Worker logs ==="
kubectl logs deployment/worker --tail=160

Write-Host "`n=== API logs ==="
kubectl logs deployment/api --tail=120

Write-Host "`n=== Recent worker previous logs, if pod restarted ==="
kubectl logs deployment/worker --previous --tail=80 2>$null
