# Visual Identity KYC — Working Guide

This guide is written for Windows + Docker Desktop users. It explains how to run locally, test APIs, deploy with Kubernetes/Helm, make a public demo with Cloudflare Tunnel, and publish the frontend on GitHub Pages.

---

## 0. What you are running

```text
Frontend UI               GitHub Pages / local browser
Go API                    receives enroll/verify/status requests
Redpanda Kafka            async jobs: kyc_enroll, kyc_verify
Go Worker                 consumes jobs and executes KYC logic
Python Inference          512D face + 768D name + liveness
Qdrant                    vector database
Redis                     transaction status + secure metadata
Prometheus + Grafana      observability
```

Core async flow:

```text
User submits image + demographics
→ API returns transaction_id immediately
→ API publishes encrypted job to Kafka
→ Worker consumes job
→ Worker calls Python inference
→ Worker searches/upserts Qdrant
→ Worker writes status/result to Redis
→ UI polls /kyc/status/{transaction_id}
```

No raw biometric image is stored by the backend. The image is processed in memory and discarded.

---

## 1. Required tools

Install these first:

```powershell
winget install -e --id Docker.DockerDesktop
winget install -e --id GoLang.Go
winget install -e --id astral-sh.uv
```

For Kubernetes/Helm:

```powershell
winget install -e --id Kubernetes.kubectl
winget install -e --id Helm.Helm
winget install -e --id Kubernetes.kind
```

For public demo:

```powershell
winget install --id Cloudflare.cloudflared
```

Check versions:

```powershell
docker --version
go version
uv --version
kubectl version --client
helm version
kind --version
cloudflared --version
```

---

## 2. Start backend locally with Docker Compose

From project root:

```powershell
cd C:\Users\sorab\Desktop\visual-identity-kyc-final
copy .env.example .env
```

Start everything:

```powershell
docker compose up -d --build
```

Check services:

```powershell
docker compose ps
curl.exe http://localhost:8080/health
```

Expected:

```json
{"mode":"async","service":"kyc-api","status":"ok"}
```

Create/check Kafka topics:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\create_topics_windows.ps1
```

---

## 3. Get auth token

If `AUTH_REQUIRED=true`, get a demo token:

```powershell
$tokenResponse = curl.exe -s -X POST http://localhost:8080/auth/token | ConvertFrom-Json
$token = $tokenResponse.token
```

Use it like:

```powershell
-H "Authorization: Bearer $token"
```

---

## 4. Test enroll and status

Put your test image in project root, for example:

```text
Self_image1.jpg
```

Enroll:

```powershell
curl.exe -X POST http://localhost:8080/kyc/enroll `
  -F "image=@Self_image1.jpg" `
  -F "name=John Doe" `
  -F "dob=1990-01-01" `
  -F "gender=M" `
  -H "Authorization: Bearer $token"
```

Copy `transaction_id`.

Check status:

```powershell
curl.exe http://localhost:8080/kyc/status/YOUR_TRANSACTION_ID `
  -H "Authorization: Bearer $token"
```

Expected lifecycle:

```text
ACCEPTED → PROCESSING → COMPLETED
```

First enrollment should return `NEW_USER`. Repeating the same enrollment should return `ALREADY_EXISTS`.

---

## 5. Test verify

```powershell
curl.exe -X POST http://localhost:8080/kyc/verify `
  -F "image=@Self_image1.jpg" `
  -F "name=John Doe" `
  -F "dob=1990-01-01" `
  -F "gender=M" `
  -H "Authorization: Bearer $token"
```

Then check status using the returned transaction ID.

Expected decision after successful enrollment:

```text
MATCHED / PARTIAL_MATCH / NO_MATCH
```

---

## 6. Run the frontend locally

The frontend is static and has no Node dependency.

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\open_frontend_local_windows.ps1
```

Open:

```text
http://localhost:5173
```

Use API Base URL:

```text
http://localhost:8080
```

Click:

```text
Check API Health → Get Demo Token → Enroll user → Poll until done → Verify user
```

---

