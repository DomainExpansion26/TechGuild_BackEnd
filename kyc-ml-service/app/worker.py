"""KYC ML worker.

Consumes verification jobs from the Redis Stream produced by the Go backend,
runs OCR / face-match / liveness inference, persists the results as JSONB on
``verification_records``, and acknowledges the job.

Phase H uses real models:

* OCR        - PaddleOCR (CPU), English/Latin script by default
* Face match - OpenCV YuNet detector + SFace recognizer (ONNX)
* Liveness   - MiniFASNetV2 ONNX through onnxruntime

Set ``ML_MODE=mock`` to keep the deterministic Phase F scores for smoke tests
that use fake document URLs.

Run standalone with::

    python worker.py

The FastAPI app in :mod:`main` starts the same worker in a background thread.
"""

from __future__ import annotations

import json
import logging
import math
import os
import re
import socket
import threading
import time
from contextlib import contextmanager
from functools import lru_cache
from pathlib import Path
from typing import Any, Iterator
from urllib.parse import unquote, urlparse

import psycopg2
import redis
from redis.exceptions import ResponseError

from app.qr.aadhaar_qr import QRResult, QRVerificationError, verify_aadhaar_qr
from app.config import settings
from app.qr.qr_decode import decode_qr_text
from app.qr.cert_loader import TrustedCert, load_pinned_certs
from app.image_preprocess import preprocess_selfie_image
from app.validators.id_formats import is_valid_pan, find_aadhaar_in_text
from app.validators.document_match import (
    compare_business_documents,
    compare_documents,
    fold_ocr_confusables,
    normalise_gstin,
    normalise_ifsc,
    normalise_pan,
    normalise_registration_number,
)
from app.validators.verhoeff import is_valid_aadhaar

logger = logging.getLogger("kyc.worker")

PAYLOAD_FIELD = "payload"

# Review-priority values must match `models.VerificationReviewPriority` in the
# Go backend.
PRIORITY_AUTO_ELIGIBLE = "auto_eligible"
PRIORITY_MANUAL_REVIEW = "manual_review"
PRIORITY_AUTO_REJECT = "auto_reject"

ML_VENDOR_MOCK = "local-mock"
ML_MODEL_VERSION_MOCK = "mock-v1"
ML_VENDOR_REAL = "local-onnx"
# `verification_records.ml_model_version` is varchar(50); keep this concise.
ML_MODEL_VERSION_REAL = "yunet2023+sface2021+minifasnetv2+ppocr2.10"

# Backwards-compatible names for the default persistence path.
ML_VENDOR = ML_VENDOR_REAL
ML_MODEL_VERSION = ML_MODEL_VERSION_REAL

GOVT_DOC_TYPES = {
    "govt_id",
    "government_id",
    "id_card",
    "pan_card",
    "passport",
    "aadhaar",
    "driving_license",
}
SELFIE_DOC_TYPES = {"selfie", "live_selfie", "face"}

# Phase G business KYC: a business job uploads the ONE business document the
# user selected (a GST certificate, a PAN card, an incorporation certificate or
# a business registration certificate) plus a cancelled cheque, and has no
# selfie at all. The selected document is the only identity anchor; the cheque
# is the bank-details anchor.
BUSINESS_PRIMARY_DOC_TYPES = (
    "gst_certificate",
    "pan_card",
    "incorporation_certificate",
    "business_registration",
)
BUSINESS_DOC_TYPES = (*BUSINESS_PRIMARY_DOC_TYPES, "cancelled_cheque")
AADHAAR_DOC_TYPES = {"aadhaar", "aadhaar_card"}

# Document "type" strings the Go producer may send for each business slot.
_BUSINESS_DOC_ALIASES = {
    "pan_card": ("pan_card", "pan", "pan_document", "pan_card_image"),
    "gst_certificate": (
        "gst_certificate",
        "gst_cert",
        "gstin_certificate",
        "gst",
    ),
    "incorporation_certificate": (
        "incorporation_certificate",
        "incorporation_cert",
        "inc_cert",
        "certificate_of_incorporation",
        "coi",
    ),
    "business_registration": (
        "business_registration",
        "biz_reg",
        "business_reg_cert",
        "registration_certificate",
        "shop_act",
        "udyam",
        "msme",
    ),
    "cancelled_cheque": (
        "cancelled_cheque",
        "canceled_cheque",
        "cancelled_check",
        "canceled_check",
        "cheque",
        "check",
    ),
}

_BUSINESS_MISSING_ERRORS = {
    "pan_card": "missing_pan_card",
    "gst_certificate": "missing_gst_certificate",
    "incorporation_certificate": "missing_incorporation_certificate",
    "business_registration": "missing_business_registration",
    "cancelled_cheque": "missing_cancelled_cheque",
}

# No selectable business document was found at all (nothing was uploaded, or the
# document list carries only individual-KYC slots).
_BUSINESS_NO_DOCUMENT_ERROR = "missing_business_documents"


def _canonical_business_doc_type(doc_type: Any) -> str | None:
    """Map a document "type" string (or an alias) onto its canonical slot."""
    value = str(doc_type or "").strip().lower()
    if not value:
        return None
    for canonical, aliases in _BUSINESS_DOC_ALIASES.items():
        if value == canonical or value in aliases:
            return canonical
    return None


def _infer_business_doc_type(documents: Any) -> str | None:
    """The selected document type implied by an uploaded document list.

    The first document, in upload order, whose type is one of the four
    selectable business documents. Deterministic on purpose: with a producer
    that sends no explicit selection, the workload itself decides, and a
    cancelled cheque alone never counts as the selected document.
    """
    if not isinstance(documents, (list, tuple)):
        return None
    for document in documents:
        if not isinstance(document, dict):
            continue
        canonical = _canonical_business_doc_type(document.get("type"))
        if canonical in BUSINESS_PRIMARY_DOC_TYPES:
            return canonical
    return None


class JobPayloadError(ValueError):
    """Raised when a Redis stream message does not match the job contract."""


class JobProcessingError(RuntimeError):
    """Raised when a job cannot be completed (DB missing, row missing, etc.)."""


class ImageFetchError(RuntimeError):
    """Raised when a document URL cannot be fetched or decoded."""


# ---------------------------------------------------------------------------
# Connections
# ---------------------------------------------------------------------------


def build_redis_client() -> redis.Redis:
    """Create a Redis client from the service settings.

    Works with local Redis as well as managed cloud Redis. TLS is disabled by
    default; set ``REDIS_SSL=true`` only for endpoints/plans that require TLS
    (Redis Cloud Free tier does not support TLS).
    """
    kwargs: dict[str, Any] = {
        "host": settings.redis_host,
        "port": settings.redis_port,
        "password": settings.redis_password or None,
        "username": settings.redis_username or None,
        "db": settings.redis_db,
        "decode_responses": True,
        "socket_connect_timeout": settings.redis_connect_timeout,
        "socket_timeout": settings.redis_socket_timeout,
        "retry_on_timeout": True,
        "health_check_interval": 30,
    }

    if settings.redis_ssl:
        kwargs["ssl"] = True
        kwargs["ssl_cert_reqs"] = settings.redis_ssl_cert_reqs
        # "none" also disables hostname verification in redis-py.
        kwargs["ssl_check_hostname"] = settings.redis_ssl_cert_reqs != "none"

    return redis.Redis(**kwargs)


@contextmanager
def db_connection() -> Iterator[Any]:
    """Open a short-lived PostgreSQL connection with commit/rollback handling.

    A connection per operation keeps the worker resilient to transient Neon /
    network failures without maintaining a stale long-lived connection.
    """
    if not settings.database_url:
        raise JobProcessingError("DATABASE_URL is not configured")

    # The URI (including sslmode=require / channel_binding=require) is passed
    # straight to libpq, which handles TLS natively for cloud Postgres.
    conn = psycopg2.connect(
        settings.database_url,
        connect_timeout=settings.db_connect_timeout,
        application_name="kyc-ml-worker",
    )
    try:
        yield conn
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()


def ping_database() -> bool:
    """Return True when PostgreSQL accepts a trivial query."""
    try:
        with db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT 1")
                cur.fetchone()
        return True
    except Exception as exc:  # noqa: BLE001 - health check must never raise
        logger.debug("PostgreSQL health check failed: %s", exc)
        return False


# ---------------------------------------------------------------------------
# Payload + result helpers
# ---------------------------------------------------------------------------


def parse_job_payload(
    payload: str | bytes | dict[str, Any],
) -> dict[str, Any]:
    """Parse and validate the JSON payload published by the Go backend.

    Expected shape (see ``kycMLJob`` in ``verification_service.go``)::

        {
          "job_id": "kyc-job-...",
          "verification_id": "uuid",
          "user_id": "uuid",
          "type": "individual",
          "documents": [{"type": "selfie", "url": "https://..."}]
        }
    """
    if isinstance(payload, (bytes, bytearray)):
        try:
            payload = payload.decode("utf-8")
        except UnicodeDecodeError as exc:
            raise JobPayloadError("job payload is not valid UTF-8") from exc

    if isinstance(payload, str):
        try:
            data = json.loads(payload)
        except json.JSONDecodeError as exc:
            raise JobPayloadError(f"job payload is not valid JSON: {exc}") from exc
    elif isinstance(payload, dict):
        data = payload
    else:
        raise JobPayloadError(
            f"job payload must be a JSON object, got {type(payload).__name__}"
        )

    if not isinstance(data, dict):
        raise JobPayloadError(
            f"job payload must be a JSON object, got {type(data).__name__}"
        )

    verification_id = str(data.get("verification_id") or "").strip()
    if not verification_id:
        raise JobPayloadError("job payload is missing verification_id")

    raw_documents = data.get("documents") or []
    if not isinstance(raw_documents, list):
        raise JobPayloadError("job payload documents must be a list")

    documents: list[dict[str, str]] = []
    for raw_doc in raw_documents:
        if not isinstance(raw_doc, dict):
            continue
        documents.append(
            {
                "type": str(raw_doc.get("type") or "").lower(),
                "url": str(raw_doc.get("url") or ""),
            }
        )

    business_info = data.get("business_info") if isinstance(data.get("business_info"), dict) else {}
    bank_details = data.get("bank_details") if isinstance(data.get("bank_details"), dict) else {}
    return {
        "job_id": str(data.get("job_id") or ""),
        "verification_id": verification_id,
        "user_id": str(data.get("user_id") or ""),
        "type": str(data.get("type") or ""),
        "documents": documents,
        "business_info": {
            "gstin": str(business_info.get("gstin") or ""),
            "pan": str(business_info.get("pan") or ""),
            "reg_no": str(business_info.get("reg_no") or ""),
        },
        "bank_details": {
            "ifsc": str(bank_details.get("ifsc") or ""),
            "account_number": str(bank_details.get("account_number") or ""),
        },
        "selected_business_doc": str(data.get("selected_business_doc") or ""),
    }


def _as_float(value: Any, default: float = 0.0) -> float:
    try:
        number = float(value)
    except (TypeError, ValueError):
        return default
    # NaN/Infinity are not valid confidence scores and could otherwise satisfy
    # every threshold comparison (inf >= pass). Treat them as missing evidence.
    if not math.isfinite(number):
        return default
    return number


def _find_document(
    documents: list[dict[str, str]], preferred_types: set[str]
) -> dict[str, str] | None:
    """Return the first document whose type is in *preferred_types*."""
    for doc in documents:
        if doc.get("type") in preferred_types:
            return doc
    return None


def _first_document_of_type(
    documents: list[dict[str, str]], preferred_types: set[str]
) -> dict[str, str] | None:
    for doc in documents:
        if doc.get("type") in preferred_types:
            return doc
    return documents[0] if documents else None


def _safe_url(url: str) -> str:
    """Strip query strings/tokens before logging a document URL."""
    try:
        parsed = urlparse(url)
        if parsed.scheme:
            return f"{parsed.scheme}://{parsed.netloc}{parsed.path}"
        return parsed.path or "<empty>"
    except Exception:  # noqa: BLE001 - diagnostic helper
        return "<invalid-url>"


# ---------------------------------------------------------------------------
# Document fetching
# ---------------------------------------------------------------------------


def fetch_image(url: str) -> Any:
    """Fetch and decode a document image as a BGR ``numpy.ndarray``.

    Supports ``http(s)://`` (Cloudinary links from the Go API) and
    ``file://`` URLs (used by local integration tests).
    """
    import cv2
    import numpy as np

    if not url:
        raise ImageFetchError("document URL is empty")

    parsed = urlparse(url)
    max_bytes = settings.max_image_bytes

    if parsed.scheme == "file":
        path = Path(unquote(parsed.path))
        try:
            if path.stat().st_size > max_bytes:
                raise ImageFetchError(
                    f"image exceeds MAX_IMAGE_BYTES ({max_bytes} bytes)"
                )
            data = path.read_bytes()
        except ImageFetchError:
            raise
        except OSError as exc:
            raise ImageFetchError(f"cannot read local image: {exc}") from exc
    elif parsed.scheme in {"http", "https"}:
        import requests

        try:
            with requests.get(
                url,
                stream=True,
                timeout=(settings.image_connect_timeout, settings.image_download_timeout),
                headers={"User-Agent": "techguild-kyc-ml/1.0"},
            ) as response:
                response.raise_for_status()
                chunks: list[bytes] = []
                total = 0
                for chunk in response.iter_content(chunk_size=64 * 1024):
                    if not chunk:
                        continue
                    total += len(chunk)
                    if total > max_bytes:
                        raise ImageFetchError(
                            f"image exceeds MAX_IMAGE_BYTES ({max_bytes} bytes)"
                        )
                    chunks.append(chunk)
                data = b"".join(chunks)
        except ImageFetchError:
            raise
        except Exception as exc:  # noqa: BLE001 - normalize to a domain error
            raise ImageFetchError(
                f"download failed for {_safe_url(url)}: {exc}"
            ) from exc
    else:
        raise ImageFetchError(f"unsupported document URL scheme: {parsed.scheme!r}")

    if not data:
        raise ImageFetchError("document is empty")

    image = cv2.imdecode(np.frombuffer(data, dtype=np.uint8), cv2.IMREAD_COLOR)
    if image is None:
        raise ImageFetchError(
            f"could not decode image from {_safe_url(url)} (not a JPEG/PNG?)"
        )

    logger.debug("Fetched %s (%d bytes)", _safe_url(url), len(data))
    return image


class ImageCache:
    """Fetch each document URL at most once per job."""

    def __init__(self) -> None:
        self._items: dict[str, tuple[Any, Exception | None]] = {}

    def get(self, url: str) -> Any:
        if url not in self._items:
            try:
                self._items[url] = (fetch_image(url), None)
            except Exception as exc:  # noqa: BLE001 - cached for this job only
                self._items[url] = (None, exc)

        image, error = self._items[url]
        if error is not None:
            if isinstance(error, ImageFetchError):
                raise error
            raise ImageFetchError(str(error)) from error
        return image


# ---------------------------------------------------------------------------
# OCR helpers
# ---------------------------------------------------------------------------

# Aadhaar cards rarely print a "Name:" label: the holder's name is usually a
# bare line between "Government of India" and the DOB. OCR also glues words
# together ("PatelFujel Gulan"), so name extraction tries an explicit label
# first, then the lines around those two Aadhaar anchors.
_NAME_LABEL_RE = re.compile(
    r"(?:FULL\s*NAME|NAME|नाम|NAAM)\s*[:\-]?\s*([^\n]+)",
    re.IGNORECASE,
)
_AADHAAR_ANCHOR_RE = re.compile(r"government\s+of\s+india", re.IGNORECASE)
_DOB_LABEL_RE = re.compile(
    r"(?:DOB|D\.O\.B\.?|DATE\s+OF\s+BIRTH|BIRTH\s*DATE|जन्म\s*(?:तिथि)?)",
    re.IGNORECASE,
)
_DATE_RE = re.compile(
    r"\b(?:\d{4}[./-]\d{1,2}[./-]\d{1,2}"
    r"|[0-9xX]{1,2}[./-][0-9xX]{1,2}[./-][0-9xX]{2,4})\b"
)
_MASKED_DATE_RE = re.compile(
    r"\b(?=[0-9xX./-]*[xX])"
    r"[0-9xX]{1,2}[./-][0-9xX]{1,2}[./-][0-9xX]{2,4}\b"
)
_NON_ASCII_RE = re.compile(r"[^\x00-\x7F]")
# Words that appear next to a name on an ID but are never part of one.
_NAME_STOPWORDS = frozenset(
    {
        "aadhaar",
        "aadhar",
        "address",
        "authority",
        "birth",
        "code",
        "date",
        "dob",
        "enrolment",
        "enrollment",
        "female",
        "government",
        "identification",
        "india",
        "issue",
        "male",
        "name",
        "number",
        "of",
        "other",
        "pin",
        "signature",
        "the",
        "uidai",
        "unique",
        "valid",
        "vid",
        "year",
    }
)
# Separators are space/hyphen only. `\s` let the pattern run across a line
# break, so "2003\n2345 6789 0124" glued an unrelated 4-digit token onto the
# Aadhaar run and left eight unmasked digits in the stored text.
_AADHAAR_RE = re.compile(r"\b(\d{4})[ -]?(\d{4})[ -]?(\d{4})\b")
_PAN_RE = re.compile(r"\b[A-Z]{5}\d{4}[A-Z]\b")
_PAN_CANDIDATE_RE = re.compile(
    r"(?<![A-Za-z0-9])"
    r"([A-Za-z]{3}[\s-]*[ABCFGHJLPT][A-Za-z][\s-]*[0-9]{4}[\s-]*[A-Za-z])"
    r"(?![A-Za-z0-9])",
    re.IGNORECASE,
)
_LONG_DIGITS_RE = re.compile(r"\b\d{9,16}\b")


