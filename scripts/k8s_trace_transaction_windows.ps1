param([Parameter(Mandatory=$true)][string]$Txn)
$ErrorActionPreference = "Continue"
Write-Host "== Trace transaction $Txn =="
Write-Host "API logs:"
kubectl logs deployment/api --since=60m | Select-String $Txn
Write-Host "Worker logs:"
kubectl logs deployment/worker --since=60m | Select-String $Txn
Write-Host "Kafka enroll topic:"
kubectl exec deploy/redpanda -- rpk topic consume kyc_enroll -X brokers=redpanda:9092 -o start -n 200 | Select-String $Txn
Write-Host "Kafka verify topic:"
kubectl exec deploy/redpanda -- rpk topic consume kyc_verify -X brokers=redpanda:9092 -o start -n 200 | Select-String $Txn
Write-Host "Consumer groups:"
kubectl exec deploy/redpanda -- rpk group describe kyc-workers-local-enroll -X brokers=redpanda:9092
kubectl exec deploy/redpanda -- rpk group describe kyc-workers-local-verify -X brokers=redpanda:9092
