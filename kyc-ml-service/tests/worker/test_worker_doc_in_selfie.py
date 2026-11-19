"""Phase A integration tests: document-in-selfie verification in process_job.

Covers the ``document_in_selfie_decision/score/match`` keys the Go router gates
auto-approval on, plus never-raise behaviour of ``run_selfie_document_ocr``.
"""

from __future__ import annotations

import dataclasses
import importlib.util
import json
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

        class _StubRedis:  # pragma: no cover - only when redis-py is absent
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

PAYLOAD = {
    "job_id": "kyc-job-doc-in-selfie",
    "verification_id": "3f2b6c1e-1111-4a2b-9c3d-1234567890ab",
    "user_id": "8a7b6c5d-2222-4e3f-8a9b-0987654321fe",
    "type": "individual",
    "documents": [
        {"type": "selfie", "url": "https://cdn.example/selfie.jpg"},
        {"type": "govt_id", "url": "https://cdn.example/govt-id.jpg"},
    ],
}

# Verhoeff-valid fixtures (see tests/validators/test_verhoeff.py).
AADHAAR = "234567890124"
OTHER_AADHAAR = "999999999999"

UPLOADED_OCR = {
    "engine": "mock-paddleocr",
    "mock": True,
    "text": "NAME: PATEL FUJEL\nDOB: 16/05/2003",
    "name": "Patel Fujel",
    "dob": "16/05/2003",
    "confidence": 0.95,
    "ocr_confidence": 0.95,
    "aadhaar_valid": True,
    "aadhaar_source": "ocr_text",
    "aadhaar_last4": AADHAAR[-4:],
    # In-memory-only full number (Phase A.5); process_job must strip it.
    "_internal_aadhaar_number": AADHAAR,
}

SELFIE_OK = {
    "error": None,
    "aadhaar_number": AADHAAR,
    "_internal_aadhaar_number": AADHAAR,
    "name": "Patel Fujel",
    "dob": "16/05/2003",
    "confidence": 0.94,
    "text_redacted": "GOVERNMENT OF INDIA\nPATEL FUJEL\nXXXX XXXX 0124",
}

SELFIE_MISMATCH = {
    **SELFIE_OK,
    "aadhaar_number": OTHER_AADHAAR,
    "_internal_aadhaar_number": OTHER_AADHAAR,
}

SELFIE_ERROR = {
    "error": "selfie_no_text_detected",
    "aadhaar_number": None,
    "name": None,
    "dob": None,
    "confidence": 0.0,
    "text_redacted": "",
}


class _StubLoader:
    """Loader double: mock mode only needs the document-in-selfie OCR call."""

    def __init__(self, selfie_ocr):
        self.selfie_ocr = selfie_ocr
        self.calls = 0

    def run_selfie_document_ocr(self, documents, cache=None):
        self.calls += 1
        return dict(self.selfie_ocr)


class _RaisingLoader:
    def run_selfie_document_ocr(self, documents, cache=None):
        raise RuntimeError("boom")


def _run(
    monkeypatch,
    *,
    selfie_ocr=SELFIE_OK,
    enabled=True,
    uploaded=None,
    selfie_raises=False,
):
    """Run the mock pipeline with the document-in-selfie stage stubbed."""
    monkeypatch.setattr(
        worker,
        "settings",
        dataclasses.replace(
            worker.settings, ml_mode="mock", doc_in_selfie_enabled=enabled
        ),
    )
    monkeypatch.setattr(worker, "mark_processing", lambda verification_id: True)
    monkeypatch.setattr(
        worker, "mock_ocr", lambda documents: dict(uploaded or UPLOADED_OCR)
    )

    captured: dict[str, object] = {}

    def fake_save_results(**kwargs):
        captured.update(kwargs)
        return True

    monkeypatch.setattr(worker, "save_results", fake_save_results)

    loader = _RaisingLoader() if selfie_raises else _StubLoader(selfie_ocr)
    result = worker.process_job(json.dumps(PAYLOAD), loader)
    return result, captured, loader


# ---------------------------------------------------------------------------
# process_job integration
# ---------------------------------------------------------------------------


