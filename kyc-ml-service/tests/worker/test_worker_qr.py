"""Phase 2.5 integration tests for :func:`worker.run_document_validation`.

The QR pipeline is stubbed with ``unittest.mock.patch``: no real UIDAI
certificate, no real Aadhaar card, and no QR decoder are required.
"""

from __future__ import annotations

import dataclasses
import json
import time
from contextlib import ExitStack, contextmanager
from unittest.mock import patch

import pytest

from app.qr.aadhaar_qr import QRResult, QRVerificationError

from app import worker

JOB = {
    "job_id": "job-qr-1",
    "verification_id": "vid-qr-1",
    "user_id": "user-qr-1",
    "documents": [
        {"type": "selfie", "url": "https://cdn.example/selfie.jpg"},
        {"type": "govt_id", "url": "https://cdn.example/govt-id.jpg"},
    ],
}

# Opaque sentinel: the worker must not inspect certificate internals.
_CERT = object()


@pytest.fixture(autouse=True)
def _clear_cert_cache():
    """``_load_trusted_certs`` is process-cached; reset it between tests."""
    worker._load_trusted_certs.cache_clear()
    yield
    worker._load_trusted_certs.cache_clear()


def _settings(monkeypatch, **overrides):
    values = {
        "ml_mode": "real",
        "qr_decode_enabled": True,
        "qr_verification_required": False,
        "qr_max_decode_seconds": 0.2,
    }
    values.update(overrides)
    monkeypatch.setattr(
        worker, "settings", dataclasses.replace(worker.settings, **values)
    )


def _qr_result(
    *,
    reference_last4: str | None = "1234",
    photo_format: str = "jpeg",
    photo: bytes = b"\xff\xd8\xffsynthetic-photo",
) -> QRResult:
    return QRResult(
        signature_valid=True,
        matched_cert_name="synthetic-uidai.cer",
        matched_cert_fingerprint="0" * 64,
        qr_version="V2",
        fields={"reference_id": "1234/56789/12345"},
        photo_bytes=photo,
        photo_format=photo_format,
        qr_reference_last4=reference_last4,
    )


@contextmanager
def _stub_pipeline(
    *,
    decode=None,
    verify=None,
    certs=_CERT,
    image: bytes = b"fake-image-bytes",
):
    """Stub every QR pipeline collaborator.

    ``decode``/``verify`` accept a callable (side effect). ``certs=None`` makes
    the pinned-cert loader raise ``ValueError``.
    """
    with ExitStack() as stack:
        stack.enter_context(
            patch.object(worker, "_fetch_document_bytes", return_value=image)
        )
        if certs is None:
            stack.enter_context(
                patch.object(
                    worker, "load_pinned_certs", side_effect=ValueError("no certs")
                )
            )
        else:
            stack.enter_context(
                patch.object(worker, "load_pinned_certs", return_value=[certs])
            )
        if decode is None:
            stack.enter_context(
                patch.object(
                    worker,
                    "decode_qr_text",
                    return_value=("1234567890", "zxing:original"),
                )
            )
        else:
            stack.enter_context(patch.object(worker, "decode_qr_text", side_effect=decode))
        if verify is None:
            stack.enter_context(
                patch.object(worker, "verify_aadhaar_qr", return_value=_qr_result())
            )
        else:
            stack.enter_context(
                patch.object(worker, "verify_aadhaar_qr", side_effect=verify)
            )
        yield


def test_qr_disabled_returns_not_attempted(monkeypatch):
    _settings(monkeypatch, qr_decode_enabled=False)

    result = worker.run_document_validation(JOB, {"aadhaar_last4": "1234"})

    assert result["overall"] == "not_attempted"
    assert result["qr_attempted"] is False
    assert result["qr_error"] is None


def test_mock_mode_returns_not_attempted(monkeypatch):
    _settings(monkeypatch, ml_mode="mock")

    result = worker.run_document_validation(JOB, {"aadhaar_last4": "1234"})

    assert result["overall"] == "not_attempted"
    assert result["qr_attempted"] is False


def test_no_document_returns_failed(monkeypatch):
    _settings(monkeypatch)

    result = worker.run_document_validation({**JOB, "documents": []}, {})

    assert result["overall"] == "failed"
    assert result["qr_error"] == "no_document"
    assert result["qr_attempted"] is True
    assert result["qr_decoded"] is False


def test_no_certs_returns_failed(monkeypatch):
    _settings(monkeypatch)

    with _stub_pipeline(certs=None):
        result = worker.run_document_validation(JOB, {})

    assert result["overall"] == "failed"
    assert result["qr_error"] == "no_trusted_certs"
    assert result["qr_decoded"] is False


