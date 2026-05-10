# Frontend + GitHub Pages Guide

The frontend is intentionally static HTML/CSS/JS in `frontend/`.

Why static instead of React/Vite for the demo?

- No Node install required for recruiters.
- No dependency conflict on Windows.
- GitHub Pages can serve it directly.
- The UI can still call your live backend through Cloudflare Tunnel.

## How frontend and backend are separated

```text
GitHub Pages: hosts only frontend static files
Cloudflare Tunnel / Kubernetes / GKE: hosts backend API
```

GitHub Pages cannot run Kafka, Qdrant, Redis, Go services, or Python inference. It only hosts the browser UI. The backend must run somewhere else.

## Backend URL in UI

The UI has an `API Base URL` field. Use:

- Local Docker/kind: `http://localhost:8080`
- Cloudflare Tunnel: `https://something.trycloudflare.com`
- GKE ingress/load balancer: `https://your-api-domain.com`

The value is saved in browser localStorage.
