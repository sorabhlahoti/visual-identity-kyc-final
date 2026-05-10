param(
  [int]$ApiPort = 8080,
  [string]$ApiUrl = "http://localhost:8080",
  [switch]$SkipPortForward
)

$ErrorActionPreference = "Stop"

Write-Host "Starting Cloudflare Quick Tunnel for Visual KYC API..." -ForegroundColor Cyan
Write-Host "Quick Tunnel is for demo/testing. Keep this window open while using the public URL." -ForegroundColor Yellow

$cf = Get-Command cloudflared -ErrorAction SilentlyContinue
if (-not $cf) {
  Write-Error "cloudflared is not installed. Install it with: winget install --id Cloudflare.cloudflared"
  exit 1
}

function Test-LocalApi {
  try {
    $health = Invoke-WebRequest -Uri "$ApiUrl/health" -UseBasicParsing -TimeoutSec 5
    return ($health.StatusCode -eq 200)
  } catch {
    return $false
  }
}

if (-not $SkipPortForward) {
  if (-not (Test-LocalApi)) {
    Write-Host "API is not reachable at $ApiUrl. Starting kubectl port-forward svc/api $ApiPort`:8080 in a new PowerShell window..." -ForegroundColor Yellow
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; kubectl port-forward svc/api $ApiPort`:8080"
    Start-Sleep -Seconds 6
  }
}

if (-not (Test-LocalApi)) {
  Write-Error "API still not reachable at $ApiUrl. First run: kubectl port-forward svc/api $ApiPort`:8080"
  exit 1
}

Write-Host "API health OK: $ApiUrl/health" -ForegroundColor Green

Write-Host "Checking DNS for Cloudflare tunnel endpoint..." -ForegroundColor Cyan
try {
  Resolve-DnsName region1.v2.argotunnel.com -ErrorAction Stop | Out-Null
} catch {
  Write-Warning "DNS lookup failed for region1.v2.argotunnel.com. If the tunnel fails, change your Windows/Wi-Fi DNS to 1.1.1.1 or 8.8.8.8 and retry."
}

# A temporary empty config avoids accidentally loading ~/.cloudflared/config.yml for a named tunnel,
# which can cause origin-cert errors during a Quick Tunnel demo.
$tmpDir = Join-Path $env:TEMP "visual-kyc-cloudflared"
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null
$emptyConfig = Join-Path $tmpDir "empty-config.yml"
"# intentionally empty; prevents accidental named-tunnel config loading" | Set-Content -Encoding ascii $emptyConfig

Write-Host "Starting tunnel. Copy the https://*.trycloudflare.com URL into the frontend API Base URL field." -ForegroundColor Green
Write-Host "If QUIC is blocked by your network, this script uses --protocol http2 to work over TCP." -ForegroundColor Yellow

& cloudflared --config $emptyConfig tunnel --no-autoupdate --protocol http2 --url $ApiUrl