def _normalize_name_candidate(line: str) -> str:
    """Turn an OCR line into a comparable English name candidate.

    Non-ASCII text is dropped because mixed Devanagari/English lines otherwise
    break the ASCII-only name patterns. Glued OCR words such as ``PatelFujel``
    are split on lower-to-upper case boundaries.
    """
    line = _NON_ASCII_RE.sub(" ", line)
    line = re.sub(r"(?<=[a-z])(?=[A-Z])", " ", line)
    line = re.sub(r"[^A-Za-z .'-]", " ", line)
    line = re.sub(r"\s+", " ", line).strip(" .:-'")
    return line


def _is_plausible_name(candidate: str) -> bool:
    words = [word for word in candidate.split() if len(word) >= 2]
    if not words:
        return False
    if sum(char.isalpha() for char in candidate) < 4:
        return False
    if not any(len(word) >= 3 for word in words):
        return False
    if len(words) > 5:
        return False
    for word in words:
        lowered = word.lower()
        if lowered in _NAME_STOPWORDS:
            return False
        # Masked digits ("XXXX") and OCR repetition are not names.
        if len(word) >= 3 and len(set(lowered)) == 1:
            return False
    return True


def _name_from_label(text: str) -> str:
    match = _NAME_LABEL_RE.search(text)
    if not match:
        return ""
    candidate = _normalize_name_candidate(match.group(1))
    return candidate if _is_plausible_name(candidate) else ""


def _name_after_aadhaar_anchor(text: str) -> str:
    lines = [line.strip() for line in text.splitlines() if line.strip()]
    for index, line in enumerate(lines):
        if not _AADHAAR_ANCHOR_RE.search(line):
            continue
        for following in lines[index + 1 :]:
            candidate = _normalize_name_candidate(following)
            if _is_plausible_name(candidate):
                return candidate
            if _DOB_LABEL_RE.search(following) or _AADHAAR_RE.search(following):
                break
        break
    return ""


def _name_before_dob_label(text: str) -> str:
    lines = [line.strip() for line in text.splitlines() if line.strip()]
    for index, line in enumerate(lines):
        if not _DOB_LABEL_RE.search(line):
            continue
        for previous in reversed(lines[:index]):
            candidate = _normalize_name_candidate(previous)
            if _is_plausible_name(candidate):
                return candidate
            if _AADHAAR_ANCHOR_RE.search(previous):
                break
        break
    return ""


def _extract_name(text: str) -> str:
    for strategy in (
        _name_from_label,
        _name_after_aadhaar_anchor,
        _name_before_dob_label,
    ):
        candidate = strategy(text)
        if candidate:
            return candidate[:80]
    return ""


def _extract_dob(text: str) -> str:
    """Extract a DOB, including masked years such as ``16/05/2xxx``.

    Aadhaar prints the ``/DOB`` label on its own line, so look in a short
    window after (and briefly before) each label instead of requiring the date
    on the same line.
    """
    for label in _DOB_LABEL_RE.finditer(text):
        window = text[label.end() : label.end() + 120]
        match = _DATE_RE.search(window)
        if match:
            return match.group(0)
        before = text[max(0, label.start() - 60) : label.start()]
        match = _DATE_RE.search(before)
        if match:
            return match.group(0)

    # A masked year is a strong DOB signal even without a visible label.
    match = _MASKED_DATE_RE.search(text)
    return match.group(0) if match else ""


# Tokens that appear on an Aadhaar card but are never part of the holder's
# name. Used by the loose selfie fallback below.
_SELFIE_NAME_STOPWORDS = frozenset(
    {
        "aadhaar",
        "address",
        "authority",
        "birth",
        "card",
        "date",
        "dob",
        "enrolment",
        "female",
        "government",
        "identification",
        "india",
        "male",
        "mobile",
        "number",
        "photo",
        "signature",
        "uidai",
        "unique",
        "vid",
        "year",
    }
)

_SELFIE_DOB_RE = re.compile(
    r"(?<!\d)(\d{2}[/-]\d{2}[/-]\d{4}|\d{4}-\d{2}-\d{2})(?!\d)"
)


def _extract_name_from_selfie_text(text: str) -> str:
    """Loose name fallback for selfie OCR text.

    :func:`_extract_name` is tuned for a tight Aadhaar crop; in a selfie the
    card is a small part of the frame and those heuristics can come up empty.
    This picks the first line that looks like a person's name (2+ alphabetic
    words, no known card label).
    """
    for line in text.splitlines():
        cleaned = re.sub(r"[^A-Za-z ]+", " ", line)
        tokens = [token for token in cleaned.split() if len(token) > 1]
        if len(tokens) < 2:
            continue
        if {token.lower() for token in tokens} & _SELFIE_NAME_STOPWORDS:
            continue
        return " ".join(tokens)[:80]
    return ""


def _extract_dob_from_selfie_text(text: str) -> str:
    """Loose DOB fallback for selfie OCR text without a ``/DOB`` label."""
    match = _SELFIE_DOB_RE.search(text)
    return match.group(0) if match else ""


def _selfie_ocr_for_storage(selfie_ocr: dict[str, Any]) -> dict[str, Any]:
    """Copy of the selfie OCR result that is safe to persist.

    ``text_redacted`` is already masked, but ``aadhaar_number`` holds the raw
    12-digit number read off the selfie. Raw government IDs are never stored
    (see :func:`_redact_sensitive_ids`), so only the last four digits are kept,
    matching the convention used for the uploaded document.
    """
    stored = {
        key: value
        for key, value in selfie_ocr.items()
        # ``_internal_aadhaar_number`` is the raw in-memory number: it must
        # never reach storage, log lines or the API response.
        if key not in {"error", "_internal_aadhaar_number"}
    }
    number = stored.get("aadhaar_number") or selfie_ocr.get(
        "_internal_aadhaar_number"
    )
    stored["aadhaar_number"] = None
    stored["aadhaar_last4"] = str(number)[-4:] if number else None
    return stored


def _redact_sensitive_ids(text: str) -> str:
    """Keep human-readable fields but mask Aadhaar/PAN/long ID numbers.

    The Go schema comment says raw government IDs are never stored, so the
    OCR text persisted for admins must not contain them either. Last four
    characters are kept for manual verification.
    """
    text = _AADHAAR_RE.sub(lambda m: f"XXXX XXXX {m.group(3)}", text)
    text = _PAN_RE.sub(lambda m: f"XXXXXX{m.group(0)[-4:]}", text)
    text = _LONG_DIGITS_RE.sub(lambda m: f"XXXX{m.group(0)[-4:]}", text)
    return text


def _softmax(values: Any) -> Any:
    import numpy as np

    logits = np.asarray(values, dtype=np.float64).reshape(-1)
    if logits.size == 1:
        probability = 1.0 / (1.0 + float(np.exp(-logits[0])))
        return np.asarray([1.0 - probability, probability], dtype=np.float64)
    shifted = logits - float(np.max(logits))
    exp = np.exp(shifted)
    return exp / float(np.sum(exp))


def _iter_ocr_lines(raw: Any) -> Iterator[tuple[str, float]]:
    """Yield ``(text, score)`` from PaddleOCR 2.x or 3.x result shapes.

    PaddleOCR 2.x: ``[[ [box, (text, score)], ... ]]``
    PaddleOCR 3.x: ``[OCRResult(rec_texts=[...], rec_scores=[...])]``
    """
    if raw is None:
        return

    pages = raw if isinstance(raw, (list, tuple)) else [raw]

    for page in pages:
        if page is None:
            continue

        # PaddleOCR 3.x result object / dict.
        if hasattr(page, "get") and not isinstance(page, (list, tuple)):
            try:
                texts = page.get("rec_texts")
            except Exception:  # noqa: BLE001 - non-mapping page
                texts = None
            if texts is not None:
                try:
                    scores = page.get("rec_scores") or []
                except Exception:  # noqa: BLE001
                    scores = []
                for index, text in enumerate(texts):
                    score = scores[index] if index < len(scores) else 0.0
                    yield str(text), _as_float(score)
                continue

        if isinstance(page, dict):
            texts = page.get("rec_texts")
            if texts is not None:
                scores = page.get("rec_scores") or []
                for index, text in enumerate(texts):
                    score = scores[index] if index < len(scores) else 0.0
                    yield str(text), _as_float(score)
                continue
            continue

        # PaddleOCR 2.x page: a list of [box, (text, score)] entries.
        if isinstance(page, (list, tuple)):
            for item in page:
                if not isinstance(item, (list, tuple)) or len(item) < 2:
                    continue
                payload = item[1]
                if isinstance(payload, (list, tuple)) and len(payload) >= 2:
                    yield str(payload[0]), _as_float(payload[1])


def _ocr_error(
    reason: str, detail: str = "", document: dict[str, str] | None = None
) -> dict[str, Any]:
    return {
        "engine": "paddleocr",
        "model_version": ML_MODEL_VERSION_REAL,
        "mock": False,
        "document_type": (document or {}).get("type", "unknown"),
        "text": "",
        "name": "",
        "full_name": "",
        "ocr_name": "",
        "dob": "",
        "date_of_birth": "",
        "ocr_dob": "",
        "confidence": 0.0,
        "ocr_confidence": 0.0,
        "line_count": 0,
        "error": reason,
        "error_detail": detail[:200],
    }


# ---------------------------------------------------------------------------
# Document-in-selfie / business document helpers
# ---------------------------------------------------------------------------


def _selfie_ocr_result() -> dict[str, Any]:
    """Frozen result contract of ModelLoader.run_selfie_document_ocr.

    The key set is pinned by tests/worker/test_worker_doc_in_selfie.py. The full
    12-digit number travels in aadhaar_number / _internal_aadhaar_number for the
    in-memory comparison only; _selfie_ocr_for_storage() reduces it to
    aadhaar_last4 before anything is persisted.
    """
    return {
        "error": None,
        "aadhaar_number": None,
        "_internal_aadhaar_number": None,
        "name": None,
        "dob": None,
        "confidence": 0.0,
        "text_redacted": "",
        "preprocess_strategy": None,
        "card_detected": False,
        "images_tried": 0,
    }


# The three identifiers a business document set is cross-checked on. Patterns
# are applied to an UPPERCASED, whitespace-stripped copy of each OCR line,
# because documents print "HDFC 0001234" as often as "HDFC0001234".
_PAN_SEARCH_RE = re.compile(r"[A-Z]{5}[0-9]{4}[A-Z]")
_GSTIN_SEARCH_RE = re.compile(r"[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]")
_IFSC_SEARCH_RE = re.compile(r"[A-Z]{4}0[A-Z0-9]{6}")
_PAN_LABEL_RE = re.compile(r"\bPAN\b|PERMANENT\s*ACCOUNT", re.IGNORECASE)
_GSTIN_LABEL_RE = re.compile(
    r"\bGSTIN\b|\bGST\b|GOODS\s*AND\s*SERVICE", re.IGNORECASE
)
_IFSC_LABEL_RE = re.compile(r"\bIFSC\b|I\.F\.S\.C", re.IGNORECASE)

# Registration numbers (incorporation certificate / business registration
# certificate) are not shape-stable the way a PAN or a GSTIN is, so the label
# carries the weight: the value printed next to the document's own label
# ("CIN", "Registration No", "Udyam") is the one it means.
_CIN_SEARCH_RE = re.compile(r"[LU][0-9]{5}[A-Z]{2}[0-9]{4}[A-Z]{3}[0-9]{6}")
_REGISTRATION_LABEL_RE = re.compile(
    r"\bCIN\b|\bREG[A-Z]*\b|\bUDYAM\b|\bMSME\b|\bSHOP\s*ACT\b", re.IGNORECASE
)
_ALNUM_TOKEN_RE = re.compile(r"[A-Za-z0-9]+")

# The cancelled cheque's account number: a labelled run of digits ("A/c No",
# "Account Number"), grouped in fours on many cheque books, or - unlabelled -
# the longest standalone digit run on the page.
_ACCOUNT_LABEL_RE = re.compile(
    r"\bA\s*/?\s*C\b|\bAC(?:CT|COUNT)?\s*(?:NO|NUMBER|NUM)?\b", re.IGNORECASE
)
_DIGIT_RUN_RE = re.compile(r"[0-9]+")
# A digit run that stands alone: "HDFC0001234" and "ABCDE1234F" are not account
# numbers, 12345678901 on its own line is.
_STANDALONE_DIGIT_RE = re.compile(r"(?<![A-Za-z0-9])[0-9]{6,}(?![A-Za-z0-9])")

_OCR_LINE_SPLIT_RE = re.compile(r"[\r\n]+")


def _ocr_text_lines(text: Any) -> list[str]:
    if not isinstance(text, str) or not text:
        return []
    return [line.strip() for line in _OCR_LINE_SPLIT_RE.split(text) if line.strip()]


def _squash_ocr_line(line: str) -> str:
    """Uppercase and drop separators, so "HDFC 0001234" scans as one token."""
    return re.sub(r"[^A-Za-z0-9]", "", line).upper()


def _candidate_matches(pattern: re.Pattern, squashed: str) -> Iterator[str]:
    """Raw matches on the line, then on its OCR-confusable-folded copy.

    The second pass recovers a value whose digit was read as a letter (a PAN
    printed as "...1O34..." folds to "...1034..."), which the strict pattern
    cannot see on its own.
    """
    for match in pattern.finditer(squashed):
        yield match.group(0)
    folded = fold_ocr_confusables(squashed)
    if folded != squashed:
        for match in pattern.finditer(folded):
            yield match.group(0)


def _business_id_candidates(
    text: Any,
    pattern: re.Pattern,
    label_re: re.Pattern,
    validate: Any,
) -> list[str]:
    """Candidates for one identifier, values on a labelled line first.

    "HDFC0001234" anywhere in a document is weak evidence; the same value on the
    line carrying the field's own label ("IFSC:", "GSTIN", "PAN") is the one the
    document means. Within a bucket the first structural match in document order
    wins, and every candidate is shape-validated before it is accepted.
    """
    labelled: list[str] = []
    plain: list[str] = []
    for line in _ocr_text_lines(text):
        squashed = _squash_ocr_line(line)
        if not squashed:
            continue
        bucket = labelled if label_re.search(line) else plain
        for candidate in _candidate_matches(pattern, squashed):
            value = validate(candidate)
            if value and value not in bucket:
                bucket.append(value)
    return labelled + plain


def _first_business_id(
    text: Any,
    pattern: re.Pattern,
    label_re: re.Pattern,
    validate: Any,
) -> str | None:
    candidates = _business_id_candidates(text, pattern, label_re, validate)
    return candidates[0] if candidates else None


def _business_text_for(text_by_type: Any, doc_type: str) -> str:
    """OCR text for one business document slot (canonical name or alias key)."""
    if not isinstance(text_by_type, dict):
        return ""
    for key in (doc_type, *_BUSINESS_DOC_ALIASES[doc_type]):
        value = text_by_type.get(key)
        if isinstance(value, str) and value.strip():
            return value
    return ""


def _append_once(bucket: list[str], value: str) -> None:
    if value and value not in bucket:
        bucket.append(value)


