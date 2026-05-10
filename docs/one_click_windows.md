# One-click Windows scripts

## Main script

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\one_click_windows.ps1 -Target k8s -InferenceMode mock -BuildImages -StartFrontend -StartTunnel
```

### Common options

| Option | Meaning |
|---|---|
| `-Target k8s` | Run local Kubernetes/kind path |
| `-Target docker` | Run Docker Compose path |
| `-InferenceMode mock` | Stable local mode with deterministic vectors and face-presence validation |
| `-InferenceMode arcface` | Real ArcFace ONNX inference mode |
| `-BuildImages` | Rebuild API and worker images, then load them into kind |
| `-LoadInference` | Also rebuild/load the large inference image |
| `-StartFrontend` | Open `frontend/index.html` |
| `-StartTunnel` | Start Cloudflare Quick Tunnel in another PowerShell window |
| `-CreateKind` | Create kind cluster if missing |
| `-Clean` | Remove previous release first |

## Safe default

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\one_click_windows.ps1 -Target k8s -InferenceMode mock -BuildImages -StartFrontend
```

## Public demo default

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\one_click_windows.ps1 -Target k8s -InferenceMode mock -BuildImages -StartFrontend -StartTunnel
```

## Real ArcFace mode

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\k8s_use_arcface_inference_windows.ps1 -BuildAndLoadImage
```

Run ArcFace only after the mock pipeline works end-to-end.
