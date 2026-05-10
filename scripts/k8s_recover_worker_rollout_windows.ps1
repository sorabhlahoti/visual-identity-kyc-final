Write-Host "Recovering worker rollout for local kind..."

Write-Host "Using current Kubernetes context:"
kubectl config current-context

Write-Host "Deleting local HPA objects if they exist, because local kind defaults use fixed replicas..."
kubectl delete hpa api worker --ignore-not-found=true

Write-Host "Upgrading Helm chart..."
helm upgrade --install visual-kyc .\helm\visual-kyc

Write-Host "Force local replica counts to 1..."
kubectl scale deployment api --replicas=1
kubectl scale deployment worker --replicas=1

Write-Host "Remove any old worker initContainers left from a previous chart revision..."
kubectl patch deployment worker --type=json -p='[{"op":"remove","path":"/spec/template/spec/initContainers"}]' 2>$null

Write-Host "Restart API and worker..."
kubectl rollout restart deployment/api
kubectl rollout restart deployment/worker

Write-Host "Waiting for rollout..."
kubectl rollout status deployment/api --timeout=180s
kubectl rollout status deployment/worker --timeout=180s

Write-Host "Create/check Kafka topics..."
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_create_topics_windows.ps1

Write-Host "Current pods:"
kubectl get pods