def _registration_candidates(text: Any) -> list[str]:
    """Registration-number candidates off a certificate, strongest first.

    Four buckets, in priority order:

    1. a CIN-shaped match on a line that carries a registration label,
    2. a CIN-shaped match anywhere else on the line,
    3. the alphanumeric run(s) the label introduces,
    4. any other alphanumeric token of six characters or more carrying a digit.

    A certificate prints several numbers (dates, serial numbers, PIN codes), so
    the label and the CIN shape are what separate the registration number from
    the rest of the page. Labels win over shape, because a state's registration
    number ("UDYAM-MH-12-3456789", "1234 5678") is often not CIN-shaped.
    """
    buckets: list[list[str]] = [[], [], [], []]
    for line in _ocr_text_lines(text):
        label_match = _REGISTRATION_LABEL_RE.search(line)
        labelled = label_match is not None
        squashed = _squash_ocr_line(line)
        for candidate in _candidate_matches(_CIN_SEARCH_RE, squashed):
            value = normalise_registration_number(candidate)
            if value:
                _append_once(buckets[0] if labelled else buckets[1], value)

        if label_match is not None:
            # The run(s) the label introduces, joined first: registrations are
            # routinely printed in groups.
            tokens = _ALNUM_TOKEN_RE.findall(line[label_match.end() :])
            joined = normalise_registration_number("".join(tokens))
            if joined:
                _append_once(buckets[2], joined)
            for token in tokens:
                value = normalise_registration_number(token)
                if value:
                    _append_once(buckets[2], value)

        for token in _ALNUM_TOKEN_RE.findall(line):
            value = normalise_registration_number(token)
            if value:
                _append_once(buckets[2] if labelled else buckets[3], value)

    return [value for bucket in buckets for value in bucket]


def _extract_primary_business_id(text: Any, doc_type: str | None) -> str | None:
    """The identifier read off the ONE selected business document.

    GST certificate -> GSTIN, PAN card -> PAN, incorporation certificate and
    business registration certificate -> the company registration number. None
    when the selected type is unknown or the value could not be read.
    """
    if doc_type == "pan_card":
        return _first_business_id(text, _PAN_SEARCH_RE, _PAN_LABEL_RE, normalise_pan)
    if doc_type == "gst_certificate":
        return _first_business_id(
            text, _GSTIN_SEARCH_RE, _GSTIN_LABEL_RE, normalise_gstin
        )
    if doc_type in {"incorporation_certificate", "business_registration"}:
        candidates = _registration_candidates(text)
        return candidates[0] if candidates else None
    return None


def _joined_digit_run(tail: str) -> str | None:
    """The digit run an account label introduces, grouping included.

    "12345678901" stays whole; "0012 3456 7890" is joined into one value. When
    the first group is short and a longer run follows it (a MICR line after a
    branch/sort code), joining stops at that run so the tail stays the account
    number. Fewer than four digits is not an account number at all.
    """
    runs = _DIGIT_RUN_RE.findall(tail)
    if not runs:
        return None
    if len(runs[0]) >= 6:
        return runs[0]
    grouped = ""
    for run in runs:
        if len(run) > 4:
            break
        grouped += run
    return grouped if len(grouped) >= 4 else None


def _longest_digit_run(line: str, minimum: int, standalone: bool = False) -> str | None:
    pattern = _STANDALONE_DIGIT_RE if standalone else _DIGIT_RUN_RE
    runs = [run for run in pattern.findall(line) if len(run) >= minimum]
    return max(runs, key=len) if runs else None


def _extract_account_last4(text: Any) -> str | None:
    """The LAST FOUR DIGITS of the account number printed on the cheque.

    A labelled account number is read first ("A/c No", "Account Number") with
    its grouped digits joined; without a label the longest standalone digit run
    on the page is used. Digit runs glued to letters are never candidates (the
    IFSC's "HDFC0001234", a PAN's "1234"), lines carrying the IFSC label are
    skipped, and anything shorter than four digits is unreadable - never a match.
    """
    if not isinstance(text, str) or not text:
        return None

    labelled: list[str] = []
    plain: list[str] = []
    for line in _ocr_text_lines(text):
        label_match = _ACCOUNT_LABEL_RE.search(line)
        if label_match is not None:
            value = _joined_digit_run(line[label_match.end() :]) or _longest_digit_run(
                line, minimum=4
            )
            if value:
                labelled.append(value)
            continue
        if _IFSC_LABEL_RE.search(line):
            continue
        value = _longest_digit_run(line, minimum=6, standalone=True)
        if value:
            plain.append(value)

    if labelled:
        return labelled[0][-4:]
    if plain:
        return max(plain, key=len)[-4:]
    return None


def extract_selected_business_document_fields(
    text_by_type: Any, primary_doc_type: str | None
) -> dict[str, Any]:
    """Read the selected document's identifier and the cheque's bank details.

    Returns extracted_biz_num (the identifier the selected document type is
    identified by), cheque_ifsc and cheque_acc_no (the last four digits of the
    cheque's account number); each is None when it could not be read.

    The identifier is read ONLY off the selected document, never borrowed from
    another uploaded document, so a fake certificate cannot inherit the identity
    of a genuine one.
    """
    if primary_doc_type in _BUSINESS_DOC_ALIASES:
        primary_text = _business_text_for(text_by_type, primary_doc_type)
    else:
        primary_text = ""
    cheque_text = _business_text_for(text_by_type, "cancelled_cheque")

    return {
        "extracted_biz_num": _extract_primary_business_id(
            primary_text, primary_doc_type
        ),
        "cheque_ifsc": _first_business_id(
            cheque_text, _IFSC_SEARCH_RE, _IFSC_LABEL_RE, normalise_ifsc
        ),
        "cheque_acc_no": _extract_account_last4(cheque_text),
        "cheque_acc_last4": _extract_account_last4(cheque_text),
    }


def _business_ocr_failure(error: str | None) -> dict[str, Any]:
    """The Phase G evidence shape with nothing read (every key present)."""
    return {
        "extracted_biz_num": None,
        "cheque_ifsc": None,
        "cheque_acc_no": None,
        "cheque_acc_last4": None,
        "primary_doc_type": None,
        "error": error,
    }


def run_business_document_ocr(
    documents: list[dict[str, str]] | dict[str, Any] | None,
    cache: Any = None,
    selected_doc_type: str | None = None,
    loader: Any = None,
    text_by_type: dict[str, str] | None = None,
) -> dict[str, Any]:
    """Read the ONE selected business document and the cancelled cheque.

    Returns the Phase G evidence shape, always carrying every key:

    * ``extracted_biz_num`` - the identifier read off the selected document
      (GSTIN / PAN / registration number), None when it could not be read
    * ``cheque_ifsc``       - the IFSC read off the cancelled cheque
    * ``cheque_acc_no``     - the LAST FOUR DIGITS of the cheque's account number
    * ``primary_doc_type``  - the resolved canonical type of the selected
      document (one of BUSINESS_PRIMARY_DOC_TYPES), None when the job carries
      no selectable business document
    * ``error``             - None, or "missing_pan_card" /
      "missing_cancelled_cheque" / "missing_business_documents" /
      "ocr_unavailable" / "<type>_fetch_failed" / "<type>_ocr_failed"

    ``selected_doc_type`` is the type the user picked. When it is absent or is
    not one of the four selectable business documents, the type is INFERRED from
    the uploaded documents (see _infer_business_doc_type) - the Go producer does
    not send the selection today, so inference is what keeps the stage working
    at runtime.

    * text_by_type - already-extracted OCR text per document type. The mock
      pipeline passes the mock_business_document_text() seam here, so no image is
      ever fetched in mock mode.
    * otherwise the located documents are fetched through cache (or fetch_image)
      and OCRd with loader.ocr_image_text() - the same engine path the selfie
      stage uses. There is no second OCR engine.

    Never raises. A missing document, an unavailable engine or a failed fetch
    sets error and returns whatever was read, so the comparison stage routes the
    record to manual review.
    """
    result: dict[str, Any] = _business_ocr_failure(None)

    try:
        document_list = [doc for doc in (documents or []) if isinstance(doc, dict)]

        primary_doc_type = _canonical_business_doc_type(selected_doc_type)
        if primary_doc_type not in BUSINESS_PRIMARY_DOC_TYPES:
            primary_doc_type = _infer_business_doc_type(document_list)
        result["primary_doc_type"] = primary_doc_type

        located: dict[str, dict] = {}
        missing_error: str | None = None

        primary_document = (
            _find_document(document_list, set(_BUSINESS_DOC_ALIASES[primary_doc_type]))
            if primary_doc_type
            else None
        )
        if primary_doc_type is None:
            # Nothing says which business document was selected or uploaded.
            missing_error = _BUSINESS_NO_DOCUMENT_ERROR
        elif primary_document is None or not str(
            primary_document.get("url") or ""
        ).strip():
            missing_error = _BUSINESS_MISSING_ERRORS[primary_doc_type]
        else:
            located[primary_doc_type] = primary_document

        cheque_document = _find_document(
            document_list, set(_BUSINESS_DOC_ALIASES["cancelled_cheque"])
        )
        if cheque_document is None or not str(cheque_document.get("url") or "").strip():
            if missing_error is None:
                missing_error = _BUSINESS_MISSING_ERRORS["cancelled_cheque"]
        else:
            located["cancelled_cheque"] = cheque_document

        if not located:
            result["error"] = missing_error or _BUSINESS_NO_DOCUMENT_ERROR
            return result

        if text_by_type is None:
            engine = getattr(loader, "ocr_engine", None)
            if loader is None or engine is None:
                result["error"] = "ocr_unavailable"
                return result
            texts: dict[str, str] = {}
            for doc_type, document in located.items():
                try:
                    image = (
                        cache.get(document["url"])
                        if cache is not None
                        else fetch_image(document["url"])
                    )
                except Exception as exc:  # noqa: BLE001 - one document only
                    logger.warning(
                        "business_doc_fetch_failed type=%s error=%s",
                        doc_type,
                        type(exc).__name__,
                    )
                    result["error"] = result["error"] or f"{doc_type}_fetch_failed"
                    continue
                try:
                    text, _confidence = loader.ocr_image_text(image)
                except Exception as exc:  # noqa: BLE001 - one document only
                    logger.warning(
                        "business_doc_ocr_failed type=%s error=%s",
                        doc_type,
                        type(exc).__name__,
                    )
                    result["error"] = result["error"] or f"{doc_type}_ocr_failed"
                    continue
                texts[doc_type] = text
        else:
            # Only documents the payload actually carries are trusted: text for
            # a slot the job never uploaded is not evidence. Alias keys ("gst")
            # are normalised onto the canonical document types here.
            texts = {
                doc_type: _business_text_for(text_by_type, doc_type)
                for doc_type in located
            }

        result.update(
            extract_selected_business_document_fields(texts, primary_doc_type)
        )
        if missing_error and not result["error"]:
            result["error"] = missing_error
        return result
    except Exception as exc:  # noqa: BLE001 - per-job stage, never fatal
        logger.warning("business_docs_ocr_exception error=%s", type(exc).__name__)
        result["error"] = result["error"] or "business_docs_ocr_exception"
        return result


def is_business_job(job: Any) -> bool:
    """True when the job is a business KYC record.

    Rule (mirrors the Go router): type == "business", or the document list
    carries at least one business slot (one of the four selectable business
    documents, or the cancelled cheque) and no individual marker (selfie /
    aadhaar).
    """
    if not isinstance(job, dict):
        return False
    if str(job.get("type") or "").strip().lower() == "business":
        return True

    document_types = {
        str(document.get("type") or "").strip().lower()
        for document in (job.get("documents") or [])
        if isinstance(document, dict)
    }
    business_slots = {
        alias for aliases in _BUSINESS_DOC_ALIASES.values() for alias in aliases
    }
    if not (document_types & business_slots):
        return False
    individual_markers = set(SELFIE_DOC_TYPES) | AADHAAR_DOC_TYPES
    return not (document_types & individual_markers)


def _raw_payload_dict(payload: Any) -> dict[str, Any]:
    """Best-effort raw job payload as a dict. Never raises."""
    if isinstance(payload, dict):
        return payload
    if isinstance(payload, (bytes, bytearray)):
        try:
            payload = payload.decode("utf-8")
        except UnicodeDecodeError:
            return {}
    if isinstance(payload, str):
        try:
            data = json.loads(payload)
        except (TypeError, ValueError):
            return {}
        return data if isinstance(data, dict) else {}
    return {}


def _first_text_value(source: dict, keys: tuple[str, ...]) -> str:
    for key in keys:
        value = source.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
        if isinstance(value, (int, float)) and not isinstance(value, bool):
            return str(value)
    return ""


_BUSINESS_NESTED_SOURCES = ("business", "business_details", "bank_details", "bank", "kyc")

_BUSINESS_SELECTION_KEYS = (
    "selected_doc_type",
    "selected_business_doc",
    "business_doc_type",
    "business_document_type",
)


def _payload_sources(data: dict) -> list[dict]:
    """The payload dict plus every nested business/bank object it carries."""
    sources = [data]
    for key in _BUSINESS_NESTED_SOURCES:
        nested = data.get(key)
        if isinstance(nested, dict):
            sources.append(nested)
    return sources


def _first_payload_value(sources: list[dict], keys: tuple[str, ...]) -> str:
    """First non-empty value among *keys*; the root payload wins over nesting."""
    for source in sources:
        value = _first_text_value(source, keys)
        if value:
            return value
    return ""


def resolve_business_doc_type(payload: Any, documents: Any = None) -> str | None:
    """The canonical business document type this job is about.

    The explicit selection the user made - "selected_doc_type" /
    "selected_business_doc" (also accepted nested inside the payload's
    "business" / "business_details" object) - wins. When it is absent or is not
    one of the four selectable business documents, the type is INFERRED from the
    uploaded documents. Inference matters in production: the Go producer
    publishes documents only, so without it a business job would never have a
    selected document to read and would always land on manual review with no
    evidence at all.

    Returns one of BUSINESS_PRIMARY_DOC_TYPES, or None when nothing in the job
    indicates a selectable business document.
    """
    data = _raw_payload_dict(payload)
    sources = _payload_sources(data)
    for key in _BUSINESS_SELECTION_KEYS:
        selected = _canonical_business_doc_type(_first_payload_value(sources, (key,)))
        if selected in BUSINESS_PRIMARY_DOC_TYPES:
            return selected
    if documents is None:
        documents = data.get("documents")
    return _infer_business_doc_type(documents)


def _business_uploaded_ocr(payload: Any) -> dict[str, Any]:
    """Return the explicit Phase G form data carried by the Go job payload."""
    data = _raw_payload_dict(payload)
    business = data.get("business_info") if isinstance(data.get("business_info"), dict) else {}
    bank = data.get("bank_details") if isinstance(data.get("bank_details"), dict) else {}
    return {
        "business_info": {
            "gstin": str(business.get("gstin") or ""),
            "pan": str(business.get("pan") or ""),
            "reg_no": str(business.get("reg_no") or ""),
        },
        "bank_details": {
            "ifsc": str(bank.get("ifsc") or ""),
            "account_number": str(bank.get("account_number") or ""),
        },
        "selected_business_doc": str(data.get("selected_business_doc") or ""),
    }


def mock_business_document_text(documents: list[dict[str, str]] | None) -> dict[str, str]:
    """Synthetic business document OCR text for the mock pipeline.

    Empty by default: mock mode runs on fake document URLs, so it must never
    fabricate a document read. A business job therefore lands on manual review
    unless a test supplies matching text by monkeypatching this hook (the seam
    tests/worker/test_worker_business.py uses).
    """
    return {}


# ---------------------------------------------------------------------------
# Face / liveness helpers
# ---------------------------------------------------------------------------


