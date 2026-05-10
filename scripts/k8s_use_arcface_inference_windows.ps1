param(
  [switch]$BuildAndLoadImage,
  [switch]$SkipModelDownload
)

$ErrorActionPreference = "Stop"

Write-Host "Switching Kubernetes inference to real ArcFace mode..." -ForegroundColor Cyan
Write-Host "This requires inference/models/w600k_r50.onnx inside the inference Docker image." -ForegroundColor Yellow

if (-not (Test-Path ".\inference\models\w600k_r50.onnx")) {
  if ($SkipModelDownload) {
    Write-Error "Model file missing: .\inference\models\w600k_r50.onnx"
    exit 1
  }
  Write-Host "Model not found. Downloading with uv inside inference folder..." -ForegroundColor Yellow
  Push-Location .\inference
  if (-not (Get-Command uv -ErrorAction SilentlyContinue)) {
    Pop-Location
    Write-Error "uv is not installed. Install it first: powershell -ExecutionPolicy Bypass -c \"irm https://astral.sh/uv/install.ps1 | iex\""
    exit 1
  }
  uv sync
  uv run python scripts/download_models.py
  Pop-Location
}

if ($BuildAndLoadImage) {
  Write-Host "Building and loading inference image into kind. This image is large and needs free disk space." -ForegroundColor Yellow
  docker build -t visual-kyc-inference:latest .\inference
  kind load docker-image visual-kyc-inference:latest --name visual-kyc
} else {
  Write-Host "Skipping image build/load. Use -BuildAndLoadImage if the model was newly downloaded." -ForegroundColor Yellow
}

helm upgrade --install visual-kyc .\helm\visual-kyc `
  -f .\helm\visual-kyc\values-arcface.yaml `
  --set env.embeddingMode=arcface `
  --set env.livenessRequired=true `
  --set env.failIfNoFace=true `
  --set env.arcfaceModelPath=/app/models/w600k_r50.onnx `
  --set replicaCount.inference=1 `
  --set replicaCount.worker=1 `
  --set replicaCount.api=1

kubectl rollout restart deployment/inference
kubectl rollout restart deployment/worker
kubectl rollout status deployment/inference --timeout=300s
kubectl rollout status deployment/worker --timeout=180s

Write-Host "Checking inference health from inside cluster..." -ForegroundColor Cyan
$pod = kubectl get pod -l app=worker -o jsonpath='{.items[0].metadata.name}'
if ($pod) {
  kubectl exec $pod -- sh -c "wget -qO- http://inference:8001/health || curl -sS http://inference:8001/health || true"
}

Write-Host "ArcFace mode applied. Submit a NEW transaction and poll status. ACCEPTED only means queued; final result appears after worker processing." -ForegroundColor Green
