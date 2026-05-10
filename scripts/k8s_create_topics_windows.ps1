$ErrorActionPreference = "Continue"

Write-Host "Creating/checking Kafka topics inside Kubernetes Redpanda..."

function Get-TopicsText {
  return (kubectl exec deploy/redpanda -- rpk -X brokers=redpanda:9092 topic list) -join "`n"
}

function Ensure-Topic($TopicName) {
  Write-Host "Ensuring topic: $TopicName"
  $topicsBefore = Get-TopicsText
  if ($topicsBefore -match "(?m)^$TopicName\s") {
    Write-Host "Topic $TopicName already exists."
    return
  }

  kubectl exec deploy/redpanda -- rpk -X brokers=redpanda:9092 topic create $TopicName -p 3 -r 1
  if ($LASTEXITCODE -ne 0) {
    Write-Host "Topic create command returned non-zero. Checking whether the topic exists anyway..."
  }

  $topicsAfter = Get-TopicsText
  if ($topicsAfter -notmatch "(?m)^$TopicName\s") {
    Write-Error "Topic $TopicName was not created. Run: kubectl logs deploy/redpanda --tail=100"
    exit 1
  }
  Write-Host "Topic $TopicName is ready."
}

Ensure-Topic "kyc_enroll"
Ensure-Topic "kyc_verify"

Write-Host "Topics now:"
kubectl exec deploy/redpanda -- rpk -X brokers=redpanda:9092 topic list