def _face_error(
    reason: str,
    detail: str = "",
    document_types: list[str] | None = None,
    govt_detected: bool = False,
    selfie_detected: bool = False,
    quality: dict[str, Any] | None = None,
) -> dict[str, Any]:
    """Failure shape for the face stage (Phase 3), extended in Phase 5.

    The Phase 5 keys (``winning_reference``, ``aadhaar_similarity``,
    ``reference_scores``, ``reference_count``, ``second_reference_divergence``,
    ``quality``) are always present so the Go router's JSON contract over
    ``face_match_result`` is uniform in success and failure: on an error the
    Aadhaar similarity is explicitly ``None`` and the divergence flag is
    ``False``, which the router's evidence gate reads as "cannot auto-approve".
    """
    return {
        "engine": "opencv-sface",
        "model_version": ML_MODEL_VERSION_REAL,
        "mock": False,
        "face_similarity": 0.0,
        "similarity": 0.0,
        "confidence": 0.0,
        "cosine_similarity": 0.0,
        # Phase 5 multi-reference fields (error-safe defaults).
        "winning_reference": None,
        "aadhaar_similarity": None,
        "reference_scores": {},
        "reference_count": 0,
        "second_reference_divergence": False,
        "quality": quality or {},
        "document_types": document_types or [],
        "face_detected": {
            "govt_id": govt_detected,
            "selfie": selfie_detected,
        },
        "thresholds": {
            "pass": settings.threshold_face_auto_pass,
            "reject": settings.threshold_face_reject,
        },
        "error": reason,
        "error_detail": detail[:200],
    }


def _compute_second_reference_divergence(
    max_ref: str | None, max_score: float, aadhaar_score: float | None
) -> bool:
    """Phase 5 "second reference corroboration" guard.

    The Aadhaar score is a cryptographic trust anchor (QR-verified photo). When
    a different reference (typically PAN) wins the comparison but the Aadhaar
    score is severely weak, the winning score is suspicious: a fake PAN carrying
    the match for an unverified Aadhaar. Divergence caps the raw priority at
    manual_review (both here and in the Go router's evidence gate).

    Single-reference jobs and jobs without a scorable Aadhaar photo are never
    divergent: there is no second opinion to disagree with (the Aadhaar floor
    gate handles that risk instead).
    """
    if aadhaar_score is None or not max_ref or max_ref == "aadhaar":
        return False
    return (max_score - aadhaar_score) > settings.second_reference_divergence_threshold


def _build_face_match_result(
    *,
    document_types: list[str],
    scores_by_ref: dict[str, float],
    quality_by_ref: dict[str, Any] | None = None,
    selfie_detected: bool = True,
    mock: bool = False,
    engine: str = "opencv-sface",
    model_version: str = ML_MODEL_VERSION_REAL,
) -> dict[str, Any]:
    """Assemble the Phase 5 multi-reference face-match result.

    Single source of truth for the result JSON contract so mock mode and real
    inference cannot drift. Requires ``scores_by_ref`` to be non-empty — an
    empty map means the caller emits a ``no_usable_reference`` error instead.

    * ``face_similarity`` is the PRIMARY decision score: max() over the per-
      reference cosines, clamped to [0, 1] (matches the Phase 3 clamp).
    * ``reference_scores`` keeps the raw (unclamped) cosines — negative values
      are real measurements, not errors.
    * ``aadhaar_similarity`` is None when the Aadhaar photo is absent OR failed
      the quality gate: "no Aadhaar uploaded" and "Aadhaar unreadable" both
      legally block auto-approval downstream; ``0.0`` would wrongly mean
      "县政府 confident mismatch" for a missing document.
    * ``cosine_similarity`` stays as an alias of the primary score for the
      Phase 3 router and any older consumers.
    """
    if not scores_by_ref:
        return _face_error(
            "no_usable_reference",
            "",
            document_types,
            selfie_detected=selfie_detected,
            quality=quality_by_ref,
        )

    max_ref = max(scores_by_ref, key=scores_by_ref.get)
    max_score_raw = float(scores_by_ref[max_ref])
    aadhaar_score_raw = scores_by_ref.get("aadhaar")
    divergence = _compute_second_reference_divergence(
        max_ref, max_score_raw, aadhaar_score_raw
    )

    similarity = round(float(max(0.0, min(1.0, max_score_raw))), 4)
    return {
        "engine": engine,
        "model_version": model_version,
        "mock": mock,
        "error": None,
        # Primary decision fields (Phase 5).
        "face_similarity": similarity,
        "winning_reference": max_ref,
        "aadhaar_similarity": (
            None if aadhaar_score_raw is None else round(float(aadhaar_score_raw), 4)
        ),
        "reference_scores": {
            ref: round(float(score), 4) for ref, score in scores_by_ref.items()
        },
        "reference_count": len(scores_by_ref),
        "second_reference_divergence": bool(divergence),
        # Quality metadata per reference (Phase 5 gates).
        "quality": quality_by_ref or {},
        # Phase 3 back-compat aliases.
        "similarity": similarity,
        "confidence": similarity,
        "cosine_similarity": round(max_score_raw, 4),
        "face_detected": {
            "selfie": selfie_detected,
            **{ref: True for ref in scores_by_ref},
        },
        "document_types": document_types,
        "thresholds": {
            "auto_pass": settings.threshold_face_auto_pass,
            "review": settings.threshold_face_review,
            "reject": settings.threshold_face_reject,
        },
    }


def _liveness_result(
    score: float,
    *,
    selfie_present: bool,
    face_detected: bool,
    error: str = "",
    detail: str = "",
    probabilities: list[float] | None = None,
) -> dict[str, Any]:
    score = round(max(0.0, min(1.0, float(score))), 4)
    result: dict[str, Any] = {
        "engine": "minifasnetv2-onnx",
        "model_version": ML_MODEL_VERSION_REAL,
        "mock": False,
        "liveness_score": score,
        "score": score,
        "is_live": score >= settings.threshold_live_pass,
        "selfie_present": selfie_present,
        "face_detected": face_detected,
        "thresholds": {
            "pass": settings.threshold_live_pass,
            "reject": settings.threshold_live_reject,
        },
    }
    if probabilities:
        result["probabilities"] = probabilities
    if error:
        result["error"] = error
        result["error_detail"] = detail[:200]
    return result


def _crop_face(image: Any, face: Any, scale: float) -> Any:
    """Crop a face bbox with a margin, matching Silent-Face-Anti-Spoofing."""
    height, width = image.shape[:2]
    x, y, box_w, box_h = (float(value) for value in face[:4])
    if box_w <= 1 or box_h <= 1:
        return image[0:0, 0:0]

    # Do not exceed the source image dimensions (upstream behaviour).
    actual_scale = min(
        (height - 1) / box_h,
        (width - 1) / box_w,
        float(scale),
    )
    new_w = box_w * actual_scale
    new_h = box_h * actual_scale

    center_x = x + box_w / 2.0
    center_y = y + box_h / 2.0

    x1 = max(0, int(center_x - new_w / 2.0))
    y1 = max(0, int(center_y - new_h / 2.0))
    x2 = min(width - 1, int(center_x + new_w / 2.0))
    y2 = min(height - 1, int(center_y + new_h / 2.0))

    return image[y1 : y2 + 1, x1 : x2 + 1]


# ---------------------------------------------------------------------------
# Mock model stages (ML_MODE=mock)
# ---------------------------------------------------------------------------


def mock_ocr(documents: list[dict[str, str]]) -> dict[str, Any]:
    """Deterministic OCR result for smoke tests/CI."""
    document = _first_document_of_type(documents, GOVT_DOC_TYPES)
    confidence = round(settings.mock_ocr_confidence, 4)

    return {
        "engine": "mock-paddleocr",
        "model_version": ML_MODEL_VERSION_MOCK,
        "mock": True,
        "document_type": (document or {}).get("type", "unknown"),
        "text": "MOCK OCR TEXT\nNAME: MOCK USER\nDOB: 1990-01-01",
        "name": "Mock User",
        "full_name": "Mock User",
        "ocr_name": "Mock User",
        "dob": "1990-01-01",
        "date_of_birth": "1990-01-01",
        "ocr_dob": "1990-01-01",
        "confidence": confidence,
        "ocr_confidence": confidence,
    }


# Deterministic multi-reference scores (ML_MODE=mock). Chosen below the
# divergence threshold (0.62 - 0.45 = 0.17 <= 0.25) so the mock exercises the
# healthy multi-reference path; divergence paths are covered by unit tests.
MOCK_FACE_REFERENCE_SCORES = {"aadhaar": 0.45, "pan_card": 0.62}


def _make_face_disabled_result(doc_types: list[str] | None = None) -> dict:
    """
    Result shape when face match is disabled.
    Must include 'error' so determine_review_priority returns
    manual_review (NOT auto_reject). face_similarity=0.0 means
    no evidence; router's SQL guard (>0.363) and evidenceComplete
    will also reject auto-approval.
    """
    return {
        "enabled": False,
        "engine": "disabled",
        "model_version": None,
        "error": "face_match_disabled",
        "face_similarity": 0.0,
        "cosine_similarity": 0.0,
        "aadhaar_similarity": None,
        "winning_reference": None,
        "reference_scores": {},
        "reference_count": 0,
        "second_reference_divergence": False,
        "quality": {},
        "document_types": doc_types or [],
        "face_detected": {"selfie": False},
    }


def mock_face_match(documents: list[dict[str, str]]) -> dict[str, Any]:
    """Deterministic SFace result for smoke tests/CI (Phase 5 shape).

    Every government-type document contributes a reference score: aadhaar and
    pan_card get the planted multi-reference scores above; unknown/legacy types
    (e.g. govt_id) reuse ``settings.mock_face_similarity``. This lets Phase 5
    tests exercise multi-reference paths without real images.
    """
    document_types = [doc.get("type", "") for doc in documents]

    scores_by_ref: dict[str, float] = {}
    for doc in documents:
        doc_type = str(doc.get("type") or "")
        if doc_type not in GOVT_DOC_TYPES or doc_type in scores_by_ref:
            continue
        planted = MOCK_FACE_REFERENCE_SCORES.get(doc_type)
        scores_by_ref[doc_type] = (
            planted if planted is not None else round(settings.mock_face_similarity, 4)
        )

    quality_by_ref = {
        doc_type: {
            "width": 200,
            "height": 200,
            "detection_score": 0.99,
            "sharpness": 250.0,
            "usable": True,
        }
        for doc_type in scores_by_ref
    }

    selfie_present = any(doc.get("type") in SELFIE_DOC_TYPES for doc in documents)
    if not scores_by_ref:
        return _face_error(
            "missing_document",
            "both a government ID and a selfie are required",
            document_types,
            selfie_detected=selfie_present,
        )
    return _build_face_match_result(
        document_types=document_types,
        scores_by_ref=scores_by_ref,
        quality_by_ref=quality_by_ref,
        selfie_detected=selfie_present,
        mock=True,
        engine="mock-sface",
        model_version=ML_MODEL_VERSION_MOCK,
    )


def mock_liveness(documents: list[dict[str, str]]) -> dict[str, Any]:
    """Deterministic MiniFASNet result for smoke tests/CI."""
    score = round(settings.mock_liveness_score, 4)
    selfie_present = any(doc.get("type") in SELFIE_DOC_TYPES for doc in documents)

    return {
        "engine": "mock-minifasnet",
        "model_version": ML_MODEL_VERSION_MOCK,
        "mock": True,
        "liveness_score": score,
        "score": score,
        "is_live": score >= settings.threshold_live_pass,
        "selfie_present": selfie_present,
        "thresholds": {
            "pass": settings.threshold_live_pass,
            "reject": settings.threshold_live_reject,
        },
    }


# ---------------------------------------------------------------------------
# Real model loader + inference
# ---------------------------------------------------------------------------


