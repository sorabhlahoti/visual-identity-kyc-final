# Visual KYC Frontend

Static recruiter demo UI for GitHub Pages. It has no build step and no Node dependency.

## Local test

Open `frontend/index.html` in your browser, or serve it with:

```powershell
python -m http.server 5173 -d frontend
```

Then open `http://localhost:5173`.

## GitHub Pages

This folder is deployed by `.github/workflows/deploy-frontend.yml`.

After deployment, paste your backend API URL into the UI. For a free demo, generate that backend URL with Cloudflare Tunnel:

```powershell
kubectl port-forward svc/api 8080:8080
cloudflared tunnel --url http://localhost:8080
```
