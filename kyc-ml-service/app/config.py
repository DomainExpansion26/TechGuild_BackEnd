"""Configuration for the KYC ML worker.

Environment variables are loaded from `.env` in this directory (and then the
process environment). The thresholds defined here drive the review-priority
decision in :mod:`worker`.
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv

# Load the service-local .env first, then let the normal dotenv lookup fill in
# anything that wasn't provided (existing environment variables always win).
_SERVICE_ROOT = Path(__file__).resolve().parent.parent
load_dotenv(_SERVICE_ROOT / ".env")
load_dotenv()

# Default model directory: `<service>/models`. In Docker this resolves to
# `/app/models`, which is where docker-compose mounts the downloaded weights.
DEFAULT_MODEL_DIR = str(_SERVICE_ROOT / "models")


def _env_str(name: str, default: str = "") -> str:
    value = os.getenv(name)
    return value.strip() if value is not None else default


def _env_int(name: str, default: int) -> int:
    raw = os.getenv(name)
    if raw is None or raw.strip() == "":
        return default
    try:
        return int(raw)
    except ValueError as exc:
        raise ValueError(f"{name} must be an integer, got {raw!r}") from exc


def _env_float(name: str, default: float) -> float:
    raw = os.getenv(name)
    if raw is None or raw.strip() == "":
        return default
    try:
        return float(raw)
    except ValueError as exc:
        raise ValueError(f"{name} must be a number, got {raw!r}") from exc


def _env_bool(name: str, default: bool = False) -> bool:
    raw = os.getenv(name)
    if raw is None or raw.strip() == "":
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


def normalize_database_url(url: str) -> str:
    """Normalize a PostgreSQL URI for psycopg2/libpq.

    Cloud providers hand out both ``postgres://`` (Heroku/Render style) and
    ``postgresql://`` URLs. libpq only understands the latter, so normalize the
    scheme while leaving the query string untouched (``sslmode=require``,
    ``channel_binding=require``, etc. are handled natively by libpq).
    """
    url = (url or "").strip()
    if not url:
        return ""

    if url.lower().startswith("postgres://"):
        url = "postgresql://" + url[len("postgres://") :]

    return url


@dataclass(frozen=True)
class Settings:
    """Runtime settings for the ML service."""

    # --- Redis ---
    redis_host: str = "localhost"
    redis_port: int = 6379
    redis_password: str = ""
    redis_username: str = ""
    redis_db: int = 0
    redis_ssl: bool = False
    # redis-py/libpq-style verification for cloud TLS endpoints. Use "none"
    # only for providers with self-signed certificates.
    redis_ssl_cert_reqs: str = "required"
    redis_connect_timeout: int = 5
    redis_socket_timeout: int = 15

    # --- PostgreSQL ---
    database_url: str = ""
    db_connect_timeout: int = 10

    # --- Redis Stream consumer ---
    kyc_stream: str = "kyc:verification:jobs"
    kyc_consumer_group: str = "kyc_ml_workers"
    kyc_consumer_name: str = ""
    kyc_batch_size: int = 10
    kyc_block_ms: int = 5000
    worker_reconnect_delay_seconds: int = 5

    # --- Decision thresholds ---
    # Face matching - OpenCV SFace cosine similarity. AUTO_PASS is the
    # vendor-calibrated "same person" line; REVIEW is the ambiguous lower
    # bound; REJECT is the confidently-different floor.
    threshold_face_auto_pass: float = 0.363
    threshold_face_review: float = 0.300
    threshold_face_reject: float = 0.240
    shadow_mode: bool = True
    threshold_live_pass: float = 0.90
    threshold_live_reject: float = 0.50
    threshold_ocr_pass: float = 0.80

    # --- Policy version (Phase 3) ---
    # Stamped on completed records so a decision can be traced back to the
    # exact policy values in force. The Go router reads the same key
    # (KYC_POLICY_VERSION, default phase3-v1); bump both at the same time.
    policy_version: str = "phase3-v1"

    # --- Aadhaar Secure QR verification (Phase 2.5) ---
    # Master switch: when False, QR verification is skipped entirely and the
    # worker behaves exactly like Phase 1.
    qr_decode_enabled: bool = True
    # Phase 2.5 is plumbing only: this flag is recorded for Phase 5 rollout and
    # does not influence routing yet.
    qr_verification_required: bool = False
    # Wall-clock upper bound for one QR decode. Prevents a slow decode from
    # blocking a job.
    qr_max_decode_seconds: float = 3.0

    # --- Mock model outputs (used only when ml_mode == "mock") ---
    mock_ocr_confidence: float = 0.95
    mock_face_similarity: float = 0.88
    mock_liveness_score: float = 0.96

    # --- Real model inference (Phase H) ---
    # ml_mode: "real" runs ONNX/PaddleOCR; "mock" keeps the deterministic
    # Phase F behaviour for CI/smoke tests that cannot fetch real documents.
    ml_mode: str = "real"
    model_dir: str = DEFAULT_MODEL_DIR
    ocr_backend: str = "paddleocr"
    ocr_lang: str = "en"
    ocr_use_gpu: bool = False

    face_detector_model: str = "face_detection_yunet_2023mar.onnx"
    face_recognizer_model: str = "face_recognition_sface_2021dec.onnx"
    face_detect_score_threshold: float = 0.80
    face_detect_nms_threshold: float = 0.30
    face_detect_top_k: int = 5000
    # Downscale very large uploads before detection; keeps CPU latency bounded.
    face_detect_max_side: int = 1600

    # --- Multi-reference face matching (Phase 5) ---
    # A reference face that fails ANY quality gate is skipped: it must never
    # contribute a similarity score (a tiny/blurry crop produces garbage
    # embeddings that could look "similar" to everything).
    # Smallest acceptable face box side (pixels, post-detection coords).
    face_min_reference_side: int = 80
    # YuNet detection confidence for the kept face row.
    face_min_detection_score: float = 0.80
    # Laplacian variance over the face box; lower = visibly blurred.
    face_min_sharpness: float = 80.0
    # Phase 5 divergence guard: when a non-Aadhaar reference wins, the raw
    # priority is capped at manual_review once the gap exceeds this threshold.
    second_reference_divergence_threshold: float = 0.25

    liveness_model: str = "MiniFASNetV2.onnx"
    liveness_use_gpu: bool = False
    # The bundled MiniFASNetV2 ONNX is the upstream (raw BGR 0-255) export:
    # softmax class 1 is "real". Set to 0 only if you replace the model with a
    # conversion whose class order starts with "live".
    liveness_live_index: int = 1
    # "none" => raw BGR [0, 255]; "div255" => [0, 1]; "pm1" => [-1, 1].
    liveness_normalize: str = "none"
    liveness_crop_scale: float = 2.7

    image_connect_timeout: int = 10
    image_download_timeout: int = 30
    max_image_bytes: int = 15 * 1024 * 1024

    @classmethod
    def from_env(cls) -> "Settings":
        settings = cls(
            redis_host=_env_str("REDIS_HOST", "localhost"),
            redis_port=_env_int("REDIS_PORT", 6379),
            redis_password=_env_str("REDIS_PASSWORD", ""),
            redis_username=_env_str("REDIS_USERNAME", ""),
            redis_db=_env_int("REDIS_DB", 0),
            redis_ssl=_env_bool("REDIS_SSL", False),
            redis_ssl_cert_reqs=_env_str("REDIS_SSL_CERT_REQS", "required").lower(),
            redis_connect_timeout=_env_int("REDIS_CONNECT_TIMEOUT", 5),
            redis_socket_timeout=_env_int("REDIS_SOCKET_TIMEOUT", 15),
            database_url=normalize_database_url(_env_str("DATABASE_URL", "")),
            db_connect_timeout=_env_int("DB_CONNECT_TIMEOUT", 10),
            kyc_stream=_env_str("KYC_STREAM", "kyc:verification:jobs"),
            kyc_consumer_group=_env_str("KYC_CONSUMER_GROUP", "kyc_ml_workers"),
            kyc_consumer_name=_env_str("KYC_CONSUMER_NAME", ""),
            kyc_batch_size=_env_int("KYC_BATCH_SIZE", 10),
            kyc_block_ms=_env_int("KYC_BLOCK_MS", 5000),
            worker_reconnect_delay_seconds=_env_int(
                "WORKER_RECONNECT_DELAY_SECONDS", 5
            ),
            threshold_face_auto_pass=_env_float("THRESHOLD_FACE_AUTO_PASS", 0.363),
            threshold_face_review=_env_float("THRESHOLD_FACE_REVIEW", 0.300),
            threshold_face_reject=_env_float("THRESHOLD_FACE_REJECT", 0.240),
            shadow_mode=_env_bool("SHADOW_MODE", True),
            threshold_live_pass=_env_float("THRESHOLD_LIVE_PASS", 0.90),
            threshold_live_reject=_env_float("THRESHOLD_LIVE_REJECT", 0.50),
            threshold_ocr_pass=_env_float("THRESHOLD_OCR_PASS", 0.80),
            policy_version=_env_str("KYC_POLICY_VERSION", "phase3-v1"),
            qr_decode_enabled=_env_bool("QR_DECODE_ENABLED", True),
            qr_verification_required=_env_bool("QR_VERIFICATION_REQUIRED", False),
            qr_max_decode_seconds=_env_float("QR_MAX_DECODE_SECONDS", 3.0),
            mock_ocr_confidence=_env_float("MOCK_OCR_CONFIDENCE", 0.95),
            mock_face_similarity=_env_float("MOCK_FACE_SIMILARITY", 0.88),
            mock_liveness_score=_env_float("MOCK_LIVENESS_SCORE", 0.96),
            ml_mode=_env_str("ML_MODE", "real").lower(),
            model_dir=_env_str("MODEL_DIR", DEFAULT_MODEL_DIR),
            ocr_backend=_env_str("OCR_BACKEND", "paddleocr").lower(),
            ocr_lang=_env_str("OCR_LANG", "en"),
            ocr_use_gpu=_env_bool("OCR_USE_GPU", False),
            face_detector_model=_env_str(
                "FACE_DETECTOR_MODEL", "face_detection_yunet_2023mar.onnx"
            ),
            face_recognizer_model=_env_str(
                "FACE_RECOGNIZER_MODEL", "face_recognition_sface_2021dec.onnx"
            ),
            face_detect_score_threshold=_env_float(
                "FACE_DETECT_SCORE_THRESHOLD", 0.80
            ),
            face_detect_nms_threshold=_env_float("FACE_DETECT_NMS_THRESHOLD", 0.30),
            face_detect_top_k=_env_int("FACE_DETECT_TOP_K", 5000),
            face_detect_max_side=_env_int("FACE_DETECT_MAX_SIDE", 1600),
            face_min_reference_side=_env_int("FACE_MIN_REFERENCE_SIDE", 80),
            face_min_detection_score=_env_float(
                "FACE_MIN_DETECTION_SCORE", 0.80
            ),
            face_min_sharpness=_env_float("FACE_MIN_SHARPNESS", 80.0),
            second_reference_divergence_threshold=_env_float(
                "SECOND_REF_DIVERGENCE_THRESHOLD", 0.25
            ),
            liveness_model=_env_str("LIVENESS_MODEL", "MiniFASNetV2.onnx"),
            liveness_use_gpu=_env_bool("LIVENESS_USE_GPU", False),
            liveness_live_index=_env_int("LIVENESS_LIVE_INDEX", 1),
            liveness_normalize=_env_str("LIVENESS_NORMALIZE", "none").lower(),
            liveness_crop_scale=_env_float("LIVENESS_CROP_SCALE", 2.7),
            image_connect_timeout=_env_int("IMAGE_CONNECT_TIMEOUT", 10),
            image_download_timeout=_env_int("IMAGE_DOWNLOAD_TIMEOUT", 30),
            max_image_bytes=_env_int("MAX_IMAGE_BYTES", 15 * 1024 * 1024),
        )
        settings.validate()
        return settings

    def validate(self) -> None:
        """Fail fast on nonsensical threshold combinations."""
        for name, value in (
            ("THRESHOLD_FACE_AUTO_PASS", self.threshold_face_auto_pass),
            ("THRESHOLD_FACE_REVIEW", self.threshold_face_review),
            ("THRESHOLD_FACE_REJECT", self.threshold_face_reject),
            ("THRESHOLD_LIVE_PASS", self.threshold_live_pass),
            ("THRESHOLD_LIVE_REJECT", self.threshold_live_reject),
            ("THRESHOLD_OCR_PASS", self.threshold_ocr_pass),
            ("MOCK_OCR_CONFIDENCE", self.mock_ocr_confidence),
            ("MOCK_FACE_SIMILARITY", self.mock_face_similarity),
            ("MOCK_LIVENESS_SCORE", self.mock_liveness_score),
        ):
            if not 0.0 <= value <= 1.0:
                raise ValueError(f"{name} must be between 0 and 1, got {value}")

        if not (
            self.threshold_face_auto_pass
            > self.threshold_face_review
            > self.threshold_face_reject
        ):
            raise ValueError(
                "THRESHOLD_FACE_AUTO_PASS must be greater than "
                "THRESHOLD_FACE_REVIEW, which must be greater than "
                "THRESHOLD_FACE_REJECT"
            )
        if self.threshold_live_reject >= self.threshold_live_pass:
            raise ValueError(
                "THRESHOLD_LIVE_REJECT must be lower than THRESHOLD_LIVE_PASS"
            )

        if self.ml_mode not in {"real", "mock"}:
            raise ValueError(
                f"ML_MODE must be 'real' or 'mock', got {self.ml_mode!r}"
            )
        if self.ocr_backend not in {"paddleocr", "none"}:
            raise ValueError(
                "OCR_BACKEND must be 'paddleocr' or 'none', "
                f"got {self.ocr_backend!r}"
            )
        if self.liveness_normalize not in {"none", "div255", "pm1"}:
            raise ValueError(
                "LIVENESS_NORMALIZE must be one of none/div255/pm1, "
                f"got {self.liveness_normalize!r}"
            )
        if not 0 <= self.liveness_live_index <= 2:
            raise ValueError("LIVENESS_LIVE_INDEX must be between 0 and 2")
        if self.liveness_crop_scale <= 0:
            raise ValueError("LIVENESS_CROP_SCALE must be greater than zero")
        # `not (x > 0)` also rejects NaN, which `x <= 0` would let through.
        if not (self.qr_max_decode_seconds > 0):
            raise ValueError("QR_MAX_DECODE_SECONDS must be greater than zero")
        if self.face_detect_max_side < 160:
            raise ValueError("FACE_DETECT_MAX_SIDE must be at least 160")

        # --- Phase 5 multi-reference gates ---
        # `not (x > 0)` / `not (0.0 <= x <= 1.0)` patterns reject NaN as well as
        # out-of-range values.
        if not (self.face_min_reference_side > 0):
            raise ValueError(
                "FACE_MIN_REFERENCE_SIDE must be greater than zero"
                " (got %r)" % (self.face_min_reference_side,)
            )
        if not 0.0 <= self.face_min_detection_score <= 1.0:
            raise ValueError(
                "FACE_MIN_DETECTION_SCORE must be between 0 and 1"
                " (got %r)" % (self.face_min_detection_score,)
            )
        if not (self.face_min_sharpness >= 0):
            raise ValueError(
                "FACE_MIN_SHARPNESS must be greater than or equal to zero"
                " (got %r)" % (self.face_min_sharpness,)
            )
        if not 0.0 <= self.second_reference_divergence_threshold <= 1.0:
            raise ValueError(
                "SECOND_REF_DIVERGENCE_THRESHOLD must be between 0 and 1"
                " (got %r)" % (self.second_reference_divergence_threshold,)
            )
        if self.image_connect_timeout <= 0 or self.image_download_timeout <= 0:
            raise ValueError("Image download timeouts must be greater than zero")
        if self.max_image_bytes <= 0:
            raise ValueError("MAX_IMAGE_BYTES must be greater than zero")

        if self.kyc_batch_size <= 0:
            raise ValueError("KYC_BATCH_SIZE must be greater than zero")
        if self.kyc_block_ms <= 0:
            raise ValueError("KYC_BLOCK_MS must be greater than zero")
        if self.redis_ssl_cert_reqs not in {"none", "optional", "required"}:
            raise ValueError(
                "REDIS_SSL_CERT_REQS must be one of none/optional/required, "
                f"got {self.redis_ssl_cert_reqs!r}"
            )


settings = Settings.from_env()