def test_disabled_returns_manual(monkeypatch):
    _, captured, loader = _run(monkeypatch, enabled=False)

    face = captured["face_result"]
    assert face["document_in_selfie"] == {"attempted": False}
    assert face["document_in_selfie_decision"] == "manual"
    assert face["document_in_selfie_score"] == 0
    assert face["document_in_selfie_match"] is False
    # The stage never even looks at the selfie when it is switched off.
    assert loader.calls == 0


def test_selfie_ocr_error_routes_manual(monkeypatch):
    _, captured, _ = _run(monkeypatch, selfie_ocr=SELFIE_ERROR)

    face = captured["face_result"]
    assert face["document_in_selfie"]["attempted"] is True
    assert face["document_in_selfie"]["error"] == "selfie_no_text_detected"
    assert face["document_in_selfie_decision"] == "manual"
    assert face["document_in_selfie_score"] == 0
    assert face["document_in_selfie_match"] is False


def test_selfie_ocr_exception_routes_manual(monkeypatch):
    _, captured, _ = _run(monkeypatch, selfie_raises=True)

    face = captured["face_result"]
    assert face["document_in_selfie"]["error"] == "selfie_ocr_exception"
    assert face["document_in_selfie_decision"] == "manual"
    assert face["document_in_selfie_score"] == 0
    assert face["document_in_selfie_match"] is False


def test_aadhaar_mismatch_rejects(monkeypatch):
    _, captured, _ = _run(monkeypatch, selfie_ocr=SELFIE_MISMATCH)

    face = captured["face_result"]
    assert face["document_in_selfie_decision"] == "reject"
    assert face["document_in_selfie_score"] == -100
    assert face["document_in_selfie_match"] is False
    assert face["document_in_selfie"]["comparison"]["signals"] == ["aadhaar_mismatch"]


def test_perfect_match_approves(monkeypatch):
    _, captured, _ = _run(monkeypatch)

    face = captured["face_result"]
    assert face["document_in_selfie"]["attempted"] is True
    assert face["document_in_selfie"]["error"] is None
    assert face["document_in_selfie_decision"] == "approve"
    assert face["document_in_selfie_score"] == 100
    assert face["document_in_selfie_match"] is True


def test_json_serializable(monkeypatch):
    _, captured, _ = _run(monkeypatch)

    face = captured["face_result"]
    round_tripped = json.loads(json.dumps(face, sort_keys=True))
    assert round_tripped["document_in_selfie_decision"] == "approve"
    assert round_tripped["document_in_selfie_score"] == 100
    assert round_tripped["document_in_selfie_match"] is True
    assert set(round_tripped["document_in_selfie"]) == {
        "attempted",
        "error",
        "selfie_ocr",
        "uploaded_ocr",
        "comparison",
    }


def test_raw_selfie_aadhaar_number_is_not_persisted(monkeypatch):
    _, captured, _ = _run(monkeypatch)

    stored = captured["face_result"]["document_in_selfie"]["selfie_ocr"]
    assert stored["aadhaar_number"] is None
    assert stored["aadhaar_last4"] == AADHAAR[-4:]
    assert AADHAAR not in json.dumps(captured["face_result"])


def test_full_aadhaar_never_persisted(monkeypatch):
    _, captured, _ = _run(monkeypatch)

    document_in_selfie = captured["face_result"]["document_in_selfie"]
    assert "_internal_aadhaar_number" not in captured["ocr_result"]
    assert "_internal_aadhaar_number" not in document_in_selfie["selfie_ocr"]
    assert "_internal_aadhaar_number" not in document_in_selfie["uploaded_ocr"]
    assert "_internal_aadhaar_number" not in json.dumps(
        captured["face_result"]
    )
    # Neither the raw number nor the internal carrier survives serialisation.
    assert AADHAAR not in json.dumps(captured["face_result"])
    assert AADHAAR not in json.dumps(captured["ocr_result"])
    json.dumps(captured["face_result"])
    json.dumps(captured["ocr_result"])


def test_redact_handles_newline_aadhaar():
    from app.worker import _redact_sensitive_ids

    redacted = _redact_sensitive_ids("2003\n2345 6789 0124")
    assert redacted == "2003\nXXXX XXXX 0124"
    assert "6789" not in redacted

    redacted_hyphen = _redact_sensitive_ids("2003\n2345-6789-0124")
    assert redacted_hyphen == "2003\nXXXX XXXX 0124"
    assert "6789" not in redacted_hyphen


