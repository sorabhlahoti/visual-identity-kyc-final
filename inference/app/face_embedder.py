import hashlib
import math
import os
from pathlib import Path
from typing import List, Tuple, Dict, Any

import cv2
import numpy as np
import onnxruntime as ort

from app.liveness import evaluate_liveness

FACE_DIM = 512
ROOT = Path(__file__).resolve().parents[1]
DEFAULT_MODEL_PATH = ROOT / "models" / "w600k_r50.onnx"


class FaceEmbeddingError(RuntimeError):
    pass


class ArcFaceEmbedder:
    def __init__(self) -> None:
        self.mode = os.getenv("EMBEDDING_MODE", "arcface").lower().strip()
        self.fail_if_no_face = os.getenv("FAIL_IF_NO_FACE", "true").lower() in {"1", "true", "yes"}
        self.model_path = Path(os.getenv("ARCFACE_MODEL_PATH", str(DEFAULT_MODEL_PATH)))
        self.session = None
        self.input_name = None
        self.output_name = None
        self.face_cascade = cv2.CascadeClassifier(
            str(Path(cv2.data.haarcascades) / "haarcascade_frontalface_default.xml")
        )
        if self.mode == "arcface":
            self._load_model()

    def _load_model(self) -> None:
        if not self.model_path.exists():
            raise FaceEmbeddingError(
                f"ArcFace model not found at {self.model_path}. Run: uv run python scripts/download_models.py"
            )
        providers = ["CPUExecutionProvider"]
        self.session = ort.InferenceSession(str(self.model_path), providers=providers)
        self.input_name = self.session.get_inputs()[0].name
        self.output_name = self.session.get_outputs()[0].name

    def embed(self, image_bytes: bytes) -> Tuple[List[float], str, Dict[str, Any]]:
        if self.mode == "mock":
            return self._mock_embedding(image_bytes), "mock-512D-for-tests-only", {"passed": True, "score": 1.0, "reason": "mock_mode", "anti_spoofing": "mock"}
        image = self._decode(image_bytes)
        face = self._detect_and_crop(image)
        live = evaluate_liveness(image, face)
        require_liveness = os.getenv("LIVENESS_REQUIRED", "true").lower() in {"1", "true", "yes"}
        if require_liveness and not live.passed:
            raise FaceEmbeddingError(f"liveness/anti-spoof check failed: {live.reason}; score={live.score}")
        tensor = self._preprocess(face)
        output = self.session.run([self.output_name], {self.input_name: tensor})[0]
        emb = np.asarray(output).reshape(-1).astype(np.float32)
        if emb.shape[0] != FACE_DIM:
            raise FaceEmbeddingError(f"model returned {emb.shape[0]} dims, expected 512")
        emb = emb / max(float(np.linalg.norm(emb)), 1e-12)
        return emb.astype(float).tolist(), f"arcface-onnx:{self.model_path.name}", live.__dict__

    def _decode(self, image_bytes: bytes) -> np.ndarray:
        arr = np.frombuffer(image_bytes, dtype=np.uint8)
        image = cv2.imdecode(arr, cv2.IMREAD_COLOR)
        if image is None:
            raise FaceEmbeddingError("invalid image bytes")
        return image

    def _detect_and_crop(self, image: np.ndarray) -> np.ndarray:
        gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
        faces = self.face_cascade.detectMultiScale(gray, scaleFactor=1.1, minNeighbors=5, minSize=(60, 60))
        if len(faces) == 0:
            if self.fail_if_no_face:
                raise FaceEmbeddingError("no face detected; upload a clear live photograph")
            return image
        # choose largest face
        x, y, w, h = max(faces, key=lambda rect: rect[2] * rect[3])
        pad = int(0.25 * max(w, h))
        x1 = max(0, x - pad)
        y1 = max(0, y - pad)
        x2 = min(image.shape[1], x + w + pad)
        y2 = min(image.shape[0], y + h + pad)
        return image[y1:y2, x1:x2]

    def _preprocess(self, face: np.ndarray) -> np.ndarray:
        face = cv2.resize(face, (112, 112), interpolation=cv2.INTER_AREA)
        # InsightFace ArcFace models use BGR images normalized to [-1, 1].
        face = face.astype(np.float32)
        face = (face - 127.5) / 127.5
        face = np.transpose(face, (2, 0, 1))[None, :, :, :]
        return face.astype(np.float32)

    def _mock_embedding(self, image_bytes: bytes) -> List[float]:
        # Deterministic local smoke-test vector. Do not use for biometric matching.
        digest = hashlib.sha256(image_bytes).digest()
        vals = []
        counter = 0
        while len(vals) < FACE_DIM:
            block = hashlib.sha256(digest + counter.to_bytes(4, "little")).digest()
            for i in range(0, len(block), 4):
                raw = int.from_bytes(block[i : i + 4], "little")
                vals.append((raw / 2**32) * 2 - 1)
                if len(vals) == FACE_DIM:
                    break
            counter += 1
        norm = math.sqrt(sum(x * x for x in vals)) or 1.0
        return [float(x / norm) for x in vals]
