param(
  [string]$ImagePath = ".\self_image1.jpg",
  [string]$Name = "John Doe",
  [string]$Dob = "1990-01-01",
  [string]$Gender = "M"
)

if (!(Test-Path $ImagePath)) {
  Write-Host "Image not found: $ImagePath"
  Write-Host "Current folder: $(Get-Location)"
  Get-ChildItem -Force | Where-Object { $_.Name -match "(?i)self.*image|\.jpg$|\.jpeg$|\.png$" } | Select-Object Name,Length,FullName
  exit 1
}

Write-Host "Using image: $((Resolve-Path $ImagePath).Path)"
$response = curl.exe -sS -X POST http://localhost:8080/kyc/enroll `
  -F "image=@$ImagePath" `
  -F "name=$Name" `
  -F "dob=$Dob" `
  -F "gender=$Gender"

Write-Host $response
try {
  $json = $response | ConvertFrom-Json
  if ($json.transaction_id) {
    Write-Host "`nChecking status..."
    Start-Sleep -Seconds 2
    curl.exe -sS "http://localhost:8080/kyc/status/$($json.transaction_id)"
  }
} catch {}
