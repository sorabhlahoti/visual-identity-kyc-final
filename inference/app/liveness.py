from dataclasses import dataclass
import cv2
import numpy as np


@dataclass
class LivenessResult:
    passed: bool
    score: float
    reason: str
    anti_spoofing: str = "heuristic_liveness_v1"


def evaluate_liveness(image_bgr: np.ndarray, face_bgr: np.ndarray) -> LivenessResult:
    """Lightweight anti-spoof/liveness signal.

    This is intentionally dependency-light so the project runs on Windows with uv.
    It checks blur, exposure, texture and face size. For production you should replace
    or combine it with a trained anti-spoof model such as Silent-Face-Anti-Spoofing.
    """
    if image_bgr is None or face_bgr is None or face_bgr.size == 0:
        return LivenessResult(False, 0.0, "no_valid_face_region")

    gray = cv2.cvtColor(face_bgr, cv2.COLOR_BGR2GRAY)
    full_h, full_w = image_bgr.shape[:2]
    face_h, face_w = face_bgr.shape[:2]
    face_area_ratio = (face_h * face_w) / max(float(full_h * full_w), 1.0)

    blur_var = float(cv2.Laplacian(gray, cv2.CV_64F).var())
    brightness = float(gray.mean())
    contrast = float(gray.std())

    edges = cv2.Canny(gray, 80, 160)
    edge_density = float(np.count_nonzero(edges)) / max(float(edges.size), 1.0)

    blur_score = _clamp((blur_var - 20.0) / 180.0)
    exposure_score = _clamp(1.0 - abs(brightness - 127.0) / 127.0)
    contrast_score = _clamp((contrast - 20.0) / 50.0)
    texture_score = _clamp(edge_density / 0.18)
    size_score = _clamp((face_area_ratio - 0.03) / 0.22)

    score = (0.30 * blur_score) + (0.20 * exposure_score) + (0.20 * contrast_score) + (0.20 * texture_score) + (0.10 * size_score)
    passed = score >= 0.52

    reasons = []
    if blur_score < 0.35:
        reasons.append("image_blurry")
    if exposure_score < 0.35:
        reasons.append("bad_exposure")
    if contrast_score < 0.25:
        reasons.append("low_contrast")
    if texture_score < 0.25:
        reasons.append("low_texture_possible_screen_or_print")
    if size_score < 0.25:
        reasons.append("face_too_small")
    if not reasons:
        reasons.append("liveness_heuristics_passed")

    return LivenessResult(bool(passed), round(float(score), 4), ",".join(reasons))


def _clamp(value: float) -> float:
    return float(max(0.0, min(1.0, value)))