class ModelLoader:
    """Load every AI model once, then reuse it for all jobs.

    ``load_models`` is idempotent and thread-safe. Individual model failures
    are recorded in ``errors`` rather than killing the service, so the health
    endpoint can report exactly which file/backend is missing.
    """

    def __init__(self, model_dir: str | Path | None = None) -> None:
        self.mode = settings.ml_mode
        self.model_dir = Path(model_dir or settings.model_dir)

        self._cv2: Any = None
        self._np: Any = None

        self.face_detector: Any = None
        self.face_recognizer: Any = None

        self.liveness_session: Any = None
        self.liveness_input_name = ""
        self.liveness_output_name = ""
        self.liveness_input_size = (80, 80)

        self.ocr_engine: Any = None

        self.errors: dict[str, str] = {}
        self._loaded = False
        self._load_lock = threading.Lock()

    # -- readiness ---------------------------------------------------------

    @property
    def core_ready(self) -> bool:
        if self.mode == "mock":
            return True
        return (
            self.face_detector is not None
            and self.face_recognizer is not None
            and self.liveness_session is not None
        )

    @property
    def ocr_ready(self) -> bool:
        return self.mode == "mock" or self.ocr_engine is not None

    @property
    def ready(self) -> bool:
        return self.mode == "mock" or (self.core_ready and self.ocr_ready)

    @property
    def model_version(self) -> str:
        return (
            ML_MODEL_VERSION_MOCK
            if self.mode == "mock"
            else ML_MODEL_VERSION_REAL
        )

    def status(self) -> dict[str, Any]:
        if self.mode == "mock":
            state = "mock"
        elif self.ready:
            state = "ready"
        elif self.core_ready:
            state = "degraded"
        else:
            state = "unavailable"

        def state_of(value: Any) -> str:
            if self.mode == "mock":
                return "not_used"
            return "loaded" if value is not None else "missing"

        return {
            "mode": self.mode,
            "state": state,
            "model_dir": str(self.model_dir),
            "ocr_backend": settings.ocr_backend,
            "face_detector": state_of(self.face_detector),
            "face_recognizer": state_of(self.face_recognizer),
            "liveness": state_of(self.liveness_session),
            "ocr": state_of(self.ocr_engine),
            "errors": dict(self.errors),
        }

    # -- loading -----------------------------------------------------------

    def load_models(self) -> None:
        with self._load_lock:
            if self._loaded:
                return
            self._loaded = True

            if self.mode == "mock":
                logger.info(
                    "ML_MODE=mock: deterministic mock scores enabled; "
                    "ONNX/PaddleOCR models were not loaded"
                )
                return

            logger.info("Loading AI models from %s", self.model_dir)
            try:
                self.model_dir.mkdir(parents=True, exist_ok=True)
            except OSError as exc:
                logger.warning("Cannot create model directory: %s", exc)

            self._load_face_models()
            self._load_liveness_model()
            self._load_ocr_engine()

            if self.core_ready:
                logger.info("Core AI models loaded (face detector recognizer + liveness)")
            else:
                logger.error(
                    "Core AI models are incomplete; real-mode jobs will fail. "
                    "Run download_models.py. Errors: %s",
                    self.errors,
                )
            if self.ocr_engine is None:
                logger.warning(
                    "OCR is unavailable (%s); OCR confidence will be 0.0",
                    self.errors.get("ocr", "unknown"),
                )

    def _load_face_models(self) -> None:
        try:
            import cv2
            import numpy as np
        except Exception as exc:  # noqa: BLE001
            self.errors["opencv"] = f"{type(exc).__name__}: {exc}"
            return

        self._cv2 = cv2
        self._np = np

        detector_path = self.model_dir / settings.face_detector_model
        recognizer_path = self.model_dir / settings.face_recognizer_model

        if not detector_path.is_file():
            self.errors["face_detector"] = (
                f"missing {detector_path}; run download_models.py"
            )
        else:
            try:
                self.face_detector = cv2.FaceDetectorYN_create(
                    str(detector_path),
                    "",
                    (320, 320),
                    settings.face_detect_score_threshold,
                    settings.face_detect_nms_threshold,
                    settings.face_detect_top_k,
                )
                logger.info("Loaded YuNet face detector: %s", detector_path)
            except Exception as exc:  # noqa: BLE001
                self.errors["face_detector"] = f"{type(exc).__name__}: {exc}"

        if not recognizer_path.is_file():
            self.errors["face_recognizer"] = (
                f"missing {recognizer_path}; run download_models.py"
            )
        else:
            try:
                self.face_recognizer = cv2.FaceRecognizerSF_create(
                    str(recognizer_path), ""
                )
                logger.info("Loaded SFace recognizer: %s", recognizer_path)
            except Exception as exc:  # noqa: BLE001
                self.errors["face_recognizer"] = f"{type(exc).__name__}: {exc}"

    def _load_liveness_model(self) -> None:
        try:
            import onnxruntime as ort
        except Exception as exc:  # noqa: BLE001
            self.errors["onnxruntime"] = f"{type(exc).__name__}: {exc}"
            return

        model_path = self.model_dir / settings.liveness_model
        if not model_path.is_file():
            self.errors["liveness"] = (
                f"missing {model_path}; run download_models.py"
            )
            return

        try:
            providers = ["CPUExecutionProvider"]
            if settings.liveness_use_gpu:
                providers = ["CUDAExecutionProvider", "CPUExecutionProvider"]

            session = ort.InferenceSession(str(model_path), providers=providers)
            input_meta = session.get_inputs()[0]
            output_meta = session.get_outputs()[0]

            self.liveness_session = session
            self.liveness_input_name = input_meta.name
            self.liveness_output_name = output_meta.name

            shape = list(input_meta.shape)
            if (
                len(shape) >= 4
                and isinstance(shape[2], int)
                and isinstance(shape[3], int)
                and shape[2] > 0
                and shape[3] > 0
            ):
                self.liveness_input_size = (int(shape[2]), int(shape[3]))

            logger.info(
                "Loaded MiniFASNetV2: %s (input=%s, providers=%s)",
                model_path,
                self.liveness_input_size,
                session.get_providers(),
            )
        except Exception as exc:  # noqa: BLE001
            self.errors["liveness"] = f"{type(exc).__name__}: {exc}"

    def _load_ocr_engine(self) -> None:
        if settings.ocr_backend == "none":
            self.errors["ocr"] = "disabled by OCR_BACKEND=none"
            return

        try:
            from paddleocr import PaddleOCR
        except Exception as exc:  # noqa: BLE001
            self.errors["ocr"] = f"{type(exc).__name__}: {exc}"
            return

        logger.info(
            "Loading PaddleOCR (lang=%s); the first run downloads its models",
            settings.ocr_lang,
        )
        try:
            try:
                engine = PaddleOCR(
                    lang=settings.ocr_lang,
                    use_angle_cls=False,
                    show_log=False,
                    use_gpu=settings.ocr_use_gpu,
                )
            except TypeError:
                # PaddleOCR 3.x constructor.
                engine = PaddleOCR(
                    lang=settings.ocr_lang,
                    use_doc_orientation_classify=False,
                    use_doc_unwarping=False,
                    use_textline_orientation=False,
                )
            self.ocr_engine = engine
            logger.info("PaddleOCR loaded")
        except Exception as exc:  # noqa: BLE001
            self.errors["ocr"] = f"{type(exc).__name__}: {exc}"

    # -- inference ---------------------------------------------------------

    def run_ocr(
        self,
        documents: list[dict[str, str]],
        cache: ImageCache,
        primary_bytes: bytes | None = None,
    ) -> dict[str, Any]:
        document = _first_document_of_type(documents, GOVT_DOC_TYPES)
        if document is None:
            return _ocr_error("missing_govt_document")

        if self.ocr_engine is None:
            return _ocr_error(
                "ocr_unavailable", self.errors.get("ocr", "OCR engine not loaded"), document
            )

        try:
            if primary_bytes is not None:
                import cv2
                import numpy as np

                image = cv2.imdecode(
                    np.frombuffer(primary_bytes, dtype=np.uint8),
                    cv2.IMREAD_COLOR,
                )
                if image is None:
                    raise ImageFetchError(
                        "could not decode provided image bytes (not a JPEG/PNG?)"
                    )
            else:
                image = cache.get(document["url"])
        except ImageFetchError as exc:
            return _ocr_error("image_fetch_failed", str(exc), document)

        try:
            raw = self._run_paddleocr(image)
            lines = [
                (text, score)
                for text, score in _iter_ocr_lines(raw)
                if text and text.strip()
            ]
        except Exception as exc:  # noqa: BLE001 - per-job failure only
            logger.exception("PaddleOCR inference failed")
            return _ocr_error(
                "ocr_failed", f"{type(exc).__name__}: {exc}", document
            )

        if not lines:
            return _ocr_error("no_text_detected", "", document)

        texts = [text.strip() for text, _ in lines]
        scores = [max(0.0, min(1.0, score)) for _, score in lines]
        weights = [max(1, len(text)) for text in texts]
        confidence = sum(score * weight for score, weight in zip(scores, weights))
        confidence /= sum(weights)

        raw_text = "\n".join(texts)
        name = _extract_name(raw_text)
        dob = _extract_dob(raw_text)
        aadhaar_from_ocr = find_aadhaar_in_text(raw_text)
        pan_valid = any(
            is_valid_pan(candidate)
            for candidate in _PAN_CANDIDATE_RE.findall(raw_text)
        )
        text = _redact_sensitive_ids(raw_text)
        confidence = round(float(confidence), 4)

        return {
            "engine": "paddleocr",
            "model_version": ML_MODEL_VERSION_REAL,
            "mock": False,
            "document_type": document.get("type", "unknown"),
            "text": text,
            "name": name,
            "full_name": name,
            "ocr_name": name,
            "dob": dob,
            "date_of_birth": dob,
            "ocr_dob": dob,
            "confidence": confidence,
            "ocr_confidence": confidence,
            "line_count": len(lines),
            "aadhaar_valid": aadhaar_from_ocr is not None,
            "aadhaar_source": "ocr_text" if aadhaar_from_ocr else "none",
            "aadhaar_last4": aadhaar_from_ocr[-4:] if aadhaar_from_ocr else None,
            # INTERNAL, in-memory only: the full 12-digit number is what makes
            # a real identity comparison possible. process_job strips this key
            # before anything is logged, persisted or returned.
            "_internal_aadhaar_number": aadhaar_from_ocr,
            "pan_valid": pan_valid,
        }

    def ocr_image_text(self, image: Any) -> tuple[str, float]:
        """Run the shared OCR engine over ONE preprocessed image.

        Returns (text, confidence): newline-joined lines plus the
        length-weighted mean of the per-line scores. Engine exceptions
        propagate so the caller can decide whether to try the next frame.
        """
        raw = self._run_paddleocr(image)
        lines = [
            (text, score)
            for text, score in _iter_ocr_lines(raw)
            if text and text.strip()
        ]
        if not lines:
            return "", 0.0
        texts = [text.strip() for text, _ in lines]
        weights = [max(1, len(text)) for text in texts]
        total = sum(
            max(0.0, min(1.0, score)) * weight
            for (_, score), weight in zip(lines, weights)
        )
        return "\n".join(texts), round(float(total / sum(weights)), 4)

    def _preprocess_selfie_frames(self, image: Any) -> dict[str, Any]:
        """Run the module-level preprocessor, falling back to the raw frame.

        The preprocessor is looked up on the module (instead of being imported
        inside the method) so a test can monkeypatch
        worker.preprocess_selfie_image; if it is missing or raises, the
        unmodified frame is OCRd instead of failing the whole stage.
        """
        processor = globals().get("preprocess_selfie_image")
        if callable(processor):
            try:
                result = processor(image)
            except Exception as exc:  # noqa: BLE001 - degrade to the raw frame
                logger.warning("selfie_preprocess_failed error=%s", type(exc).__name__)
            else:
                if isinstance(result, dict):
                    return result
        if image is None:
            return {"images": [], "strategy": "decode_failed", "card_detected": False}
        return {"images": [image], "strategy": "full_image", "card_detected": False}

    def run_selfie_document_ocr(
        self, documents: list[dict[str, str]], cache: Any = None
    ) -> dict:
        """OCR the government document held in the selfie.

        Every failure path returns the full result contract with error set: the
        method never raises, because an unreadable selfie must not fail the job.
        The frames the preprocessor returned are OCRd in order; the first frame
        carrying a Verhoeff-valid Aadhaar wins, otherwise the highest-confidence
        read is kept. A read below settings.doc_in_selfie_min_confidence is
        reported as selfie_ocr_low_confidence (with its evidence) so the
        comparison stage routes it to manual review.
        """
        result = _selfie_ocr_result()

        document_list = [doc for doc in (documents or []) if isinstance(doc, dict)]
        selfie_doc = _find_document(document_list, SELFIE_DOC_TYPES)
        if selfie_doc is None or not str(selfie_doc.get("url") or "").strip():
            result["error"] = "selfie_not_found"
            return result

        if self.ocr_engine is None:
            result["error"] = "ocr_unavailable"
            return result

        try:
            image = (
                cache.get(selfie_doc["url"])
                if cache is not None
                else fetch_image(selfie_doc["url"])
            )
        except Exception:  # noqa: BLE001 - reported as a stage error
            result["error"] = "selfie_fetch_failed"
            return result

        preprocessed = self._preprocess_selfie_frames(image)
        result["preprocess_strategy"] = str(preprocessed.get("strategy") or "full_image")
        result["card_detected"] = bool(preprocessed.get("card_detected"))
        frames = [frame for frame in (preprocessed.get("images") or [])]

        if not frames:
            result["error"] = "selfie_ocr_decode_failed"
            return result

        best_text = ""
        best_confidence = 0.0
        for frame in frames:
            result["images_tried"] += 1
            try:
                text, confidence = self.ocr_image_text(frame)
            except Exception as exc:  # noqa: BLE001 - try the next frame
                logger.warning("Selfie OCR pass failed: %s", exc)
                continue
            if not text.strip():
                continue
            if not best_text or confidence > best_confidence:
                best_text = text
                best_confidence = confidence
            if find_aadhaar_in_text(text):
                best_text = text
                best_confidence = confidence
                break

        if not best_text.strip():
            result["error"] = "selfie_no_text_detected"
            return result

        aadhaar = find_aadhaar_in_text(best_text)
        result["aadhaar_number"] = aadhaar
        result["_internal_aadhaar_number"] = aadhaar
        result["name"] = _extract_name_from_selfie_text(best_text) or None
        result["dob"] = _extract_dob_from_selfie_text(best_text) or None
        result["confidence"] = float(best_confidence)
        result["text_redacted"] = _redact_sensitive_ids(best_text)

        if best_confidence < settings.doc_in_selfie_min_confidence:
            result["error"] = "selfie_ocr_low_confidence"
        return result

    def _run_paddleocr(self, image: Any) -> Any:
        engine = self.ocr_engine
        # PaddleOCR 2.x exposes .ocr(img, cls=...). 3.x still accepts .ocr but
        # the supported entry point is .predict(); try both for compatibility.
        try:
            return engine.ocr(image, cls=False)
        except TypeError:
            pass
        try:
            return engine.ocr(image)
        except TypeError:
            pass
        return engine.predict(image)

    def _detect_largest_face(self, image: Any) -> tuple[Any, Any]:
        """Return ``(possibly_resized_image, face_row)`` for the largest face."""
        cv2 = self._cv2
        height, width = image.shape[:2]
        if max(height, width) > settings.face_detect_max_side:
            scale = settings.face_detect_max_side / float(max(height, width))
            image = cv2.resize(
                image,
                (max(1, int(width * scale)), max(1, int(height * scale))),
                interpolation=cv2.INTER_AREA,
            )
            height, width = image.shape[:2]

        self.face_detector.setInputSize((width, height))
        _, faces = self.face_detector.detect(image)
        if faces is None or len(faces) == 0:
            return image, None

        face = max(faces, key=lambda row: float(row[2]) * float(row[3]))
        return image, face

    def _face_embedding(self, image: Any, face: Any) -> Any:
        np = self._np
        aligned = self.face_recognizer.alignCrop(image, face)
        feature = self.face_recognizer.feature(aligned)
        return np.asarray(feature, dtype=np.float32).reshape(-1)

    def _reference_quality(self, image: Any, face: Any) -> dict[str, Any]:
        """Quality metrics + gate decision for one reference face (Phase 5).

        ``face`` is the full YuNet row (NOT a hand-crop): width/height/score
        columns are used as detected, so the metrics match the crop that
        ``_face_embedding`` feeds to SFace's ``alignCrop``. Sharpness is the
        Laplacian variance of the grayscale face box.

        A reference failing any gate reports ``usable=False``; the caller skips
        it entirely (a tiny/blurry crop yields garbage embeddings that could
        look spuriously similar to anything).
        """
        face_width = float(face[2])
        face_height = float(face[3])
        # YuNet rows are 15 floats; the detection score is the last column.
        detection_score = float(face[14]) if len(tuple(face)) > 14 else 1.0

        # Laplacian variance over the exact face box (fail-safe: any cv2/
        # geometry problem marks the reference unusable).
        sharpness = 0.0
        try:
            height, image_width = image.shape[:2]
            x1 = max(int(float(face[0])), 0)
            y1 = max(int(float(face[1])), 0)
            x2 = min(int(float(face[0]) + face_width), image_width)
            y2 = min(int(float(face[1]) + face_height), height)
            if x2 > x1 and y2 > y1:
                cv2 = self._cv2
                gray = cv2.cvtColor(image[y1:y2, x1:x2], cv2.COLOR_BGR2GRAY)
                sharpness = float(cv2.Laplacian(gray, cv2.CV_64F).var())
        except Exception:  # noqa: BLE001 - the gate must never crash a job
            logger.exception("Reference face sharpness measurement failed")
            sharpness = 0.0

        usable = (
            min(face_width, face_height) >= settings.face_min_reference_side
            and detection_score >= settings.face_min_detection_score
            and sharpness >= settings.face_min_sharpness
        )
        return {
            "width": int(round(face_width)),
            "height": int(round(face_height)),
            "detection_score": round(detection_score, 4),
            "sharpness": round(sharpness, 2),
            "usable": bool(usable),
        }

    def run_face_match(
        self, documents: list[dict[str, str]], cache: ImageCache
    ) -> dict[str, Any]:
        """Multi-reference face matching (Phase 5).

        Every government-type document (aadhaar / pan_card / legacy govt_id
        aliases) is scored against the selfie independently; the highest cosine
        wins as ``face_similarity``. Per-reference quality gates skip unusable
        reference photos, and the Aadhaar-vs-winner divergence guard plus the
        ``aadhaar_similarity`` floor are enforced downstream (Python priority
        logic and the Go router) as defence-in-depth.
        """
        document_types = [doc.get("type", "") for doc in documents]
        reference_docs = [
            doc for doc in documents if doc.get("type") in GOVT_DOC_TYPES
        ]
        selfie = _find_document(documents, SELFIE_DOC_TYPES)

        if not reference_docs or selfie is None:
            return _face_error(
                "missing_document",
                "both a government ID and a selfie are required",
                document_types,
            )

        if self.face_detector is None or self.face_recognizer is None:
            return _face_error(
                "model_unavailable",
                self.errors.get("face_detector")
                or self.errors.get("face_recognizer")
                or "face models not loaded",
                document_types,
            )

        # The selfie is the anchor of every comparison: a failure here is
        # global (as in Phase 3), while per-reference failures only drop that
        # reference.
        try:
            selfie_image = cache.get(selfie["url"])
        except ImageFetchError as exc:
            return _face_error("image_fetch_failed", str(exc), document_types)

        try:
            selfie_image, selfie_face = self._detect_largest_face(selfie_image)
        except Exception as exc:  # noqa: BLE001 - per-job failure only
            logger.exception("Face detection failed")
            return _face_error(
                "face_detection_failed",
                f"{type(exc).__name__}: {exc}",
                document_types,
            )

        selfie_detected = selfie_face is not None
        if not selfie_detected:
            return _face_error(
                "no_face_in_selfie",
                "",
                document_types,
                selfie_detected=False,
            )

        try:
            selfie_embedding = self._face_embedding(selfie_image, selfie_face)
        except Exception as exc:  # noqa: BLE001 - per-job failure only
            logger.exception("SFace embedding failed")
            return _face_error(
                "face_embedding_failed",
                f"{type(exc).__name__}: {exc}",
                document_types,
                selfie_detected=True,
            )

        quality_by_ref: dict[str, Any] = {}
        scores_by_ref: dict[str, float] = {}
        handled_types: set[str] = set()
        for ref_doc in reference_docs:
            ref_type = str(ref_doc.get("type") or "").strip() or "unknown"
            if ref_type in handled_types:
                # Deduplicate alias uploads; the first document of a type wins
                # (matches the backend's one-upload-per-type contract).
                continue
            handled_types.add(ref_type)

            ref_log = _safe_url(ref_doc.get("url") or "")

            try:
                ref_image = cache.get(ref_doc["url"])
            except ImageFetchError:
                logger.warning(
                    "kyc.face_reference_skipped ref=%s reason=image_fetch_failed url=%s",
                    ref_type,
                    ref_log,
                )
                quality_by_ref[ref_type] = {
                    "usable": False,
                    "error": "image_fetch_failed",
                }
                continue

            try:
                ref_image, ref_face = self._detect_largest_face(ref_image)
            except Exception as exc:  # noqa: BLE001 - drop only this reference
                logger.warning(
                    "kyc.face_reference_skipped ref=%s reason=face_detection_failed "
                    "detail=%s url=%s",
                    ref_type,
                    type(exc).__name__,
                    ref_log,
                )
                quality_by_ref[ref_type] = {
                    "usable": False,
                    "error": "face_detection_failed",
                }
                continue

            if ref_face is None:
                quality_by_ref[ref_type] = {
                    "usable": False,
                    "error": "no_face_in_reference",
                }
                continue

            ref_quality = self._reference_quality(ref_image, ref_face)
            quality_by_ref[ref_type] = ref_quality
            if not ref_quality["usable"]:
                logger.warning(
                    "kyc.face_reference_skipped ref=%s reason=quality_gate "
                    "width=%d height=%d detection_score=%.4f sharpness=%.2f",
                    ref_type,
                    ref_quality["width"],
                    ref_quality["height"],
                    ref_quality["detection_score"],
                    ref_quality["sharpness"],
                )
                continue

            try:
                np = self._np
                ref_embedding = self._face_embedding(ref_image, ref_face)
                denominator = float(
                    np.linalg.norm(ref_embedding) * np.linalg.norm(selfie_embedding)
                )
                if denominator <= 1e-9:
                    raise ValueError("zero-length face embedding")
                cosine = float(
                    np.dot(ref_embedding, selfie_embedding) / denominator
                )
            except Exception as exc:  # noqa: BLE001 - drop only this reference
                logger.exception("SFace embedding failed")
                quality_by_ref[ref_type] = {
                    **ref_quality,
                    "usable": False,
                    "error": "face_embedding_failed",
                }
                continue

            scores_by_ref[ref_type] = cosine

        if not scores_by_ref:
            return _face_error(
                "no_usable_reference",
                "",
                document_types,
                selfie_detected=True,
                quality=quality_by_ref,
            )

        return _build_face_match_result(
            document_types=document_types,
            scores_by_ref=scores_by_ref,
            quality_by_ref=quality_by_ref,
            selfie_detected=True,
        )

    def run_liveness(
        self, documents: list[dict[str, str]], cache: ImageCache
    ) -> dict[str, Any]:
        selfie = _find_document(documents, SELFIE_DOC_TYPES)
        if selfie is None:
            return _liveness_result(
                0.0,
                selfie_present=False,
                face_detected=False,
                error="missing_selfie",
            )

        if self.liveness_session is None:
            return _liveness_result(
                0.0,
                selfie_present=True,
                face_detected=False,
                error="liveness_model_unavailable",
                detail=self.errors.get("liveness", "model not loaded"),
            )

        if self.face_detector is None:
            return _liveness_result(
                0.0,
                selfie_present=True,
                face_detected=False,
                error="face_detector_unavailable",
                detail=self.errors.get("face_detector", "face detector not loaded"),
            )

        try:
            image = cache.get(selfie["url"])
        except ImageFetchError as exc:
            return _liveness_result(
                0.0,
                selfie_present=True,
                face_detected=False,
                error="image_fetch_failed",
                detail=str(exc),
            )

        try:
            image, face = self._detect_largest_face(image)
        except Exception as exc:  # noqa: BLE001 - per-job failure only
            logger.exception("Liveness face detection failed")
            return _liveness_result(
                0.0,
                selfie_present=True,
                face_detected=False,
                error="face_detection_failed",
                detail=f"{type(exc).__name__}: {exc}",
            )

        if face is None:
            return _liveness_result(
                0.0,
                selfie_present=True,
                face_detected=False,
                error="no_face_in_selfie",
            )

        try:
            crop = _crop_face(image, face, settings.liveness_crop_scale)
            if crop.size == 0:
                raise ValueError("empty face crop")

            height, width = self.liveness_input_size
            if crop.shape[0] != height or crop.shape[1] != width:
                crop = self._cv2.resize(crop, (width, height))

            values = crop.astype(self._np.float32)
            if settings.liveness_normalize == "div255":
                values = values / 255.0
            elif settings.liveness_normalize == "pm1":
                values = values / 127.5 - 1.0

            tensor = self._np.transpose(values, (2, 0, 1))[None, ...]
            outputs = self.liveness_session.run(
                [self.liveness_output_name],
                {self.liveness_input_name: tensor},
            )
            probabilities = _softmax(outputs[0])
            index = min(settings.liveness_live_index, len(probabilities) - 1)
            score = float(probabilities[index])
        except Exception as exc:  # noqa: BLE001 - per-job failure only
            logger.exception("MiniFASNetV2 inference failed")
            return _liveness_result(
                0.0,
                selfie_present=True,
                face_detected=True,
                error="liveness_inference_failed",
                detail=f"{type(exc).__name__}: {exc}",
            )

        return _liveness_result(
            score,
            selfie_present=True,
            face_detected=True,
            probabilities=[round(float(p), 4) for p in probabilities],
        )


