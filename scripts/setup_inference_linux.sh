#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../inference"
if ! command -v uv >/dev/null 2>&1; then
  curl -LsSf https://astral.sh/uv/install.sh | sh
  export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$PATH"
fi
uv venv .venv --python 3.11
uv sync
.venv/bin/python scripts/download_models.py
