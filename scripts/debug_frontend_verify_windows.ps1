$ErrorActionPreference = "Continue"
Write-Host "=== Frontend verify debug checklist ==="
Write-Host "1) Confirm frontend API Base URL is the same backend you are watching."
Write-Host "   Local kind should be: http://localhost:8080"
Write-Host "   GitHub Pages must use HTTPS Cloudflare tunnel URL, not http://localhost."
Write-Host ""
Write-Host "=== API health ==="
curl.exe http://localhost:8080/health
Write-Host ""
Write-Host "=== Last API logs ==="
kubectl logs deployment/api --tail=80
Write-Host ""
Write-Host "=== Last worker logs ==="
kubectl logs deployment/worker --tail=120
Write-Host ""
Write-Host "=== Redpanda topics ==="
kubectl exec deploy/redpanda -- rpk topic list -X brokers=redpanda:9092
Write-Host ""
Write-Host "Open browser DevTools -> Network -> verify request. Confirm:"
Write-Host "Request URL ends with /kyc/verify"
Write-Host "Response JSON has type=verify and kafka_topic=kyc_verify"
Write-Host "If response lacks kafka_topic, your browser is hitting an old backend or old cached UI."