_default_model_loader: ModelLoader | None = None
_default_model_loader_lock = threading.Lock()


def get_model_loader() -> ModelLoader:
    global _default_model_loader
    if _default_model_loader is None:
        with _default_model_loader_lock:
            if _default_model_loader is None:
                _default_model_loader = ModelLoader()
    return _default_model_loader


def _log_document_validations(
    job_id: str,
    ocr_result: dict[str, Any],
) -> dict[str, Any]:
    """Phase 1: observational only.

    Logs Aadhaar/PAN validation metadata already computed from the raw OCR
    text. Returns the result dict, but the caller MUST NOT use it to influence
    ``determine_review_priority`` or any routing decision.
    """
    aadhaar_valid = bool(ocr_result.get("aadhaar_valid"))
    aadhaar_source = str(ocr_result.get("aadhaar_source") or "none")
    aadhaar_last4 = ocr_result.get("aadhaar_last4") or None
    pan_valid = bool(ocr_result.get("pan_valid"))

    logger.info(
        "kyc.doc_validation job_id=%s aadhaar_valid=%s aadhaar_source=%s "
        "pan_valid=%s",
        job_id,
        aadhaar_valid,
        aadhaar_source,
        pan_valid,
        extra={
            "event": "kyc.doc_validation",
            "job_id": job_id,
            "aadhaar_valid": aadhaar_valid,
            "aadhaar_source": aadhaar_source,
            "aadhaar_last4": aadhaar_last4,
            "pan_valid": pan_valid,
        },
    )
    return {
        "aadhaar_valid": aadhaar_valid,
        "aadhaar_source": aadhaar_source,
        "pan_valid": pan_valid,
    }


# ---------------------------------------------------------------------------
# Aadhaar Secure QR verification (Phase 2.5 — plumbing only)
# ---------------------------------------------------------------------------
#
# Phase 2.5 populates QR metadata inside ``ocr_ocr_result.document_validation``
# but does NOT feed it into ``determine_review_priority``. Nothing in this
# section may influence routing yet.

# Pinned UIDAI certificates live next to the service, never in a temp path.
# They are loaded at most once per process; a missing directory is a normal
# state until Phase 5 ships a real certificate.
_UIDAI_CERT_DIR = (
    Path(__file__).resolve().parent.parent / "certs" / "uidai"
)


class _QRDecodeTimeout(Exception):
    """Internal marker: the QR decode exceeded ``qr_max_decode_seconds``."""


def _new_document_validation() -> dict[str, Any]:
    """Return the complete, JSON-serializable document_validation shape."""
    return {
        "qr_attempted": False,
        "qr_decoded": False,
        "qr_signature_verified": False,
        "qr_error": None,
        "matched_cert_name": None,
        "photo_format": None,
        "qr_photo_extracted": False,
        "qr_reference_last4": None,
        "qr_reference_matches_aadhaar": None,
        "overall": "not_attempted",
        "attempt_duration_ms": 0,
    }


def _fetch_document_bytes(url: str) -> bytes:
    """Fetch raw document bytes for QR decoding.

    Mirrors :func:`fetch_image`'s transport limits but returns the original
    encoded bytes: :func:`qr_decode.decode_qr_text` performs its own image
    decoding (PIL/zxing-cpp) and does not need a cv2 ``ndarray``.
    """
    if not url:
        raise ImageFetchError("document URL is empty")

    parsed = urlparse(url)
    max_bytes = settings.max_image_bytes

    if parsed.scheme == "file":
        path = Path(unquote(parsed.path))
        try:
            if path.stat().st_size > max_bytes:
                raise ImageFetchError(
                    f"image exceeds MAX_IMAGE_BYTES ({max_bytes} bytes)"
                )
            data = path.read_bytes()
        except ImageFetchError:
            raise
        except OSError as exc:
            raise ImageFetchError(f"cannot read local image: {exc}") from exc
    elif parsed.scheme in {"http", "https"}:
        import requests

        try:
            with requests.get(
                url,
                stream=True,
                timeout=(settings.image_connect_timeout, settings.image_download_timeout),
                headers={"User-Agent": "techguild-kyc-ml/1.0"},
            ) as response:
                response.raise_for_status()
                chunks: list[bytes] = []
                total = 0
                for chunk in response.iter_content(chunk_size=64 * 1024):
                    if not chunk:
                        continue
                    total += len(chunk)
                    if total > max_bytes:
                        raise ImageFetchError(
                            f"image exceeds MAX_IMAGE_BYTES ({max_bytes} bytes)"
                        )
                    chunks.append(chunk)
                data = b"".join(chunks)
        except ImageFetchError:
            raise
        except Exception as exc:  # noqa: BLE001 - normalize to a domain error
            raise ImageFetchError(
                f"download failed for {_safe_url(url)}: {exc}"
            ) from exc
    else:
        raise ImageFetchError(f"unsupported document URL scheme: {parsed.scheme!r}")

    if not data:
        raise ImageFetchError("document is empty")

    return data


def _fetch_primary_document_bytes(
    documents: list[dict[str, str]],
) -> bytes | None:
    """Fetch the government document's raw bytes ONCE per job.

    Uses the same URL validation (SSRF, size limits, timeout) as the existing
    :func:`_fetch_document_bytes`. Returns ``None`` if no government document is
    present or if the fetch fails, so callers can fall back to their previous
    behaviour without turning a fetch failure into a job failure.
    """
    document = _first_document_of_type(documents, GOVT_DOC_TYPES)
    if document is None:
        return None
    url = document.get("url")
    if not url:
        return None
    try:
        return _fetch_document_bytes(url)
    except Exception:  # noqa: BLE001 - failures must degrade, never raise
        return None


@lru_cache(maxsize=1)
def _load_trusted_certs() -> tuple[TrustedCert, ...] | None:
    """Load pinned UIDAI certs once. ``None`` means verification is unavailable."""
    try:
        certs = load_pinned_certs(_UIDAI_CERT_DIR)
    except (ValueError, OSError):
        logger.warning("kyc.qr_certs_unavailable dir=%s", _UIDAI_CERT_DIR)
        return None
    if not certs:
        logger.warning("kyc.qr_certs_empty dir=%s", _UIDAI_CERT_DIR)
        return None
    return tuple(certs)


def _decode_qr_with_timeout(image_bytes: bytes, timeout_seconds: float) -> str:
    """Run :func:`decode_qr_text` with a wall-clock bound.

    A daemon thread is used instead of ``ThreadPoolExecutor`` because a decode
    stuck in native code cannot be cancelled; a non-daemon worker would keep the
    interpreter alive at shutdown. On timeout the orphaned daemon thread is
    abandoned and its eventual result is ignored.
    """
    outcome: dict[str, Any] = {}

    def _target() -> None:
        try:
            outcome["result"] = decode_qr_text(image_bytes)
        except Exception as exc:  # noqa: BLE001 - handed back to the caller
            outcome["error"] = exc

    thread = threading.Thread(target=_target, name="kyc-qr-decode", daemon=True)
    thread.start()
    thread.join(timeout_seconds)

    if thread.is_alive():
        raise _QRDecodeTimeout()

    error = outcome.get("error")
    if error is not None:
        raise error

    result = outcome.get("result")
    if not result:
        raise _QRDecodeTimeout()
    text, _engine = result
    return str(text)


def _reference_matches(qr_last4: Any, ocr_last4: Any) -> bool | None:
    """Compare QR reference digits with the Aadhaar last-4 from OCR text."""
    qr_normalized = str(qr_last4).strip() if qr_last4 else ""
    ocr_normalized = str(ocr_last4).strip() if ocr_last4 else ""
    if not qr_normalized or not ocr_normalized:
        return None
    return qr_normalized == ocr_normalized


def run_document_validation(
    job: dict[str, Any],
    ocr_result: dict[str, Any],
    primary_bytes: bytes | None = None,
) -> dict[str, Any]:
    """Attempt Aadhaar QR decode + verification for the government document.

    Phase 2.5: the returned dict is stored under
    ``ocr_result["document_validation"]`` (and therefore inside the existing
    ``ocr_ocr_result`` JSONB column). It MUST NOT influence review routing.

    Never raises. Unexpected failures are reported as ``overall="failed"`` with
    ``qr_error="unexpected_exception"``; the exception type is logged but its
    message is not (library messages can embed PII).
    """
    started = time.perf_counter()
    result = _new_document_validation()

    def _finish() -> dict[str, Any]:
        duration_ms = max(int((time.perf_counter() - started) * 1000), 0)
        result["attempt_duration_ms"] = duration_ms
        fetch_reused = primary_bytes is not None
        try:
            logger.info(
                "kyc.qr_validation job_id=%s overall=%s duration_ms=%d error=%s "
                "fetch_reused=%s",
                job.get("job_id") or "unknown",
                result["overall"],
                duration_ms,
                result["qr_error"],
                fetch_reused,
            )
        except Exception:  # noqa: BLE001 - logging must never fail a job
            pass
        return result

    try:
        if settings.ml_mode == "mock" or not settings.qr_decode_enabled:
            result["overall"] = "not_attempted"
            return _finish()

        result["qr_attempted"] = True

        documents = job.get("documents") or []
        document = _first_document_of_type(documents, GOVT_DOC_TYPES)
        if document is None or not str(document.get("url") or "").strip():
            result["overall"] = "failed"
            result["qr_error"] = "no_document"
            return _finish()

        certs = _load_trusted_certs()
        if not certs:
            result["overall"] = "failed"
            result["qr_error"] = "no_trusted_certs"
            return _finish()

        if primary_bytes is not None:
            image_bytes = primary_bytes
        else:
            try:
                image_bytes = _fetch_document_bytes(str(document.get("url") or ""))
            except Exception:  # noqa: BLE001 - document fetch failures never fail a job
                result["overall"] = "failed"
                result["qr_error"] = "image_fetch_failed"
                return _finish()

        try:
            qr_text = _decode_qr_with_timeout(
                image_bytes, settings.qr_max_decode_seconds
            )
        except _QRDecodeTimeout:
            result["overall"] = "failed"
            result["qr_error"] = "timeout"
            return _finish()
        except QRVerificationError as exc:
            result["overall"] = "failed"
            result["qr_error"] = exc.code
            return _finish()

        result["qr_decoded"] = True

        try:
            verified: QRResult = verify_aadhaar_qr(qr_text, list(certs))
        except QRVerificationError as exc:
            result["qr_error"] = exc.code
            if exc.code == "qr_signature_invalid":
                result["overall"] = "unverified"
            else:
                result["overall"] = "failed"
            return _finish()

        result.update(
            overall="verified",
            qr_signature_verified=bool(verified.signature_valid),
            matched_cert_name=verified.matched_cert_name,
            photo_format=verified.photo_format,
            # Success guarantees a non-empty, recognized photo; do not transport
            # the bytes themselves in Phase 2.5.
            qr_photo_extracted=verified.photo_format in {"jpeg", "jpeg2000"},
            qr_reference_last4=verified.qr_reference_last4,
        )
        result["qr_reference_matches_aadhaar"] = _reference_matches(
            verified.qr_reference_last4, ocr_result.get("aadhaar_last4")
        )
        return _finish()

    except Exception as exc:  # noqa: BLE001 - contract: this function never raises
        logger.warning(
            "kyc.qr_validation_unexpected job_id=%s error_type=%s",
            job.get("job_id") or "unknown",
            type(exc).__name__,
        )
        result["overall"] = "failed"
        result["qr_error"] = "unexpected_exception"
        result["qr_signature_verified"] = False
        return _finish()


