param(
  [ValidateSet("docker", "k8s")]
  [string]$Target = "k8s",

  [ValidateSet("mock", "arcface")]
  [string]$InferenceMode = "mock",

  [switch]$CreateKind,
  [switch]$BuildImages,
  [switch]$LoadInference,
  [switch]$StartTunnel,
  [switch]$StartFrontend,
  [switch]$Clean,
  [int]$ApiPort = 8080
)

$ErrorActionPreference = "Stop"

function Require-Command($Name, $InstallHint) {
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    Write-Error "$Name not found. $InstallHint"
    exit 1
  }
}

function Wait-Http($Url, $Seconds = 90) {
  $deadline = (Get-Date).AddSeconds($Seconds)
  while ((Get-Date) -lt $deadline) {
    try {
      $r = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 5
      if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 500) { return $true }
    } catch {}
    Start-Sleep -Seconds 3
  }
  return $false
}

Write-Host "Visual KYC one-click runner" -ForegroundColor Cyan
Write-Host "Target=$Target InferenceMode=$InferenceMode BuildImages=$BuildImages StartTunnel=$StartTunnel StartFrontend=$StartFrontend" -ForegroundColor Cyan

if ($Target -eq "docker") {
  Require-Command docker "Install Docker Desktop."
  if ($Clean) { docker compose down -v } else { docker compose down }
  docker compose up -d --build
  Write-Host "Waiting for API..." -ForegroundColor Cyan
  if (-not (Wait-Http "http://localhost:$ApiPort/health" 120)) { Write-Warning "API did not respond yet. Check: docker compose logs api" }
  if ($StartFrontend) { Start-Process ((Resolve-Path .\frontend\index.html).Path) }
  if ($StartTunnel) { Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; .\scripts\live_api_cloudflare_windows.ps1 -ApiPort $ApiPort -ApiUrl http://localhost:$ApiPort -SkipPortForward" }
  docker compose ps
  Write-Host "Docker local stack command finished." -ForegroundColor Green
  exit 0
}

Require-Command kubectl "Install with: winget install -e --id Kubernetes.kubectl"
Require-Command helm "Install with: winget install -e --id Helm.Helm"
Require-Command kind "Install with: winget install -e --id Kubernetes.kind"
Require-Command docker "Install Docker Desktop."

if ($CreateKind) {
  $contexts = kubectl config get-contexts -o name
  if ($contexts -notcontains "kind-visual-kyc") {
    Write-Host "Creating kind cluster visual-kyc..." -ForegroundColor Cyan
    kind create cluster --name visual-kyc --image kindest/node:v1.30.0 --wait 5m
  }
}

kubectl config use-context kind-visual-kyc

if ($Clean) {
  Write-Host "Removing old Helm release before reinstall..." -ForegroundColor Yellow
  try { helm uninstall visual-kyc 2>$null } catch { Write-Host "No previous Helm release to remove." -ForegroundColor Yellow }
}

if ($BuildImages) {
  Write-Host "Building API and worker images..." -ForegroundColor Cyan
  docker build -t visual-kyc-api:latest --build-arg TARGET=server .\api
  docker build -t visual-kyc-worker:latest --build-arg TARGET=worker .\api
  kind load docker-image visual-kyc-api:latest --name visual-kyc
  kind load docker-image visual-kyc-worker:latest --name visual-kyc

  if ($LoadInference -or $InferenceMode -eq "arcface") {
    Write-Host "Building/loading inference image. This can be large." -ForegroundColor Yellow
    docker build -t visual-kyc-inference:latest .\inference
    kind load docker-image visual-kyc-inference:latest --name visual-kyc
  }
}

if ($InferenceMode -eq "arcface") {
  .\scripts\k8s_use_arcface_inference_windows.ps1 -BuildAndLoadImage:$LoadInference
} else {
  .\scripts\k8s_use_mock_inference_windows.ps1
}

Write-Host "Creating Kafka topics..." -ForegroundColor Cyan
.\scripts\k8s_create_topics_windows.ps1

Write-Host "Restarting API and worker..." -ForegroundColor Cyan
kubectl rollout restart deployment/api
kubectl rollout restart deployment/worker
kubectl rollout status deployment/api --timeout=180s
kubectl rollout status deployment/worker --timeout=180s
kubectl get pods

Write-Host "Starting API port-forward in a new PowerShell window..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; kubectl port-forward svc/api $ApiPort`:8080"
Start-Sleep -Seconds 6

if (Wait-Http "http://localhost:$ApiPort/health" 60) {
  Write-Host "API is reachable at http://localhost:$ApiPort" -ForegroundColor Green
} else {
  Write-Warning "API not reachable yet. Check: kubectl logs deployment/api --tail=100"
}

if ($StartFrontend) {
  Write-Host "Opening frontend..." -ForegroundColor Cyan
  Start-Process ((Resolve-Path .\frontend\index.html).Path)
}

if ($StartTunnel) {
  Write-Host "Starting Cloudflare tunnel in a new PowerShell window..." -ForegroundColor Cyan
  Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; .\scripts\live_api_cloudflare_windows.ps1 -ApiPort $ApiPort -ApiUrl http://localhost:$ApiPort -SkipPortForward"
}

Write-Host "Done. Use frontend/index.html or GitHub Pages UI with API Base URL http://localhost:$ApiPort or the Cloudflare URL." -ForegroundColor Green
