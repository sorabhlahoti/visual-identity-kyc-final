$ErrorActionPreference = "Stop"

Write-Host "Switching Kubernetes inference to real ArcFace mode..."
Write-Host "Use this only after inference/models/w600k_r50.onnx exists and the inference image was rebuilt + loaded into kind."

if (-not (Test-Path ".\inference\models\w600k_r50.onnx")) {
  Write-Warning "Model file not found: .\inference\models\w600k_r50.onnx"
  Write-Host "Download it first from inside inference folder:"
  Write-Host "  cd inference"
  Write-Host "  uv sync"
  Write-Host "  uv run python scripts/download_models.py"
  Write-Host "Then rebuild/load the inference image:"
  Write-Host "  docker build -t visual-kyc-inference:latest .\inference"
  Write-Host "  kind load docker-image visual-kyc-inference:latest --name visual-kyc"
  exit 1
}

helm upgrade --install visual-kyc .\helm\visual-kyc `
  -f .\helm\visual-kyc\values-arcface.yaml `
  --set env.embeddingMode=arcface `
  --set env.livenessRequired=true `
  --set env.failIfNoFace=true `
  --set env.arcfaceModelPath=/app/models/w600k_r50.onnx

kubectl rollout restart deployment/inference
kubectl rollout restart deployment/worker
kubectl rollout status deployment/inference --timeout=300s
kubectl rollout status deployment/worker --timeout=180s

Write-Host "Inference health:"
$pod = kubectl get pod -l app=worker -o jsonpath='{.items[0].metadata.name}'
if ($pod) {
  kubectl exec $pod -- sh -c "wget -qO- http://inference:8001/health || curl -sS http://inference:8001/health || true"
}

Write-Host "ArcFace mode applied. Submit a fresh enroll/verify request."