def test_missing_selfie_document_does_not_break_the_job(monkeypatch):
    # No selfie document at all: the OCR method reports it and the pipeline
    # still completes with a manual decision.
    payload = {**PAYLOAD, "documents": [{"type": "govt_id", "url": "https://x/y.jpg"}]}
    monkeypatch.setattr(
        worker,
        "settings",
        dataclasses.replace(
            worker.settings, ml_mode="mock", doc_in_selfie_enabled=True
        ),
    )
    monkeypatch.setattr(worker, "mark_processing", lambda verification_id: True)
    monkeypatch.setattr(worker, "mock_ocr", lambda documents: dict(UPLOADED_OCR))
    captured: dict[str, object] = {}

    def fake_save_results(**kwargs):
        captured.update(kwargs)
        return True

    monkeypatch.setattr(worker, "save_results", fake_save_results)

    result = worker.process_job(json.dumps(payload), worker.ModelLoader())

    face = captured["face_result"]
    assert result["ml_status"] == "ml_completed"
    assert face["document_in_selfie"]["error"] == "selfie_not_found"
    assert face["document_in_selfie_decision"] == "manual"
    assert face["document_in_selfie_match"] is False


def test_determine_review_priority_unchanged(monkeypatch):
    """Phase A keys must not influence routing (they gate in the Go router)."""
    baseline, captured_baseline, _ = _run(monkeypatch, enabled=False)
    approved, captured_approved, _ = _run(monkeypatch)
    rejected, captured_rejected, _ = _run(monkeypatch, selfie_ocr=SELFIE_MISMATCH)

    assert (
        approved["review_priority"]
        == rejected["review_priority"]
        == baseline["review_priority"]
    )
    assert (
        approved["ai_recommended_priority"]
        == rejected["ai_recommended_priority"]
        == baseline["ai_recommended_priority"]
    )
    assert (
        captured_approved["review_priority"]
        == captured_rejected["review_priority"]
        == captured_baseline["review_priority"]
    )
    # Priorities come straight from the untouched Phase 1/3 decision function.
    assert captured_approved["review_priority"] in {
        worker.PRIORITY_AUTO_ELIGIBLE,
        worker.PRIORITY_MANUAL_REVIEW,
        worker.PRIORITY_AUTO_REJECT,
    }


# ---------------------------------------------------------------------------
# Real-mode path (shared image cache, loader.run_selfie_document_ocr)
# ---------------------------------------------------------------------------


class _RealModeLoader:
    """Real-mode loader double: only the stages process_job calls are stubbed."""

    def __init__(self, selfie_ocr):
        self.selfie_ocr = selfie_ocr
        self.core_ready = True
        self.errors: dict[str, str] = {}
        self.model_version = "test-models-v1"
        self.selfie_caches = []

    def load_models(self):
        return None

    def run_ocr(self, documents, cache, primary_bytes=None):
        return dict(UPLOADED_OCR)

    def run_liveness(self, documents, cache):
        return {
            "enabled": True,
            "engine": "stub-liveness",
            "liveness_score": 0.96,
            "is_live": True,
            "error": None,
        }

    def run_selfie_document_ocr(self, documents, cache):
        self.selfie_caches.append(cache)
        return dict(self.selfie_ocr)


def test_real_mode_path_writes_contract_fields(monkeypatch):
    loader = _RealModeLoader(SELFIE_OK)
    monkeypatch.setattr(
        worker,
        "settings",
        dataclasses.replace(
            worker.settings,
            ml_mode="real",
            # Face matching is disabled in production (Phase B); process_job
            # then uses the disabled-result helper instead of run_face_match.
            face_match_enabled=False,
            doc_in_selfie_enabled=True,
        ),
    )
    monkeypatch.setattr(worker, "mark_processing", lambda verification_id: True)
    monkeypatch.setattr(worker, "_fetch_primary_document_bytes", lambda docs: None)
    monkeypatch.setattr(
        worker,
        "run_document_validation",
        lambda job, ocr_result, primary_bytes=None: {"overall": "not_attempted"},
    )
    captured: dict[str, object] = {}

    def fake_save_results(**kwargs):
        captured.update(kwargs)
        return True

    monkeypatch.setattr(worker, "save_results", fake_save_results)

    result = worker.process_job(json.dumps(PAYLOAD), loader)

    face = captured["face_result"]
    assert result["ml_status"] == "ml_completed"
    assert face["document_in_selfie_decision"] == "approve"
    assert face["document_in_selfie_score"] == 100
    assert face["document_in_selfie_match"] is True
    # The real path reuses the job-wide image cache instead of fetching twice.
    assert len(loader.selfie_caches) == 1
    assert isinstance(loader.selfie_caches[0], worker.ImageCache)
    assert json.loads(json.dumps(face))["document_in_selfie_score"] == 100


