$ErrorActionPreference = "Stop"
Write-Host "Serving frontend at http://localhost:5173"
python -m http.server 5173 -d frontend
