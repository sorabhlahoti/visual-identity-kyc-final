$ErrorActionPreference = "Stop"

Write-Host "Starting local API public demo using Kubernetes port-forward + Cloudflare Quick Tunnel..."
Write-Host "This is for demos/testing. For production, create a named Cloudflare Tunnel."

$cf = Get-Command cloudflared -ErrorAction SilentlyContinue
if (-not $cf) {
  Write-Error "cloudflared is not installed. Install it with: winget install --id Cloudflare.cloudflared"
  exit 1
}

kubectl get svc api | Out-Host

Write-Host "Opening port-forward in a new PowerShell window..."
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; kubectl port-forward svc/api 8080:8080"
Start-Sleep -Seconds 5

Write-Host "Checking local API..."
curl.exe http://localhost:8080/health | Out-Host

Write-Host "Starting Cloudflare tunnel. Copy the https://*.trycloudflare.com URL and paste it in the frontend API Base URL field."
cloudflared tunnel --url http://localhost:8080
