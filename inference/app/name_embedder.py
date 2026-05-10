import hashlib
import math
import re
import unicodedata
from typing import List

DIM = 768


def _normalize_name(name: str) -> str:
    text = unicodedata.normalize("NFKD", name or "")
    text = "".join(ch for ch in text if not unicodedata.combining(ch))
    text = text.lower().strip()
    text = re.sub(r"[^a-z0-9 ]+", " ", text)
    text = re.sub(r"\s+", " ", text)
    return text


def _ngrams(text: str):
    padded = f"  {text}  "
    for n in (2, 3, 4):
        for i in range(max(0, len(padded) - n + 1)):
            gram = padded[i : i + n]
            if gram.strip():
                yield gram


def name_embedding_768(name: str) -> List[float]:
    """Deterministic 768D char n-gram embedding for names.

    This avoids storing raw names and gives stable fuzzy matching for spelling variants.
    It is deliberately dependency-light for Windows/uv setup. For a heavier semantic model,
    replace this function with an ONNX/SentenceTransformer model that returns 768D.
    """
    text = _normalize_name(name)
    vec = [0.0] * DIM
    if not text:
        return vec

    tokens = text.split()
    features = list(_ngrams(text)) + tokens
    for feat in features:
        digest = hashlib.blake2b(feat.encode("utf-8"), digest_size=16).digest()
        idx = int.from_bytes(digest[:4], "little") % DIM
        sign = 1.0 if digest[4] % 2 == 0 else -1.0
        weight = 1.0 + min(len(feat), 8) / 8.0
        vec[idx] += sign * weight

    # Add a weak token-order signal without preserving the raw name.
    for pos, token in enumerate(tokens):
        digest = hashlib.sha256(f"{pos}:{token}".encode("utf-8")).digest()
        idx = int.from_bytes(digest[:4], "little") % DIM
        vec[idx] += 0.5

    norm = math.sqrt(sum(x * x for x in vec))
    if norm == 0:
        return vec
    return [float(x / norm) for x in vec]