## 7. Test CORS for UI

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\test_ui_cors_windows.ps1 -ApiBase http://localhost:8080 -Origin https://yourname.github.io
```

Expected response headers should include:

```text
Access-Control-Allow-Origin
Access-Control-Allow-Headers
Access-Control-Allow-Methods
```

For demo, default backend CORS is permissive:

```env
CORS_ALLOWED_ORIGINS=*
```

For stricter production, use comma-separated origins:

```env
CORS_ALLOWED_ORIGINS=https://YOUR_GITHUB_USERNAME.github.io,https://YOUR_TUNNEL_OR_API_DOMAIN
```

---

## 8. Local Kubernetes with kind + Helm

Make sure Docker Desktop is running in Linux container mode.

Create cluster:

```powershell
kind create cluster --name visual-kyc --image kindest/node:v1.30.0 --wait 5m
kubectl config use-context kind-visual-kyc
kubectl get nodes
```

Build images:

```powershell
docker build -t visual-kyc-api:latest --build-arg TARGET=server .\api
docker build -t visual-kyc-worker:latest --build-arg TARGET=worker .\api
docker build -t visual-kyc-inference:latest .\inference
```

Load images into kind:

```powershell
kind load docker-image visual-kyc-api:latest --name visual-kyc
kind load docker-image visual-kyc-worker:latest --name visual-kyc
kind load docker-image visual-kyc-inference:latest --name visual-kyc
```

If `kind load` fails with disk space issue, clean temp files:

```powershell
Remove-Item -Recurse -Force "$env:TEMP\images-tar*" -ErrorAction SilentlyContinue
docker builder prune -af
docker image prune -f
```

Install/upgrade Helm chart:

```powershell
helm upgrade --install visual-kyc .\helm\visual-kyc
```

Create Kafka topics:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_create_topics_windows.ps1
```

Check pods:

```powershell
kubectl get pods
```

Expose API locally:

```powershell
kubectl port-forward svc/api 8080:8080
```

Open another PowerShell and test:

```powershell
curl.exe http://localhost:8080/health
```

---

## 9. Switch inference mode

### Stable local demo mode

Use mock inference for a stable Kubernetes demo:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_use_mock_inference_windows.ps1
```

This sets:

```text
EMBEDDING_MODE=mock
LIVENESS_REQUIRED=false
FAIL_IF_NO_FACE=false
```

### Real ArcFace mode

Use only after downloading the ONNX model and giving Docker enough memory.

Download model:

```powershell
cd inference
uv sync
uv run python scripts/download_models.py
cd ..
```

Rebuild and load inference image:

```powershell
docker build -t visual-kyc-inference:latest .\inference
kind load docker-image visual-kyc-inference:latest --name visual-kyc
```

Switch to ArcFace:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_use_arcface_inference_windows.ps1
```

This sets:

```text
EMBEDDING_MODE=arcface
LIVENESS_REQUIRED=true
FAIL_IF_NO_FACE=true
ARCFACE_MODEL_PATH=/app/models/w600k_r50.onnx
```

---

## 10. Debug Kubernetes worker

If status is stuck at `ACCEPTED`, check worker:

```powershell
kubectl logs deployment/worker --tail=160
```

Healthy worker logs look like:

```text
worker started
events polled
job received
job completed
```

Force fresh worker consumer group:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_force_worker_consume_windows.ps1
```

If inference connection fails:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_debug_inference_windows.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_use_mock_inference_windows.ps1
```

---

## 11. Make backend live for free using Cloudflare Tunnel

Keep Kubernetes API port-forward running:

```powershell
kubectl port-forward svc/api 8080:8080
```

In another PowerShell:

```powershell
cloudflared tunnel --url http://localhost:8080
```

