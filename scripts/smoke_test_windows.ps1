param(
  [string]$ImagePath = "Self_image1.jpg",
  [string]$Name = "John Doe",
  [string]$Dob = "1990-01-01",
  [string]$Gender = "M"
)
$ErrorActionPreference = "Stop"
if (!(Test-Path $ImagePath)) { throw "Image not found: $ImagePath" }

Write-Host "1) Health"
curl.exe http://localhost:8080/health

Write-Host "`n2) Enroll async"
$enroll = curl.exe -s -X POST http://localhost:8080/kyc/enroll `
  -F "image=@$ImagePath" `
  -F "name=$Name" `
  -F "dob=$Dob" `
  -F "gender=$Gender"
Write-Host $enroll
$txn = ($enroll | ConvertFrom-Json).transaction_id
if (!$txn) { throw "No transaction_id returned" }

Write-Host "`n3) Poll enroll status"
for ($i=0; $i -lt 20; $i++) {
  Start-Sleep -Seconds 1
  $status = curl.exe -s "http://localhost:8080/kyc/status/$txn"
  Write-Host $status
  $obj = $status | ConvertFrom-Json
  if ($obj.status -eq "COMPLETED" -or $obj.status -eq "FAILED") { break }
}

Write-Host "`n4) Enroll same image again - should become ALREADY_EXISTS or PARTIAL_MATCH"
$enroll2 = curl.exe -s -X POST http://localhost:8080/kyc/enroll `
  -F "image=@$ImagePath" `
  -F "name=$Name" `
  -F "dob=$Dob" `
  -F "gender=$Gender"
Write-Host $enroll2
$txn2 = ($enroll2 | ConvertFrom-Json).transaction_id
for ($i=0; $i -lt 20; $i++) {
  Start-Sleep -Seconds 1
  $status2 = curl.exe -s "http://localhost:8080/kyc/status/$txn2"
  Write-Host $status2
  $obj2 = $status2 | ConvertFrom-Json
  if ($obj2.status -eq "COMPLETED" -or $obj2.status -eq "FAILED") { break }
}

Write-Host "`n5) Verify"
$verify = curl.exe -s -X POST http://localhost:8080/kyc/verify `
  -F "image=@$ImagePath" `
  -F "name=$Name" `
  -F "dob=$Dob" `
  -F "gender=$Gender"
Write-Host $verify
$txn3 = ($verify | ConvertFrom-Json).transaction_id
for ($i=0; $i -lt 20; $i++) {
  Start-Sleep -Seconds 1
  $status3 = curl.exe -s "http://localhost:8080/kyc/status/$txn3"
  Write-Host $status3
  $obj3 = $status3 | ConvertFrom-Json
  if ($obj3.status -eq "COMPLETED" -or $obj3.status -eq "FAILED") { break }
}