def test_valid_qr_returns_verified(monkeypatch):
    _settings(monkeypatch)

    with _stub_pipeline():
        result = worker.run_document_validation(JOB, {"aadhaar_last4": "1234"})

    assert result["overall"] == "verified"
    assert result["qr_attempted"] is True
    assert result["qr_decoded"] is True
    assert result["qr_signature_verified"] is True
    assert result["qr_error"] is None
    assert result["matched_cert_name"] == "synthetic-uidai.cer"
    assert result["photo_format"] == "jpeg"
    assert result["qr_photo_extracted"] is True
    assert result["qr_reference_last4"] == "1234"
    assert result["qr_reference_matches_aadhaar"] is True
    assert isinstance(result["attempt_duration_ms"], int)
    assert result["attempt_duration_ms"] >= 0


def test_signature_invalid_returns_unverified(monkeypatch):
    _settings(monkeypatch)

    def _invalid(_text, _certs):
        raise QRVerificationError("qr_signature_invalid")

    with _stub_pipeline(verify=_invalid):
        result = worker.run_document_validation(JOB, {})

    assert result["overall"] == "unverified"
    assert result["qr_decoded"] is True
    assert result["qr_signature_verified"] is False
    assert result["qr_error"] == "qr_signature_invalid"
    assert result["matched_cert_name"] is None


def test_decode_error_returns_failed(monkeypatch):
    _settings(monkeypatch)

    def _not_decodable(_image):
        raise QRVerificationError("qr_not_decodable")

    with _stub_pipeline(decode=_not_decodable):
        result = worker.run_document_validation(JOB, {})

    assert result["overall"] == "failed"
    assert result["qr_error"] == "qr_not_decodable"
    assert result["qr_decoded"] is False


def test_timeout_returns_failed(monkeypatch):
    _settings(monkeypatch, qr_max_decode_seconds=0.05)

    def _slow_decode(_image):
        time.sleep(0.5)
        return "1234567890", "zxing:original"

    with _stub_pipeline(decode=_slow_decode):
        result = worker.run_document_validation(JOB, {})

    assert result["overall"] == "failed"
    assert result["qr_error"] == "timeout"
    assert result["qr_decoded"] is False


def test_reference_matches_aadhaar_true(monkeypatch):
    _settings(monkeypatch)

    with _stub_pipeline():
        result = worker.run_document_validation(JOB, {"aadhaar_last4": "1234"})

    assert result["qr_reference_matches_aadhaar"] is True


def test_reference_mismatch_false(monkeypatch):
    _settings(monkeypatch)

    with _stub_pipeline():
        result = worker.run_document_validation(JOB, {"aadhaar_last4": "9999"})

    assert result["qr_reference_matches_aadhaar"] is False


def test_reference_missing_returns_none(monkeypatch):
    _settings(monkeypatch)

    with _stub_pipeline():
        result = worker.run_document_validation(JOB, {})

    assert result["qr_reference_matches_aadhaar"] is None


def test_result_is_json_serializable(monkeypatch):
    _settings(monkeypatch)

    with _stub_pipeline():
        result = worker.run_document_validation(JOB, {"aadhaar_last4": "1234"})

    # Raises TypeError if any value (e.g. photo_bytes) is not serializable.
    json.dumps(result)


def test_unexpected_exception_returns_failed(monkeypatch):
    _settings(monkeypatch)

    def _boom(_image):
        raise RuntimeError("internal detail that must not leak")

    with _stub_pipeline(decode=_boom):
        result = worker.run_document_validation(JOB, {})

    assert result["overall"] == "failed"
    assert result["qr_error"] == "unexpected_exception"


# ---------------------------------------------------------------------------
# Phase 4.5: single-download reuse
# ---------------------------------------------------------------------------


def test_run_document_validation_uses_provided_bytes(monkeypatch):
    _settings(monkeypatch)

    with _stub_pipeline():
        with patch.object(
            worker,
            "_fetch_document_bytes",
            side_effect=AssertionError("must not fetch when bytes are provided"),
        ):
            result = worker.run_document_validation(
                JOB, {"aadhaar_last4": "1234"}, primary_bytes=b"provided-bytes"
            )

    assert result["overall"] == "verified"
    assert result["qr_decoded"] is True


def test_run_document_validation_falls_back_without_bytes(monkeypatch):
    _settings(monkeypatch)

    with _stub_pipeline():
        result = worker.run_document_validation(JOB, {"aadhaar_last4": "1234"})

    assert result["overall"] == "verified"
    assert result["qr_decoded"] is True


def test_fetch_primary_document_bytes_returns_govt_doc(monkeypatch):
    documents = [
        {"type": "aadhaar", "url": "https://cdn.example/aadhaar.jpg"},
        {"type": "pan_card", "url": "https://cdn.example/pan.jpg"},
    ]

    with patch.object(
        worker, "_fetch_document_bytes", return_value=b"govt-bytes"
    ) as fetch:
        result = worker._fetch_primary_document_bytes(documents)

    assert result == b"govt-bytes"
    fetch.assert_called_once_with("https://cdn.example/aadhaar.jpg")


def test_fetch_primary_document_bytes_returns_none_when_empty():
    with patch.object(worker, "_fetch_document_bytes") as fetch:
        result = worker._fetch_primary_document_bytes([])

    assert result is None
    fetch.assert_not_called()
