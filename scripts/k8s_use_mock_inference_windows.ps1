$ErrorActionPreference = "Stop"

Write-Host "Switching local Kubernetes to stable mock inference mode..."
helm upgrade --install visual-kyc .\helm\visual-kyc `
  --set env.embeddingMode=mock `
  --set env.livenessRequired=false `
  --set env.failIfNoFace=false `
  --set replicaCount.inference=1 `
  --set replicaCount.worker=1 `
  --set replicaCount.api=1

kubectl rollout restart deployment/inference
kubectl rollout restart deployment/worker
kubectl rollout status deployment/inference --timeout=180s
kubectl rollout status deployment/worker --timeout=180s

Write-Host "Checking inference health from inside cluster..."
$pod = kubectl get pod -l app=worker -o jsonpath='{.items[0].metadata.name}'
if ($pod) {
  kubectl exec $pod -- sh -c "wget -qO- http://inference:8001/health || curl -sS http://inference:8001/health || true"
}
Write-Host "Mock mode applied. Submit a fresh request; old FAILED transactions remain failed."