# ---------------------------------------------------------------------------
# run_selfie_document_ocr: never raises
# ---------------------------------------------------------------------------


class _FakeOCR:
    """Minimal PaddleOCR 2.x shaped engine."""

    def __init__(self, lines):
        self.lines = lines

    def ocr(self, image, cls=False):
        return [[[None, (text, score)] for text, score in self.lines]]


class _FakeCache:
    def __init__(self, image=object(), error=None):
        self.image = image
        self.error = error
        self.urls = []

    def get(self, url):
        self.urls.append(url)
        if self.error is not None:
            raise self.error
        return self.image


def _loader_with(monkeypatch, lines, *, min_confidence=0.60):
    monkeypatch.setattr(
        worker,
        "settings",
        dataclasses.replace(
            worker.settings,
            ml_mode="mock",
            doc_in_selfie_min_confidence=min_confidence,
        ),
    )
    loader = worker.ModelLoader()
    loader.ocr_engine = _FakeOCR(lines)
    return loader


SELFIE_DOC = [{"type": "selfie", "url": "https://cdn.example/selfie.jpg"}]

SELFIE_LINES = [
    ("GOVERNMENT OF INDIA", 0.95),
    ("PATEL FUJEL", 0.95),
    ("2345 6789 0124", 0.95),
    ("DOB: 16/05/2003", 0.95),
]


def test_run_selfie_document_ocr_not_found(monkeypatch):
    loader = _loader_with(monkeypatch, SELFIE_LINES)

    result = loader.run_selfie_document_ocr(
        [{"type": "govt_id", "url": "https://cdn.example/govt-id.jpg"}]
    )

    assert result["error"] == "selfie_not_found"
    assert result["confidence"] == 0.0
    assert result["aadhaar_number"] is None


def test_run_selfie_document_ocr_requires_engine(monkeypatch):
    monkeypatch.setattr(
        worker,
        "settings",
        dataclasses.replace(worker.settings, ml_mode="mock"),
    )
    loader = worker.ModelLoader()
    loader.ocr_engine = None

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert result["error"] == "ocr_unavailable"


def test_run_selfie_document_ocr_fetch_failure(monkeypatch):
    loader = _loader_with(monkeypatch, SELFIE_LINES)
    cache = _FakeCache(error=worker.ImageFetchError("no network"))

    result = loader.run_selfie_document_ocr(SELFIE_DOC, cache)

    assert result["error"] == "selfie_fetch_failed"


def test_run_selfie_document_ocr_extracts_fields(monkeypatch):
    loader = _loader_with(monkeypatch, SELFIE_LINES)

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert result["error"] is None
    assert result["aadhaar_number"] == AADHAAR
    assert "PATEL" in (result["name"] or "").upper()
    assert result["dob"] == "16/05/2003"
    assert result["confidence"] == pytest.approx(0.95)
    assert "2345 6789 0124" not in result["text_redacted"]


def test_run_selfie_document_ocr_low_confidence_is_flagged(monkeypatch):
    loader = _loader_with(
        monkeypatch,
        [("PATEL FUJEL", 0.30), ("DOB: 16/05/2003", 0.30)],
        min_confidence=0.60,
    )

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert result["error"] == "selfie_ocr_low_confidence"
    # The evidence is still reported so a reviewer can look at it.
    assert result["dob"] == "16/05/2003"
    assert result["confidence"] == pytest.approx(0.30)


def test_run_selfie_document_ocr_no_text(monkeypatch):
    loader = _loader_with(monkeypatch, [("   ", 0.95)])

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert result["error"] == "selfie_no_text_detected"


