# Test Report

Generated for the final full-stack UI + Kubernetes package.

## Verified in this environment

```bash
cd api && go test ./...
```

Result: passed.

```bash
cd inference && python3 -m py_compile app/*.py
```

Result: passed.

```bash
node --check frontend/app.js
```

Result: passed.

## Not executed inside this environment

Docker Compose, kind, Helm, Redpanda, Qdrant, Redis, and Cloudflare Tunnel cannot be fully executed from this container environment. The `working.md` file contains the exact Windows commands to run the end-to-end system locally.

## Most important manual checks on your Windows machine

```powershell
curl.exe http://localhost:8080/health
powershell -ExecutionPolicy Bypass -File .\scripts\test_ui_cors_windows.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\open_frontend_local_windows.ps1
```

For Kubernetes:

```powershell
helm upgrade --install visual-kyc .\helm\visual-kyc
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_create_topics_windows.ps1
kubectl port-forward svc/api 8080:8080
```