Or use helper script:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\live_api_cloudflare_windows.ps1
```

Cloudflare will print a URL like:

```text
https://random-name.trycloudflare.com
```

Paste this URL into the frontend `API Base URL` field.

Important: Quick Tunnels are for testing/demo. For production, use a named Cloudflare Tunnel or a cloud load balancer.

---

## 12. Deploy frontend to GitHub Pages

The frontend lives in:

```text
frontend/
```

The GitHub Actions workflow lives in:

```text
.github/workflows/deploy-frontend.yml
```

Push code to GitHub:

```powershell
git init
git add .
git commit -m "initial visual kyc full stack project"
git branch -M main
git remote add origin https://github.com/YOUR_USERNAME/YOUR_REPO.git
git push -u origin main
```

In GitHub:

```text
Repository → Settings → Pages → Source → GitHub Actions
```

Then go to:

```text
Repository → Actions → Deploy frontend to GitHub Pages
```

After the workflow succeeds, your frontend URL will look like:

```text
https://YOUR_USERNAME.github.io/YOUR_REPO/
```

Open it, paste your Cloudflare Tunnel API URL, get token, and test enroll/verify.

---

## 13. Separation of frontend and backend

GitHub Pages hosts only static files:

```text
frontend/index.html
frontend/styles.css
frontend/app.js
```

Backend still runs separately:

```text
Local Kubernetes + Cloudflare Tunnel
or
GKE / Oracle / another Kubernetes cluster
```

Do not expect GitHub Pages to run Kafka, Qdrant, Redis, Go API, or Python inference.

---

## 14. Suggested GitHub repository layout

```text
visual-identity-kyc-final/
  api/
  inference/
  frontend/
  helm/
  scripts/
  docs/
  monitoring/
  .github/workflows/deploy-frontend.yml
  .gitignore
  README.md
  working.md
