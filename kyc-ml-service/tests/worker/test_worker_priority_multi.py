"""Phase 5 unit tests: review-priority guardrails for multi-reference matching.

``determine_review_priority`` must cap the raw priority at manual_review when
``second_reference_divergence`` is set, while keeping every Phase 3 rule
(hard fraud reject, missing evidence, thresholds, shadow wrapper) intact.
"""

from __future__ import annotations

import dataclasses
import importlib.util
import sys
import types

import pytest


def _install_stub(name: str, **attrs):
    module = types.ModuleType(name)
    for key, value in attrs.items():
        setattr(module, key, value)
    sys.modules[name] = module
    return module


def _ensure_test_imports() -> None:
    if importlib.util.find_spec("dotenv") is None:
        _install_stub("dotenv", load_dotenv=lambda *args, **kwargs: False)

    if importlib.util.find_spec("redis") is None:
        redis_module = _install_stub("redis")

        class _StubRedis:  # pragma: no cover
            def __init__(self, *args, **kwargs):
                pass

        redis_module.Redis = _StubRedis
        exceptions_module = _install_stub("redis.exceptions")

        class ResponseError(Exception):
            pass

        exceptions_module.ResponseError = ResponseError
        redis_module.exceptions = exceptions_module

    if importlib.util.find_spec("psycopg2") is None:

        def _no_database(*args, **kwargs):
            raise RuntimeError("psycopg2 is stubbed for unit tests")

        _install_stub("psycopg2", connect=_no_database)


_ensure_test_imports()

from app import worker  # noqa: E402  (must come after the dependency stubs)


@pytest.fixture(autouse=True)
def stable_settings(monkeypatch):
    test_settings = dataclasses.replace(
        worker.settings,
        ml_mode="mock",
        threshold_face_auto_pass=0.60,
        threshold_face_review=0.40,
        threshold_face_reject=0.35,
        threshold_live_pass=0.90,
        threshold_live_reject=0.50,
        threshold_ocr_pass=0.80,
        second_reference_divergence_threshold=0.25,
        shadow_mode=False,
    )
    monkeypatch.setattr(worker, "settings", test_settings)
    return test_settings


def _inputs(
    *,
    face: dict,
    live: float = 0.95,
    ocr: float = 0.90,
    ocr_text: str = "NAME: TEST USER",
):
    ocr_result = {"ocr_confidence": ocr, "text": ocr_text}
    liveness_result = {"liveness_score": live, "is_live": live >= 0.90}
    return ocr_result, face, liveness_result


def _face_result(
    *,
    max_score: float,
    divergence: bool,
    winning_reference: str = "pan_card",
    aadhaar_similarity: float | None = 0.30,
) -> dict:
    result = {
        "face_similarity": max_score,
        "cosine_similarity": max_score,
        "winning_reference": winning_reference,
        "reference_count": 2,
        "error": None,
    }
    if aadhaar_similarity is not None:
        result["aadhaar_similarity"] = aadhaar_similarity
    result["second_reference_divergence"] = divergence
    return result


def test_divergence_forces_manual():
    """PAN wins (0.70) with a weak Aadhaar (0.30): manual, never auto."""
    effective, raw = worker.determine_review_priority(
        *_inputs(face=_face_result(max_score=0.70, divergence=True))
    )

    assert (effective, raw) == (
        worker.PRIORITY_MANUAL_REVIEW,
        worker.PRIORITY_MANUAL_REVIEW,
    )


def test_divergence_does_not_mask_auto_reject():
    """The divergence cap must not rescue a hard fraud signal: face below the
    reject threshold still auto-rejects."""
    effective, raw = worker.determine_review_priority(
        *_inputs(
            face=_face_result(max_score=0.34, divergence=True),  # < reject 0.35
        )
    )

    assert raw == worker.PRIORITY_AUTO_REJECT
    assert effective == worker.PRIORITY_AUTO_REJECT


def test_no_divergence_high_score_auto():
    effective, raw = worker.determine_review_priority(
        *_inputs(face=_face_result(max_score=0.70, divergence=False))
    )

    assert (effective, raw) == (
        worker.PRIORITY_AUTO_ELIGIBLE,
        worker.PRIORITY_AUTO_ELIGIBLE,
    )


def test_shadow_still_forces_manual(monkeypatch):
    shadow_settings = dataclasses.replace(worker.settings, shadow_mode=True)
    monkeypatch.setattr(worker, "settings", shadow_settings)

    effective, raw = worker.determine_review_priority(
        *_inputs(face=_face_result(max_score=0.70, divergence=False))
    )

    assert effective == worker.PRIORITY_MANUAL_REVIEW
    assert raw == worker.PRIORITY_AUTO_ELIGIBLE


def test_divergence_key_absent_defaults_to_phase3_behaviour(monkeypatch):
    """Legacy Phase 1/3 face results carry no divergence key: unchanged path."""
    face = {"face_similarity": 0.70}
    effective, raw = worker.determine_review_priority(
        *_inputs(face=face)
    )

    assert (effective, raw) == (
        worker.PRIORITY_AUTO_ELIGIBLE,
        worker.PRIORITY_AUTO_ELIGIBLE,
    )


def test_divergence_log_recorded(monkeypatch, caplog):
    """The kyc.face_divergence log line names the winner and both scores."""
    with caplog.at_level("INFO", logger="kyc.worker"):
        worker.determine_review_priority(
            *_inputs(face=_face_result(max_score=0.70, divergence=True))
        )

    face_logs = [
        record for record in caplog.records if record.getMessage().startswith(
            "kyc.face_divergence"
        )
    ]
    assert face_logs, "expected a kyc.face_divergence log entry"
    message = face_logs[-1].getMessage()
    assert "pan_card" in message
    assert "max=0.700" in message
    assert "aadhaar=0.300" in message
