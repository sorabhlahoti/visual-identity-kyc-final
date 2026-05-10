from __future__ import annotations

import shutil
import zipfile
from pathlib import Path

import requests

ROOT = Path(__file__).resolve().parents[1]
MODELS = ROOT / "models"
ZIP_PATH = MODELS / "buffalo_l.zip"
TARGET = MODELS / "w600k_r50.onnx"
URL = "https://github.com/deepinsight/insightface/releases/download/v0.7/buffalo_l.zip"


def main() -> None:
    MODELS.mkdir(parents=True, exist_ok=True)
    if TARGET.exists() and TARGET.stat().st_size > 10_000_000:
        print(f"Model already exists: {TARGET}")
        return

    print(f"Downloading InsightFace buffalo_l model archive from {URL}")
    with requests.get(URL, stream=True, timeout=60) as resp:
        resp.raise_for_status()
        total = int(resp.headers.get("content-length", 0))
        downloaded = 0
        with ZIP_PATH.open("wb") as f:
            for chunk in resp.iter_content(chunk_size=1024 * 1024):
                if not chunk:
                    continue
                f.write(chunk)
                downloaded += len(chunk)
                if total:
                    pct = downloaded * 100 / total
                    print(f"\r{pct:5.1f}%", end="")
    print("\nExtracting w600k_r50.onnx ...")
    with zipfile.ZipFile(ZIP_PATH) as zf:
        member = next((m for m in zf.namelist() if m.endswith("w600k_r50.onnx")), None)
        if member is None:
            raise RuntimeError("w600k_r50.onnx not found in model archive")
        with zf.open(member) as src, TARGET.open("wb") as dst:
            shutil.copyfileobj(src, dst)
    print(f"Saved {TARGET}")
    print("You can delete models/buffalo_l.zip after setup if disk space matters.")


if __name__ == "__main__":
    main()
