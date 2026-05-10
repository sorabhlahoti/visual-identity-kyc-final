param(
  [string]$ImagePath = ".\Self_image1.jpg",
  [switch]$BuildImages,
  [string]$InferenceMode = "arcface"
)

$ErrorActionPreference = "Stop"
Write-Host "== Visual KYC verify end-to-end fixer ==" -ForegroundColor Cyan

kubectl config use-context kind-visual-kyc | Out-Host

if ($BuildImages) {
  Write-Host "Building API image..." -ForegroundColor Yellow
  docker build -t visual-kyc-api:latest --build-arg TARGET=server .\api | Out-Host
  Write-Host "Building worker image..." -ForegroundColor Yellow
  docker build -t visual-kyc-worker:latest --build-arg TARGET=worker .\api | Out-Host
  Write-Host "Loading API image into kind..." -ForegroundColor Yellow
  kind load docker-image visual-kyc-api:latest --name visual-kyc | Out-Host
  Write-Host "Loading worker image into kind..." -ForegroundColor Yellow
  kind load docker-image visual-kyc-worker:latest --name visual-kyc | Out-Host
}

Write-Host "Upgrading Helm with explicit Kafka topics..." -ForegroundColor Yellow
helm upgrade --install visual-kyc .\helm\visual-kyc `
  --set env.embeddingMode=$InferenceMode `
  --set env.kafkaTopicEnroll=kyc_enroll `
  --set env.kafkaTopicVerify=kyc_verify `
  --set env.livenessRequired=true `
  --set replicaCount.api=1 `
  --set replicaCount.worker=1 `
  --set replicaCount.inference=1 | Out-Host

Write-Host "Restarting API and worker so latest config/code is used..." -ForegroundColor Yellow
kubectl rollout restart deployment/api | Out-Host
kubectl rollout restart deployment/worker | Out-Host
kubectl rollout status deployment/api --timeout=180s | Out-Host
kubectl rollout status deployment/worker --timeout=180s | Out-Host

Write-Host "Resetting local Kafka topics to remove stale test queue..." -ForegroundColor Yellow
kubectl exec deploy/redpanda -- rpk topic delete kyc_enroll -X brokers=redpanda:9092 2>$null | Out-Host
kubectl exec deploy/redpanda -- rpk topic delete kyc_verify -X brokers=redpanda:9092 2>$null | Out-Host
Start-Sleep -Seconds 3
kubectl exec deploy/redpanda -- rpk topic create kyc_enroll -p 3 -r 1 -X brokers=redpanda:9092 | Out-Host
kubectl exec deploy/redpanda -- rpk topic create kyc_verify -p 3 -r 1 -X brokers=redpanda:9092 | Out-Host
kubectl exec deploy/redpanda -- rpk topic list -X brokers=redpanda:9092 | Out-Host

Write-Host "Restarting worker after topic reset..." -ForegroundColor Yellow
kubectl rollout restart deployment/worker | Out-Host
kubectl rollout status deployment/worker --timeout=180s | Out-Host

Write-Host "Checking API port-forward on localhost:8080..." -ForegroundColor Yellow
$portOpen = $false
try {
  $tcp = New-Object System.Net.Sockets.TcpClient
  $iar = $tcp.BeginConnect("127.0.0.1", 8080, $null, $null)
  $portOpen = $iar.AsyncWaitHandle.WaitOne(500, $false)
  if ($portOpen) { $tcp.EndConnect($iar) }
  $tcp.Close()
} catch { $portOpen = $false }

if (-not $portOpen) {
  Write-Host "Starting kubectl port-forward svc/api 8080:8080 in a new PowerShell window..." -ForegroundColor Yellow
  Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; kubectl port-forward svc/api 8080:8080"
  Start-Sleep -Seconds 5
}

Write-Host "Health check:" -ForegroundColor Yellow
curl.exe -s http://localhost:8080/health | Out-Host

if (-not (Test-Path $ImagePath)) {
  throw "Image not found: $ImagePath. Pass -ImagePath .\YourImage.jpg"
}

Write-Host "Getting bearer token..." -ForegroundColor Yellow
$tokenResponse = Invoke-RestMethod -Method Post http://localhost:8080/auth/token
$token = $tokenResponse.token

Write-Host "Submitting VERIFY request. Response must show kafka_topic = kyc_verify" -ForegroundColor Yellow
$verifyRaw = curl.exe -s -X POST http://localhost:8080/kyc/verify `
  -F "image=@$ImagePath" `
  -F "name=John Doe" `
  -F "dob=1990-01-01" `
  -F "gender=M" `
  -H "Authorization: Bearer $token"
Write-Host $verifyRaw
$verify = $verifyRaw | ConvertFrom-Json

if ($verify.kafka_topic -ne "kyc_verify") {
  Write-Host "ERROR: API did not publish verify to kyc_verify. Check API logs below." -ForegroundColor Red
  kubectl logs deployment/api --tail=80 | Out-Host
  exit 1
}

Write-Host "Polling status for $($verify.transaction_id)..." -ForegroundColor Yellow
for ($i = 1; $i -le 30; $i++) {
  Start-Sleep -Seconds 2
  $statusRaw = curl.exe -s "http://localhost:8080/kyc/status/$($verify.transaction_id)" -H "Authorization: Bearer $token"
  Write-Host $statusRaw
  try { $status = $statusRaw | ConvertFrom-Json } catch { continue }
  if ($status.status -eq "COMPLETED" -or $status.status -eq "FAILED") { break }
}

Write-Host "Worker logs:" -ForegroundColor Yellow
kubectl logs deployment/worker --tail=160 | Out-Host

Write-Host "API logs:" -ForegroundColor Yellow
kubectl logs deployment/api --tail=80 | Out-Host
