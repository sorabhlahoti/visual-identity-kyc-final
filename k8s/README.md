# Kubernetes Deployment

Use the Helm chart in:

```text
helm/visual-kyc
```

The old raw YAML manifests were intentionally replaced by Helm because the final architecture has multiple services: API, worker, inference, Redpanda, Qdrant, Redis, secrets, config, services, ingress, and HPA.
