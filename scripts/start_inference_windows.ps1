$ErrorActionPreference = "Stop"
Set-Location "$PSScriptRoot\..\inference"
$env:EMBEDDING_MODE = "arcface"
.\.venv\Scripts\uvicorn.exe app.main:app --host 0.0.0.0 --port 8001
