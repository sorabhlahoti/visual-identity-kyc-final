Write-Host "Creating topics first..."
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_create_topics_windows.ps1
Write-Host "Restarting worker deployment..."
kubectl rollout restart deployment/worker
kubectl rollout status deployment/worker --timeout=180s
Write-Host "Worker logs:"
kubectl logs deployment/worker --tail=80