```

Do not commit:

```text
.env
Self_image1.jpg
inference/models/*.onnx
local data folders
```

The `.gitignore` already protects these.

---

## 15. Recruiter demo script

1. Open GitHub Pages frontend.
2. Paste Cloudflare Tunnel API URL.
3. Click `Check API Health`.
4. Click `Get Demo Token`.
5. Upload image and click `Enroll user`.
6. Click `Poll until done`.
7. Repeat enrollment to show `ALREADY_EXISTS`.
8. Use Verify tab to show `MATCHED`.
9. Show architecture panel and explain async flow.
10. Open Grafana locally if needed:

```text
http://localhost:3000
admin / admin
```

---

## 16. Going to real cloud later

Recommended beginner path:

```text
Local kind + Helm → Cloudflare Tunnel demo → GKE using free credits
```

For GKE later, you will need:

- A container registry for images
- Real image names in `helm/visual-kyc/values-prod.yaml`
- Ingress or LoadBalancer
- Strong secrets
- Strict CORS origins
- Real persistent storage
- Named Cloudflare Tunnel or domain

For now, the free demo path is GitHub Pages + Cloudflare Quick Tunnel.

---

# One-click Windows runner

Use this when you do not want to run every command manually.

## Local Kubernetes demo, stable mode

```powershell
cd C:\Users\sorab\Desktop\visual-identity-kyc-final
powershell -ExecutionPolicy Bypass -File .\scripts\one_click_windows.ps1 -Target k8s -InferenceMode mock -BuildImages -StartFrontend
```

This does:

```text
1. Selects the kind cluster context
2. Builds API + worker images
3. Loads API + worker images into kind
4. Deploys/updates Helm chart
5. Uses mock 512D vectors but still checks face presence
6. Creates Kafka topics
7. Restarts API/worker
8. Starts kubectl port-forward on localhost:8080
9. Opens frontend/index.html
```

## Local Kubernetes demo + public Cloudflare API URL

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\one_click_windows.ps1 -Target k8s -InferenceMode mock -BuildImages -StartFrontend -StartTunnel
```

A new PowerShell window will show a URL like:

```text
https://something.trycloudflare.com
```

Paste that URL into the frontend **API Base URL** field.

## Switch to real ArcFace mode

Use this only after the stable mock pipeline works.

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_use_arcface_inference_windows.ps1 -BuildAndLoadImage
```

This downloads/checks the model, builds the inference image, loads it into kind, applies `values-arcface.yaml`, and restarts inference/worker.

Important: `ACCEPTED` is not the final decision. It means the API queued the job. Click **Poll until done** or check `/kyc/status/{transaction_id}` until status becomes `COMPLETED` or `FAILED`.

## Cloudflare tunnel troubleshooting

Use:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\live_api_cloudflare_windows.ps1
```

The script intentionally uses:

```text
--protocol http2
--no-autoupdate
--config <temporary empty config>
```

Why:

- `--protocol http2` avoids networks that block QUIC/UDP.
- The empty config avoids accidentally loading an old named-tunnel config that asks for `cert.pem`.
- Quick Tunnels do not need a Cloudflare account or DNS setup.

If you still see:

```text
lookup region1.v2.argotunnel.com: i/o timeout
```

then your Wi-Fi/router/DNS is blocking Cloudflare lookup. Try:

```text
1. Change DNS to 1.1.1.1 or 8.8.8.8
2. Try mobile hotspot
3. Disable VPN/proxy temporarily
4. Retry the script
```

## Frontend live on GitHub Pages

1. Push this repo to GitHub.
2. Go to **Repository → Settings → Pages**.
3. Choose **Source: GitHub Actions**.
4. The workflow `.github/workflows/deploy-frontend.yml` publishes only the `frontend/` folder.
5. Open the GitHub Pages URL.
6. Paste the Cloudflare Tunnel backend URL into **API Base URL**.
7. Click **Check API Health** and **Get Bearer Token**.

GitHub Pages hosts only the static UI. The backend still runs locally/kind/GKE and is exposed through Cloudflare Tunnel or a real cloud load balancer.

---

## Fix: verify stuck at ACCEPTED while enroll completes

If enroll completes but verify stays `ACCEPTED` and the verify transaction ID never appears in worker logs, rebuild the patched worker and reset only the local Kafka command topics:

```powershell
cd C:\Users\sorab\Desktop\visual-identity-kyc-final
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_fix_verify_worker_windows.ps1
```

What this does:

1. Rebuilds the worker image.
2. Loads the worker image into the `kind` cluster.
3. Upgrades the Helm release.
4. Deletes and recreates only `kyc_enroll` and `kyc_verify` topics.
5. Restarts a single worker.
6. Prints worker logs.

After running it, submit a **new** verify request. Do not reuse old `ACCEPTED` transaction IDs.

Expected worker logs for verify:

```text
consumer loop started topic=kyc_verify
job received topic=kyc_verify
job completed topic=kyc_verify
```

## Verify endpoint troubleshooting / fixed flow

If `/kyc/enroll` completes but `/kyc/verify` stays `ACCEPTED`, run the end-to-end verify fixer. It rebuilds API + worker when requested, explicitly sets Kafka topics, resets local Kafka topics, restarts API/worker, submits a verify request, and checks that the accepted response says `kafka_topic: kyc_verify`.

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_fix_verify_end_to_end_windows.ps1 -BuildImages -ImagePath .\Self_image1.jpg -InferenceMode arcface
```

Expected accepted response must include:

```json
{
  "type": "verify",
  "kafka_topic": "kyc_verify",
  "status": "ACCEPTED"
}
```

Then worker logs should show:

```text
consumer loop started topic=kyc_verify
job received topic=kyc_verify
job completed topic=kyc_verify
```


## Frontend verify debugging

If PowerShell `/kyc/verify` reaches worker but frontend `/kyc/verify` does not, check these in order:

1. In the frontend `API Base URL`, use exactly the same backend you watch in Kubernetes logs.
   - Local kind: `http://localhost:8080`
   - GitHub Pages: use your `https://...trycloudflare.com` URL. Do not use `http://localhost:8080` from GitHub Pages.
2. Refresh browser with cache clear: `Ctrl + Shift + R`.
3. Open DevTools -> Network -> click the `kyc/verify` request.
4. Confirm:
   - Request URL ends with `/kyc/verify`
   - Response contains `"type":"verify"`
   - Response contains `"kafka_topic":"kyc_verify"`
5. If the response does not contain `kafka_topic`, your browser is hitting an old backend image or old cached frontend.
6. Run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\debug_frontend_verify_windows.ps1
```

The updated frontend prints the expected endpoint and expected Kafka topic before submitting, and it has a finite poll timeout so it no longer waits forever silently.

## Hard reset when transaction stays ACCEPTED

If a transaction is accepted but never reaches the worker, reset only the local command queue and restart the worker:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_hard_reset_pipeline_windows.ps1 -BuildApiWorker -InferenceMode arcface
```

Use a valid DOB format: `YYYY-MM-DD`, for example `1990-01-21`. Invalid dates such as `1990-21-01` are rejected by the patched API.

Trace a transaction:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_trace_transaction_windows.ps1 -Txn txn_xxx
```

The worker no longer publishes processing/audit events back into `kyc_enroll` or `kyc_verify`. Those topics now remain command-only topics, so the worker will not repeatedly consume its own audit events.
