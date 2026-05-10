param(
  [Parameter(Mandatory=$true)] [string] $TransactionId,
  [Parameter(Mandatory=$true)] [string] $Token
)

curl.exe "http://localhost:8080/kyc/status/$TransactionId" -H "Authorization: Bearer $Token"
