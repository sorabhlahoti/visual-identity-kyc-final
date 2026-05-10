FROM python:3.11-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends curl libglib2.0-0 libgl1 && rm -rf /var/lib/apt/lists/*
RUN curl -LsSf https://astral.sh/uv/install.sh | sh
ENV PATH="/root/.local/bin:$PATH"
COPY pyproject.toml ./
RUN uv venv .venv --python 3.11 && uv sync
COPY app ./app
COPY scripts ./scripts
COPY models ./models
# The ArcFace model is intentionally not committed. Build with a downloaded models/w600k_r50.onnx
# or run: uv run python scripts/download_models.py before starting.
EXPOSE 8001
CMD [".venv/bin/uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8001"]
