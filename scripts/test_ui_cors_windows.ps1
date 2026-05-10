$ErrorActionPreference = "Stop"
param(
  [string]$ApiBase = "http://localhost:8080",
  [string]$Origin = "https://example.github.io"
)

$ApiBase = $ApiBase.TrimEnd('/')
Write-Host "Testing CORS preflight against $ApiBase from origin $Origin"

curl.exe -i -X OPTIONS "$ApiBase/kyc/enroll" `
  -H "Origin: $Origin" `
  -H "Access-Control-Request-Method: POST" `
  -H "Access-Control-Request-Headers: authorization, content-type"

Write-Host "`nTesting health with Origin header"
curl.exe -i "$ApiBase/health" -H "Origin: $Origin"