def test_run_selfie_document_ocr_result_is_json_serializable(monkeypatch):
    loader = _loader_with(monkeypatch, SELFIE_LINES)

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert json.loads(json.dumps(result)) == result
    assert set(result) == {
        "error",
        "aadhaar_number",
        "_internal_aadhaar_number",
        "name",
        "dob",
        "confidence",
        "text_redacted",
        # Phase A.6 diagnostics (added, never replacing the keys above).
        "preprocess_strategy",
        "card_detected",
        "images_tried",
    }
    # The full number is carried in memory for the comparison only.
    assert result["_internal_aadhaar_number"] == AADHAAR
    assert result["aadhaar_number"] == AADHAAR


# ---------------------------------------------------------------------------
# Phase A.6: card-region preprocessing before selfie OCR
# ---------------------------------------------------------------------------


class _SequencedOCR:
    """PaddleOCR 2.x shaped engine returning a different read per call."""

    def __init__(self, payloads):
        self.payloads = list(payloads)
        self.calls = 0
        self.images = []

    def ocr(self, image, cls=False):
        self.images.append(image)
        index = min(self.calls, len(self.payloads) - 1)
        self.calls += 1
        return [[[None, (text, score)] for text, score in self.payloads[index]]]


class _RaisingOCR:
    """Fails on the first image, succeeds afterwards."""

    def __init__(self, lines):
        self.lines = lines
        self.calls = 0

    def ocr(self, image, cls=False):
        self.calls += 1
        if self.calls == 1:
            raise RuntimeError("engine exploded on the warped card")
        return [[[None, (text, score)] for text, score in self.lines]]


def _patch_preprocess(monkeypatch, *, strategy, card_detected, images):
    payload = {
        "images": images,
        "strategy": strategy,
        "card_detected": card_detected,
        "decode_error": None,
    }
    seen = []

    def _fake_preprocess(image):
        seen.append(image)
        return dict(payload)

    monkeypatch.setattr(worker, "preprocess_selfie_image", _fake_preprocess)
    return seen


CARD_LINES = [
    ("GOVERNMENT OF INDIA", 0.95),
    ("PATEL FUJEL", 0.95),
    ("2345 6789 0124", 0.95),
    ("DOB: 16/05/2003", 0.95),
]
NOISE_LINES = [("sibnltotnsmmsvo", 0.95), ("0 ao", 0.95)]


def test_selfie_ocr_uses_preprocess_strategy(monkeypatch):
    """The second preprocessed frame wins when the first has no Aadhaar."""
    loader = _loader_with(monkeypatch, SELFIE_LINES)
    engine = _SequencedOCR([[("no aadhaar here", 0.95)], CARD_LINES])
    loader.ocr_engine = engine
    _patch_preprocess(
        monkeypatch,
        strategy="card_extracted",
        card_detected=True,
        images=["warped-card", "enhanced-full"],
    )

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert result["error"] is None
    assert result["_internal_aadhaar_number"] == AADHAAR
    assert result["_internal_aadhaar_number"][-4:] == "0124"
    assert result["preprocess_strategy"] == "card_extracted"
    assert result["card_detected"] is True
    assert result["images_tried"] == 2
    # Both frames were OCRd, in the order the preprocessor returned them.
    assert engine.calls == 2
    assert engine.images == ["warped-card", "enhanced-full"]


def test_selfie_ocr_falls_back_to_full_when_no_card(monkeypatch):
    """Without a detected card the pipeline is identical to the baseline."""
    loader = _loader_with(monkeypatch, SELFIE_LINES)
    engine = _SequencedOCR([SELFIE_LINES])
    loader.ocr_engine = engine
    _patch_preprocess(
        monkeypatch,
        strategy="full_image",
        card_detected=False,
        images=["enhanced-full"],
    )

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert engine.calls == 1
    assert result["error"] is None
    assert result["aadhaar_number"] == AADHAAR
    assert result["preprocess_strategy"] == "full_image"
    assert result["card_detected"] is False
    assert result["images_tried"] == 1
    assert result["confidence"] == pytest.approx(0.95)
    assert result["dob"] == "16/05/2003"