def determine_review_priority(
    ocr_result: dict[str, Any],
    face_result: dict[str, Any],
    liveness_result: dict[str, Any],
) -> tuple[str, str]:
    """Apply the KYC threshold policy and return ``(effective, raw)``.

    Contract with the Go router (Phase 3):

    * ``auto_reject``    - a hard fraud signal (liveness/face below reject)
    * ``auto_eligible``  - every score is at/above its pass threshold
    * ``manual_review``  - borderline; a human must decide

    The returned tuple separates the two concerns:

    * ``raw_priority``       - what the thresholds say, before any shadow
      policy is applied. Persisted in ``ai_recommended_priority`` so the Go
      router can re-gate it against the DB-level evidence guard.
    * ``effective_priority`` - what is written to ``review_priority``
      (backwards compatibility until the Go router is the only decision
      writer). In shadow mode this is always ``manual_review``; the raw
      decision is logged as ``kyc.shadow_priority``, never acted upon.
    """
    face_similarity = _as_float(face_result.get("face_similarity"))
    liveness_score = _as_float(liveness_result.get("liveness_score"))
    ocr_confidence = _as_float(ocr_result.get("ocr_confidence"))
    ocr_text = str(ocr_result.get("text") or "").strip()

    # A stage that could not run (missing document, corrupted image, model
    # unavailable, inference failure) reports score 0.0. That is missing
    # evidence, not a fraud signal, so it must never trigger auto_reject.
    # This matters especially for business KYC, which has no selfie.
    face_error = bool(face_result.get("error"))
    liveness_error = bool(liveness_result.get("error"))
    ocr_error = bool(ocr_result.get("error"))

    # Hard fraud gate: only a *successful* inference scoring below the reject
    # threshold auto-rejects. A borderline score (reject <= score < pass) must
    # stay in the manual queue, not be silently rejected.
    # Face matching disabled (FACE_MATCH_ENABLED=false): the reported 0.0 is
    # "no evidence", not a fraud signal. Guard first so the reject branch below
    # can never turn a disabled stage into an auto_reject.
    # Phase G: a business KYC record has no selfie and no liveness stage, so the
    # selected-document + cheque verdict is its whole decision. The business
    # branch is checked FIRST so a missing selfie/liveness can never turn a
    # business record into an auto_reject, and anything other than a confirmed
    # approve is manual. The canonical Phase G key is read first, with the
    # Phase E flat mirror as the fallback for records written before it.
    business_decision = str(
        face_result.get("business_decision")
        or face_result.get("business_docs_decision")
        or ""
    ).strip()
    if business_decision:
        if business_decision == "approve":
            raw_priority = PRIORITY_AUTO_ELIGIBLE
            logger.info(
                "kyc.business_docs_approved raw=auto_eligible score=%s",
                face_result.get("business_docs_score"),
            )
        else:
            raw_priority = PRIORITY_MANUAL_REVIEW
            logger.info(
                "kyc.business_docs_manual raw=manual_review decision=%s",
                business_decision,
            )
    elif (
        face_result.get("document_in_selfie_decision") == "approve"
        and settings.ml_mode == "real"
    ):
        # Phase B: with FACE_MATCH_ENABLED=false the document-in-selfie verdict
        # is an individual record's identity anchor, and the Go router only
        # auto-approves when this AI recommendation is auto_eligible.
        # Mock mode is excluded on purpose: it never touched a real image, so a
        # synthetic verdict must not promote a job (this is the routing the
        # Phase A test test_determine_review_priority_unchanged pins).
        raw_priority = PRIORITY_AUTO_ELIGIBLE
        logger.info("kyc.doc_in_selfie_approved raw=auto_eligible")
    elif not face_result.get("enabled", True) or (
        not settings.face_match_enabled
        and face_result.get("error") == "face_match_disabled"
    ):
        raw_priority = PRIORITY_MANUAL_REVIEW
        logger.info(
            "kyc.face_disabled raw=manual_review reason=face_match_disabled"
        )
    elif not face_error and face_similarity < settings.threshold_face_reject:
        raw_priority = PRIORITY_AUTO_REJECT
    elif not liveness_error and liveness_score < settings.threshold_live_reject:
        raw_priority = PRIORITY_AUTO_REJECT
    # Incomplete evidence (failed stage, missing/empty OCR output) always needs
    # a human; never auto-approve on it even if a malformed upstream result
    # carries a high confidence value.
    elif face_error or liveness_error or ocr_error or not ocr_text:
        raw_priority = PRIORITY_MANUAL_REVIEW
    # Phase 5 "second reference corroboration": when a non-Aadhaar reference
    # wins the comparison but the (QR-anchored) Aadhaar score is severely
    # weak, treat the winning score as suspicious (fake PAN carrying the match
    # for another person's Aadhaar) and force a human. Hard fraud signals
    # above (auto_reject) still outrank this cap.
    elif face_result.get("second_reference_divergence"):
        raw_priority = PRIORITY_MANUAL_REVIEW
        winning_reference = str(face_result.get("winning_reference") or "unknown")
        aadhaar_score = _as_float(face_result.get("aadhaar_similarity"))
        logger.info(
            "kyc.face_divergence winner=%s max=%.3f aadhaar=%.3f",
            winning_reference,
            face_similarity,
            aadhaar_score,
            extra={
                "event": "kyc.face_divergence",
                "winning_reference": winning_reference,
                "max_score": face_similarity,
                "aadhaar_similarity": aadhaar_score,
            },
        )
    elif (
        face_similarity >= settings.threshold_face_auto_pass
        and liveness_score >= settings.threshold_live_pass
        and ocr_confidence >= settings.threshold_ocr_pass
    ):
        raw_priority = PRIORITY_AUTO_ELIGIBLE
    else:
        raw_priority = PRIORITY_MANUAL_REVIEW

    if settings.shadow_mode:
        effective_priority = PRIORITY_MANUAL_REVIEW
        logger.info(
            "kyc.shadow_priority raw=%s face_similarity=%s liveness_score=%s ocr_confidence=%s",
            raw_priority,
            face_similarity,
            liveness_score,
            ocr_confidence,
            extra={
                "event": "kyc.shadow_priority",
                "effective_priority": effective_priority,
                "raw_priority": raw_priority,
                "face_similarity": face_similarity,
                "liveness_score": liveness_score,
                "ocr_confidence": ocr_confidence,
            },
        )
        return effective_priority, raw_priority

    return raw_priority, raw_priority


# ---------------------------------------------------------------------------
# PostgreSQL writes
# ---------------------------------------------------------------------------


def mark_processing(verification_id: str) -> bool:
    """Move the record into the ``processing`` ML state.

    Returns True when the record was moved, False when the update matched no
    row. The guards make this idempotent:

    * ``ml_status IN ('queued', 'processing')`` — a duplicate/redelivered
      message cannot drag a record that is already ``ml_completed`` (or
      ``ml_failed``) back into ``processing``.
    * ``system_escalated_at IS NULL`` — a record the stale-pending escalator
      has escalated is terminal at the DB level; the worker stops processing
      and the caller acks instead of retrying (stale-escalation race, P0 #1).

    JobProcessingError still escapes for transient DB failures so the normal
    redelivery path keeps working.
    """
    with db_connection() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                UPDATE verification_records
                   SET ml_status = 'processing',
                       ml_other_reason_code = NULL,
                       updated_at = NOW()
                 WHERE id = %s::uuid
                   AND ml_status IN ('queued', 'processing')
                   AND system_escalated_at IS NULL
                """,
                (verification_id,),
            )
            if cur.rowcount == 0:
                logger.info(
                    "kyc.mark_processing_noop job_id=%s",
                    verification_id,
                    extra={
                        "event": "kyc.mark_processing_noop",
                        "verification_id": verification_id,
                    },
                )
                return False
            return True


def save_results(
    verification_id: str,
    ocr_result: dict[str, Any],
    face_result: dict[str, Any],
    liveness_result: dict[str, Any],
    review_priority: str,
    ai_recommended_priority: str,
    processing_time_ms: int,
    ml_vendor: str = ML_VENDOR,
    ml_model_version: str = ML_MODEL_VERSION,
) -> bool:
    """Persist all ML results and the review priority in one transaction.

    Returns True when the row was updated, False on a no-op (duplicate,
    already-escalated, or already-routed record). Callers ack the message on
    False instead of retrying.

    Column notes:

    * ``review_priority`` — kept in sync for backward compatibility (admin
      queues read it) until the Go router is the only decision writer.
    * ``ai_recommended_priority`` — the raw threshold-only decision, saved
      *separately* from the effective priority so the Go router can re-gate
      and enrich it (human decisions and AI recommendations stay independent).
    * ``routing_state = 'pending'`` — hands the record to the Go router. The
      guards below ensure a late/duplicate completion can never overwrite a
      record the router or the stale-pending escalator already processed.
    * ``policy_version`` — recorded so decisions can be traced back to the
      exact policy values in force.
    """
    # NOTE: The Go model field is `OCROcrResult`, which GORM maps to the
    # column `ocr_ocr_result` (not `ocr_result`). Using the real column name is
    # required for the shared database schema.
    policy_version = settings.policy_version
    sql = """
        UPDATE verification_records
           SET ml_status = 'ml_completed',
               ml_completed_at = NOW(),
               ml_vendor = %(ml_vendor)s,
               ml_model_version = %(ml_model_version)s,
               ml_processing_time = %(processing_time_ms)s,
               ml_other_reason_code = NULL,
               ocr_ocr_result = %(ocr_result)s::jsonb,
               face_match_result = %(face_match_result)s::jsonb,
               liveness_result = %(liveness_result)s::jsonb,
               ai_recommended_priority = %(ai_recommended_priority)s,
               review_priority = %(review_priority)s,
               routing_state = 'pending',
               policy_version = %(policy_version)s,
               updated_at = NOW()
         WHERE id = %(verification_id)s::uuid
           AND ml_status <> 'ml_completed'
           AND system_escalated_at IS NULL
           AND routing_state = 'pending'
    """
    params = {
        "ml_vendor": ml_vendor,
        "ml_model_version": ml_model_version,
        "processing_time_ms": max(int(processing_time_ms), 0),
        "ocr_result": json.dumps(ocr_result),
        "face_match_result": json.dumps(face_result),
        "liveness_result": json.dumps(liveness_result),
        "ai_recommended_priority": ai_recommended_priority,
        "review_priority": review_priority,
        "policy_version": policy_version,
        "verification_id": verification_id,
    }

    with db_connection() as conn:
        with conn.cursor() as cur:
            cur.execute(sql, params)
            if cur.rowcount == 0:
                # No-op: the record is already ml_completed (duplicate), was
                # escalated by the stale-pending cron, or was already routed.
                # Ack and never retry — retrying would fight the Go router.
                logger.info(
                    "kyc.save_results_noop job_id=%s reason=already_completed_or_missing",
                    verification_id,
                    extra={
                        "event": "kyc.save_results_noop",
                        "verification_id": verification_id,
                    },
                )
                return False
            return True


def mark_failed(verification_id: str, reason_code: str) -> None:
    """Best-effort update to ``ml_failed`` after a processing error.

    A duplicate/redelivered job must not downgrade an already-completed record,
    so the update is guarded with ``ml_status <> 'ml_completed'``.
    """
    reason = (reason_code or "worker_error")[:50]
    with db_connection() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                UPDATE verification_records
                   SET ml_status = 'ml_failed',
                       ml_other_reason_code = %s,
                       updated_at = NOW()
                 WHERE id = %s::uuid
                   AND ml_status <> 'ml_completed'
                """,
                (reason, verification_id),
            )


# ---------------------------------------------------------------------------
# Job processing
# ---------------------------------------------------------------------------


