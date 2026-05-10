Write-Host "=== Inference pods ==="
kubectl get pods -l app=inference -o wide
Write-Host "`n=== Inference service ==="
kubectl get svc inference -o wide
Write-Host "`n=== Inference endpoints ==="
kubectl get endpoints inference -o yaml
Write-Host "`n=== Inference logs ==="
kubectl logs deployment/inference --tail=160
Write-Host "`n=== Test inference health from worker pod network ==="
$pod = kubectl get pod -l app=worker -o jsonpath='{.items[0].metadata.name}'
if ($pod) {
  kubectl exec $pod -- sh -c "wget -qO- http://inference:8001/health || curl -sS http://inference:8001/health || true"
} else {
  Write-Host "No worker pod found"
}