def test_selfie_ocr_reports_unreadable_when_no_aadhaar_anywhere(monkeypatch):
    """No Aadhaar in any frame: nothing is claimed, evidence is kept.

    The worker keeps its historical error semantics here (``error`` stays
    ``None`` when the read is confident enough to trust); mapping the missing
    number to the ``selfie_aadhaar_unreadable`` reason is
    :mod:`app.validators.document_match`'s job and is unchanged.
    """
    loader = _loader_with(monkeypatch, SELFIE_LINES)
    engine = _SequencedOCR([NOISE_LINES, NOISE_LINES])
    loader.ocr_engine = engine
    _patch_preprocess(
        monkeypatch,
        strategy="card_extracted",
        card_detected=True,
        images=["warped-card", "enhanced-full"],
    )

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert engine.calls == 2
    assert result["aadhaar_number"] is None
    assert result["_internal_aadhaar_number"] is None
    assert result["error"] is None
    assert result["confidence"] == pytest.approx(0.95)
    assert result["preprocess_strategy"] == "card_extracted"


def test_selfie_ocr_low_confidence_when_no_aadhaar_anywhere(monkeypatch):
    """The pre-existing low-confidence gate still fires on noisy reads."""
    loader = _loader_with(
        monkeypatch,
        [("PATEL FUJEL", 0.30), ("DOB: 16/05/2003", 0.30)],
        min_confidence=0.60,
    )
    loader.ocr_engine = _SequencedOCR(
        [[("PATEL FUJEL", 0.20)], [("DOB: 16/05/2003", 0.30)]]
    )
    _patch_preprocess(
        monkeypatch,
        strategy="top_half_crop",
        card_detected=False,
        images=["crop", "full"],
    )

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert result["error"] == "selfie_ocr_low_confidence"
    assert result["_internal_aadhaar_number"] is None
    assert result["confidence"] == pytest.approx(0.30)
    assert result["dob"] == "16/05/2003"
    assert result["preprocess_strategy"] == "top_half_crop"


def test_selfie_ocr_preserves_return_contract(monkeypatch):
    """Every pre-A.6 key keeps its name, type and meaning."""
    loader = _loader_with(monkeypatch, SELFIE_LINES)
    loader.ocr_engine = _SequencedOCR([SELFIE_LINES])
    _patch_preprocess(
        monkeypatch,
        strategy="full_image",
        card_detected=False,
        images=["full"],
    )

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    for key in (
        "error",
        "aadhaar_number",
        "_internal_aadhaar_number",
        "name",
        "dob",
        "confidence",
        "text_redacted",
        "preprocess_strategy",
        "card_detected",
        "images_tried",
    ):
        assert key in result, key

    assert result["error"] is None
    assert result["_internal_aadhaar_number"] == AADHAAR
    # Historical behaviour: the loader reports the full number and process_job
    # reduces it to ``aadhaar_last4`` before persistence.
    assert result["aadhaar_number"] == AADHAAR
    assert isinstance(result["confidence"], float)
    assert isinstance(result["text_redacted"], str)
    assert isinstance(result["card_detected"], bool)
    assert isinstance(result["images_tried"], int)
    assert result["preprocess_strategy"] in {
        "card_extracted",
        "top_half_crop",
        "full_image",
        "decode_failed",
    }


def test_selfie_ocr_continues_after_a_failing_pass(monkeypatch):
    """A warped card the engine chokes on must not lose the full-frame read."""
    loader = _loader_with(monkeypatch, SELFIE_LINES)
    engine = _RaisingOCR(SELFIE_LINES)
    loader.ocr_engine = engine
    _patch_preprocess(
        monkeypatch,
        strategy="card_extracted",
        card_detected=True,
        images=["warped-card", "enhanced-full"],
    )

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert engine.calls == 2
    assert result["error"] is None
    assert result["_internal_aadhaar_number"] == AADHAAR


def test_selfie_ocr_decode_failed_when_no_images(monkeypatch):
    loader = _loader_with(monkeypatch, SELFIE_LINES)
    engine = _SequencedOCR([SELFIE_LINES])
    loader.ocr_engine = engine
    _patch_preprocess(
        monkeypatch,
        strategy="decode_failed",
        card_detected=False,
        images=[],
    )

    result = loader.run_selfie_document_ocr(SELFIE_DOC, _FakeCache())

    assert engine.calls == 0
    assert result["error"] == "selfie_ocr_decode_failed"
    assert result["images_tried"] == 0
    assert result["confidence"] == 0.0
    assert result["_internal_aadhaar_number"] is None