def process_job(
    payload: str | bytes | dict[str, Any],
    model_loader: ModelLoader | None = None,
) -> dict[str, Any]:
    """Run the full pipeline for one stream message and persist the outcome."""
    job = parse_job_payload(payload)
    verification_id = job["verification_id"]
    job_id = job["job_id"] or "unknown"

    loader = model_loader or get_model_loader()

    if settings.ml_mode == "real":
        # Idempotent: the worker normally preloads in start(), but direct
        # callers (tests/CLI) should still work.
        loader.load_models()
        if not loader.core_ready:
            raise JobProcessingError(
                "AI models are not loaded: "
                + json.dumps(loader.errors, sort_keys=True)
            )

    logger.info(
        "Processing KYC job %s (verification_id=%s, documents=%d, mode=%s)",
        job_id,
        verification_id,
        len(job["documents"]),
        settings.ml_mode,
    )

    started = time.perf_counter()
    if not mark_processing(verification_id):
        # Duplicate/redelivered message or the record has already been
        # escalated by the stale-pending cron. Nothing to process: return the
        # skip marker so the consumer ACKs and never retries (which would
        # fight the Go router / escalator).
        logger.info(
            "kyc.job_skipped job_id=%s verification_id=%s reason=mark_processing_noop",
            job_id,
            verification_id,
        )
        return {
            "job_id": job_id,
            "verification_id": verification_id,
            "user_id": job["user_id"],
            "ml_status": "noop",
            "skipped": True,
        }

    primary_bytes: bytes | None = None
    # Only real mode builds a shared cache; mock mode never fetches images, so
    # the document-in-selfie stage creates its own cache if it needs one.
    cache: ImageCache | None = None
    if settings.ml_mode == "mock":
        ocr_result = mock_ocr(job["documents"])
        if settings.face_match_enabled:
            face_result = mock_face_match(job["documents"])
        else:
            face_result = _make_face_disabled_result(
                doc_types=[
                    d.get("type", "unknown") for d in job.get("documents", [])
                ]
            )
        liveness_result = mock_liveness(job["documents"])
        ml_vendor = ML_VENDOR_MOCK
        ml_model_version = ML_MODEL_VERSION_MOCK
    else:
        cache = ImageCache()
        # Phase 4.5: download the government document once and share the bytes
        # with both OCR and QR validation. Mock mode never reaches here.
        primary_bytes = _fetch_primary_document_bytes(job["documents"])
        ocr_result = loader.run_ocr(
            job["documents"], cache, primary_bytes=primary_bytes
        )
        if settings.face_match_enabled:
            face_result = loader.run_face_match(job["documents"], cache)
        else:
            face_result = _make_face_disabled_result(
                doc_types=[
                    d.get("type", "unknown") for d in job.get("documents", [])
                ]
            )
        liveness_result = loader.run_liveness(job["documents"], cache)
        ml_vendor = ML_VENDOR_REAL
        ml_model_version = loader.model_version

    # Phase A: document-in-selfie verification. The Go router gates
    # auto-approval on document_in_selfie_decision/score/match, so these keys
    # must be written for both the real and the mock pipeline.
    if settings.doc_in_selfie_enabled:
        try:
            selfie_ocr = loader.run_selfie_document_ocr(job["documents"], cache)
        except Exception as exc:  # noqa: BLE001 - must never fail the job
            logger.warning("selfie_ocr_exception error=%s", type(exc).__name__)
            selfie_ocr = {
                "error": "selfie_ocr_exception",
                "aadhaar_number": None,
                "name": None,
                "dob": None,
                "confidence": 0.0,
                "text_redacted": "",
            }

        uploaded_ocr = {
            # The full 12-digit number travels only in memory: it is what lets
            # compare_documents verify identity instead of comparing tails.
            "_internal_aadhaar_number": ocr_result.get(
                "_internal_aadhaar_number"
            ),
            # Everything else on this side stays redacted.
            "aadhaar_number": ocr_result.get("aadhaar_number"),
            "aadhaar_last4": ocr_result.get("aadhaar_last4"),
            "name": ocr_result.get("name"),
            "dob": ocr_result.get("dob"),
        }

        comparison = compare_documents(
            uploaded_ocr,
            selfie_ocr,
            config={
                "approve_threshold": settings.doc_in_selfie_approve_threshold,
                "manual_threshold": settings.doc_in_selfie_manual_threshold,
                "reject_on_mismatch": settings.doc_in_selfie_reject_on_mismatch,
                "name_match_threshold": settings.doc_in_selfie_name_match_threshold,
            },
        )

        face_result["document_in_selfie"] = {
            "attempted": True,
            "error": selfie_ocr.get("error"),
            "selfie_ocr": _selfie_ocr_for_storage(selfie_ocr),
            "uploaded_ocr": uploaded_ocr,
            "comparison": comparison,
        }
        face_result["document_in_selfie_decision"] = comparison["decision"]
        face_result["document_in_selfie_score"] = comparison["score"]
        face_result["document_in_selfie_match"] = (
            comparison["decision"] == "approve"
        )
    else:
        face_result["document_in_selfie"] = {"attempted": False}
        face_result["document_in_selfie_decision"] = "manual"
        face_result["document_in_selfie_score"] = 0
        face_result["document_in_selfie_match"] = False

    # Phase G: business KYC. A business job has no selfie at all, so its identity
    # anchor is the ONE business document the user selected: that document is
    # checked against the identifier on the form, and the cancelled cheque
    # against the bank details on the form. The verdict is written BEFORE
    # determine_review_priority so the routing branch can read it.
    if is_business_job(job):
        selected_doc_type = resolve_business_doc_type(payload, job["documents"])
        uploaded_business_info = _business_uploaded_ocr(payload)
        uploaded_business_info["selected_business_doc"] = job.get("selected_business_doc") or uploaded_business_info.get("selected_business_doc") or selected_doc_type
        try:
            if settings.ml_mode == "mock":
                # Mock mode never fetches an image: the hook is the seam a test
                # uses to supply synthetic OCR text. It returns {} by default,
                # which keeps the verdict at manual review.
                business_ocr = run_business_document_ocr(
                    job["documents"],
                    None,
                    selected_doc_type,
                    None,
                    mock_business_document_text(job["documents"]),
                )
            else:
                business_ocr = run_business_document_ocr(
                    job["documents"], cache, selected_doc_type, loader
                )
        except Exception as exc:  # noqa: BLE001 - must never fail the job
            logger.warning(
                "business_docs_ocr_exception error=%s", type(exc).__name__
            )
            business_ocr = _business_ocr_failure("business_docs_ocr_exception")

        try:
            business_comparison = compare_business_documents(
                uploaded_business_info, business_ocr
            )
        except Exception as exc:  # noqa: BLE001 - must never fail the job
            logger.warning(
                "business_docs_comparison_exception error=%s", type(exc).__name__
            )
            business_comparison = {
                "decision": "manual",
                "score": 0,
                "reason": "mismatch",
                "signals": ["business_docs_comparison_error"],
            }

        # The nested object carries the evidence an operator needs to see WHY a
        # record was held back plus the decision/score shape the Go router
        # decodes; the flat keys mirror it, and "business_decision" is the Phase
        # G canonical alias all three writers of that field agree on.
        face_result["business_docs"] = {
            "extracted_biz_num": business_ocr.get("extracted_biz_num"),
            "primary_doc_type": business_ocr.get("primary_doc_type"),
            "cheque_ifsc": business_ocr.get("cheque_ifsc"),
            "cheque_acc_no": business_ocr.get("cheque_acc_no"),
            "error": business_ocr.get("error"),
            "decision": business_comparison["decision"],
            "score": business_comparison["score"],
            "reason": business_comparison["reason"],
            "signals": business_comparison.get("signals") or [],
        }
        face_result["business_decision"] = business_comparison["decision"]
        face_result["business_docs_decision"] = business_comparison["decision"]
        face_result["business_docs_score"] = business_comparison["score"]
        face_result["business_docs_match"] = (
            business_comparison["decision"] == "approve"
        )
        logger.info(
            "kyc.business_docs decision=%s score=%s reason=%s signals=%s "
            "doc_type=%s uploaded_gstin=%s uploaded_pan=%s "
            "uploaded_registration=%s uploaded_ifsc=%s uploaded_account=%s",
            business_comparison["decision"],
            business_comparison["score"],
            business_comparison["reason"],
            ",".join(str(signal) for signal in business_comparison.get("signals") or []),
            business_ocr.get("primary_doc_type"),
            bool(uploaded_business_info.get("gstin")),
            bool(uploaded_business_info.get("pan_number")),
            bool(uploaded_business_info.get("registration_number")),
            bool(uploaded_business_info.get("ifsc_code")),
            bool(uploaded_business_info.get("account_number")),
        )

    # Privacy invariant: the full 12-digit Aadhaar number is used only for the
    # in-memory comparison above and is never persisted (or handed to the
    # validation/logging stages). Strip every internal carrier before the
    # payloads leave this function.
    ocr_result.pop("_internal_aadhaar_number", None)
    document_in_selfie = face_result.get("document_in_selfie")
    if isinstance(document_in_selfie, dict):
        for side_key in ("selfie_ocr", "uploaded_ocr"):
            side = document_in_selfie.get(side_key)
            if isinstance(side, dict):
                side.pop("_internal_aadhaar_number", None)

    _log_document_validations(job_id, ocr_result)

    # Phase 2.5: observational QR metadata only. determine_review_priority keeps
    # its Phase 1 signature and never sees this value.
    doc_validation = run_document_validation(
        job, ocr_result, primary_bytes=primary_bytes
    )
    ocr_result["document_validation"] = doc_validation

    # Phase 3: (effective, raw). The effective priority keeps the legacy
    # review-priority column in sync; the raw decision is persisted separately
    # in ai_recommended_priority for the Go router to re-gate.
    effective_priority, raw_priority = determine_review_priority(
        ocr_result, face_result, liveness_result
    )

    processing_time_ms = max(int((time.perf_counter() - started) * 1000), 0)
    saved = save_results(
        verification_id=verification_id,
        ocr_result=ocr_result,
        face_result=face_result,
        liveness_result=liveness_result,
        review_priority=effective_priority,
        ai_recommended_priority=raw_priority,
        processing_time_ms=processing_time_ms,
        ml_vendor=ml_vendor,
        ml_model_version=ml_model_version,
    )
    if not saved:
        # Rowcount 0: duplicate completion, stale-escalated, or already routed.
        # Logged inside save_results; here we ack and move on (no retry —
        # retrying a completed/escalated record would fight the Go router).
        logger.info(
            "kyc.job_not_persisted job_id=%s verification_id=%s reason=save_results_noop",
            job_id,
            verification_id,
        )

    result = {
        "job_id": job_id,
        "verification_id": verification_id,
        "user_id": job["user_id"],
        "ml_status": "ml_completed",
        "saved": saved,
        "review_priority": effective_priority,
        "ai_recommended_priority": raw_priority,
        "processing_time_ms": processing_time_ms,
        "ml_vendor": ml_vendor,
        "ml_model_version": ml_model_version,
        "scores": {
            "ocr_confidence": ocr_result["ocr_confidence"],
            "face_similarity": face_result["face_similarity"],
            "liveness_score": liveness_result["liveness_score"],
            "is_live": liveness_result["is_live"],
        },
    }

    logger.info(
        "Completed KYC job %s (verification_id=%s): priority=%s, face=%.4f, "
        "liveness=%.4f, ocr=%.4f",
        job_id,
        verification_id,
        effective_priority,
        face_result["face_similarity"],
        liveness_result["liveness_score"],
        ocr_result["ocr_confidence"],
    )
    return result


def _extract_verification_id(payload: Any) -> str | None:
    """Best-effort extraction for error reporting, without logging PII."""
    try:
        return parse_job_payload(payload)["verification_id"]
    except Exception:  # noqa: BLE001 - diagnostic helper
        return None


def _unpack_stream_message(message: Any) -> tuple[str, dict[str, Any]]:
    """Normalize a Redis stream entry across redis-py response shapes.

    redis-py 5.x returns entries as ``(message_id, fields_dict)`` tuples inside
    the XREADGROUP response, while other versions/parsers expose a dict with an
    explicit ``id`` key. Support both so the worker is not tied to one minor
    redis-py release.
    """
    if isinstance(message, dict):
        message_id = str(message.get("id", ""))
        # A dict response may either be the fields map itself (with `id`
        # alongside the fields) or a wrapper containing `fields`.
        fields = message.get("fields")
        if isinstance(fields, dict):
            return message_id, fields
        return message_id, message

    if isinstance(message, (tuple, list)) and len(message) >= 2:
        message_id = str(message[0])
        fields = message[1]
        if isinstance(fields, dict):
            return message_id, fields
        if isinstance(fields, (tuple, list)):
            # Defensive fallback for flat [key, value, key, value] payloads.
            return message_id, dict(zip(fields[::2], fields[1::2]))

    return "", {}


def _restore_app_log_levels() -> None:
    """Keep application logs visible after third-party model warm-up.

    PaddleOCR/PaddlePaddle can reset the root logger to WARNING while loading
    models, which silently hides every INFO line the worker emits afterwards.
    Pin the application loggers to LOG_LEVEL instead of re-enabling all
    third-party loggers.
    """
    level_name = os.getenv("LOG_LEVEL", "INFO").upper()
    level = getattr(logging, level_name, logging.INFO)
    logging.getLogger("kyc.worker").setLevel(level)
    logging.getLogger("kyc.main").setLevel(level)


# ---------------------------------------------------------------------------
# Redis Stream consumer
# ---------------------------------------------------------------------------


class KYCWorker:
    """Background Redis Stream consumer.

    The worker is intentionally a plain thread (not an asyncio task) because
    ``redis-py`` and ``psycopg2`` are blocking clients. FastAPI starts/stops it
    through the lifespan hook in :mod:`main`.
    """

    def __init__(self, model_loader: ModelLoader | None = None) -> None:
        self._stop_event = threading.Event()
        self._thread: threading.Thread | None = None
        self._redis_client: redis.Redis | None = None
        self._group_ready = False
        self.model_loader = model_loader or get_model_loader()

        default_name = f"{socket.gethostname()}-{os.getpid()}"
        self.consumer_name = settings.kyc_consumer_name or default_name

    # -- lifecycle ---------------------------------------------------------

    @property
    def redis_client(self) -> redis.Redis:
        if self._redis_client is None:
            self._redis_client = build_redis_client()
        return self._redis_client

    def start(self) -> None:
        """Load models once, then start the consumer thread (idempotent)."""
        if self._thread is not None and self._thread.is_alive():
            logger.warning("KYC worker is already running")
            return

        try:
            self.model_loader.load_models()
        except Exception:  # noqa: BLE001 - keep health endpoint available
            logger.exception("Model loading failed; worker starting in degraded mode")
        finally:
            # Some OCR/model libraries reset the root logger during warm-up.
            _restore_app_log_levels()

        self._stop_event.clear()
        self._thread = threading.Thread(
            target=self._run,
            name="kyc-ml-worker",
            daemon=True,
        )
        self._thread.start()
        logger.info(
            "KYC worker started (stream=%s, group=%s, consumer=%s, mode=%s)",
            settings.kyc_stream,
            settings.kyc_consumer_group,
            self.consumer_name,
            settings.ml_mode,
        )

    def stop(self, timeout: float = 10.0) -> None:
        """Ask the consumer thread to stop and close the Redis client."""
        self._stop_event.set()
        thread = self._thread
        if thread is not None:
            thread.join(timeout=timeout)
            if thread.is_alive():
                logger.warning(
                    "KYC worker thread did not stop within %.1fs", timeout
                )
            else:
                self._thread = None

        if self._redis_client is not None:
            try:
                self._redis_client.close()
            except Exception as exc:  # noqa: BLE001 - shutdown best effort
                logger.debug("Failed to close Redis client: %s", exc)
            finally:
                self._redis_client = None
                self._group_ready = False

    def ping_redis(self) -> bool:
        try:
            return bool(self.redis_client.ping())
        except Exception as exc:  # noqa: BLE001 - health check must never raise
            logger.debug("Redis health check failed: %s", exc)
            return False

    # -- consumer loop -----------------------------------------------------

    def _run(self) -> None:
        logger.info("KYC worker loop started")
        delay = settings.worker_reconnect_delay_seconds
        pending_replayed = False

        while not self._stop_event.is_set():
            try:
                self._ensure_group()
                if not pending_replayed:
                    self._drain_pending()
                    pending_replayed = True
                self._read_batch()
            except ResponseError as exc:
                if "NOGROUP" in str(exc).upper():
                    # The stream/group was deleted or Redis was recreated.
                    self._group_ready = False
                    logger.warning("KYC consumer group is missing; recreating it")
                else:
                    logger.exception("Redis error in KYC worker loop")
                self._stop_event.wait(delay)
            except Exception:  # noqa: BLE001 - keep the worker alive
                self._group_ready = False
                logger.exception(
                    "KYC worker loop error; retrying in %ss", delay
                )
                self._stop_event.wait(delay)

        logger.info("KYC worker loop stopped")

    def _ensure_group(self) -> None:
        if self._group_ready:
            return

        try:
            self.redis_client.xgroup_create(
                name=settings.kyc_stream,
                groupname=settings.kyc_consumer_group,
                id="0",
                mkstream=True,
            )
            logger.info(
                "Created Redis consumer group %s on %s",
                settings.kyc_consumer_group,
                settings.kyc_stream,
            )
        except ResponseError as exc:
            if "BUSYGROUP" not in str(exc).upper():
                raise
            logger.debug("Redis consumer group already exists")

        self._group_ready = True

    def _drain_pending(self) -> None:
        """Replay entries left pending by a previous worker run.

        ``XREADGROUP`` with ID ``0`` returns the current consumer's Pending
        Entries List (unacknowledged messages) instead of new messages. This
        runs once at startup, before the normal ``>`` read loop.
        """
        logger.info(
            "Checking pending Redis stream messages (stream=%s, group=%s, consumer=%s, id=0)",
            settings.kyc_stream,
            settings.kyc_consumer_group,
            self.consumer_name,
        )
        seen: set[str] = set()

        while not self._stop_event.is_set():
            streams = self.redis_client.xreadgroup(
                groupname=settings.kyc_consumer_group,
                consumername=self.consumer_name,
                streams={settings.kyc_stream: "0"},
                count=settings.kyc_batch_size,
            )
            entries = [
                message
                for _stream_name, messages in (streams or [])
                for message in messages
            ]
            logger.info("Received %d raw pending entries from Redis", len(entries))
            if not entries:
                return

            new_entries = [
                message
                for message in entries
                if _unpack_stream_message(message)[0] not in seen
            ]
            if not new_entries:
                logger.warning(
                    "Pending Redis messages did not advance; stopping replay "
                    "to avoid an infinite loop"
                )
                return

            for message in new_entries:
                message_id, _fields = _unpack_stream_message(message)
                seen.add(message_id)
                self._handle_message(message)

            # A short batch means the PEL is drained (or only failed entries
            # that we already saw remain pending for a future retry).
            if len(entries) < settings.kyc_batch_size:
                return

    def _read_batch(self) -> None:
        streams = self.redis_client.xreadgroup(
            groupname=settings.kyc_consumer_group,
            consumername=self.consumer_name,
            streams={settings.kyc_stream: ">"},
            count=settings.kyc_batch_size,
            block=settings.kyc_block_ms,
        )
        entries = [
            message
            for _stream_name, messages in (streams or [])
            for message in messages
        ]
        logger.info("Received %d raw entries from Redis", len(entries))
        if not entries:
            return

        for message in entries:
            self._handle_message(message)

    def _handle_message(self, message: Any) -> None:
        message_id, fields = _unpack_stream_message(message)
        if not message_id or not fields:
            logger.error("Ignoring malformed Redis stream entry: %r", message)
            return

        payload = fields.get(PAYLOAD_FIELD)

        # Small compatibility fallback if a future producer uses `data`.
        if payload is None:
            payload = fields.get("data")

        if payload is None:
            logger.error(
                "KYC stream message %s has no %r field; leaving it pending",
                message_id,
                PAYLOAD_FIELD,
            )
            return

        try:
            result = process_job(payload, self.model_loader)
        except Exception as exc:  # noqa: BLE001 - one bad job must not kill worker
            verification_id = _extract_verification_id(payload)
            logger.exception(
                "KYC job %s failed (verification_id=%s): %s",
                message_id,
                verification_id or "unknown",
                exc,
            )
            if verification_id:
                try:
                    mark_failed(verification_id, type(exc).__name__)
                except Exception:  # noqa: BLE001 - best effort only
                    logger.exception(
                        "Failed to mark verification %s as ml_failed",
                        verification_id,
                    )

            # Deliberately no XACK here: the entry stays in the consumer
            # group's Pending Entries List for a future retry/DLQ (XAUTOCLAIM)
            # implementation.
            return

        self.redis_client.xack(
            settings.kyc_stream, settings.kyc_consumer_group, message_id
        )
        # Skipped (noop) results carry no priority — handle both shapes.
        logger.info(
            "KYC job %s acknowledged (verification_id=%s, priority=%s)",
            message_id,
            result["verification_id"],
            result.get("review_priority", "<skipped>"),
        )


# ---------------------------------------------------------------------------
# Standalone entry point
# ---------------------------------------------------------------------------


def main() -> None:
    logging.basicConfig(
        level=os.getenv("LOG_LEVEL", "INFO").upper(),
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    worker = KYCWorker()
    worker.start()

    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        logger.info("Shutdown requested")
    finally:
        worker.stop()


if __name__ == "__main__":
    main()
