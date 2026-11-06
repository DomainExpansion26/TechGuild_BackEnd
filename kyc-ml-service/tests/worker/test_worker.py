"""Unit tests for the KYC ML worker.

Run from this directory::

    pip install -r requirements.txt
    pytest test_worker.py -v

The tests focus on pure decision/parsing logic plus the graceful-failure paths.
They do not require Redis, PostgreSQL, ONNX models, or PaddleOCR. Missing
runtime-only third-party packages are replaced with minimal stubs at import
time, so the suite also works in a lightweight CI image.
"""

from __future__ import annotations

import dataclasses
import importlib.util
import json
import sys
import types
from pathlib import Path

import pytest


# ---------------------------------------------------------------------------
# Import bootstrap: keep the worker importable without the heavy runtime deps.
# ---------------------------------------------------------------------------


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


# ---------------------------------------------------------------------------
# Shared fixtures / helpers
# ---------------------------------------------------------------------------


@pytest.fixture(autouse=True)
def stable_settings(monkeypatch):
    """Pin thresholds so tests don't depend on a developer's .env file."""
    test_settings = dataclasses.replace(
        worker.settings,
        ml_mode="mock",
        threshold_face_auto_pass=0.60,
        threshold_face_review=0.40,
        threshold_face_reject=0.35,
        shadow_mode=False,
        threshold_live_pass=0.90,
        threshold_live_reject=0.50,
        threshold_ocr_pass=0.80,
    )
    monkeypatch.setattr(worker, "settings", test_settings)
    return test_settings


def _priority_inputs(
    *,
    face: float | str | None = 0.90,
    live: float | str | None = 0.95,
    ocr: float | str | None = 0.90,
    ocr_text: str | None = "NAME: TEST USER",
):
    ocr_result: dict[str, object] = {}
    if ocr is not None:
        ocr_result["ocr_confidence"] = ocr
    if ocr_text is not None:
        ocr_result["text"] = ocr_text

    face_result: dict[str, object] = {}
    if face is not None:
        face_result["face_similarity"] = face

    liveness_result: dict[str, object] = {}
    if live is not None:
        liveness_result["liveness_score"] = live
        if isinstance(live, (int, float)):
            liveness_result["is_live"] = live >= 0.90

    return ocr_result, face_result, liveness_result


GOOD_PAYLOAD = {
    "job_id": "kyc-job-abc123",
    "verification_id": "3f2b6c1e-1111-4a2b-9c3d-1234567890ab",
    "user_id": "8a7b6c5d-2222-4e3f-8a9b-0987654321fe",
    "type": "individual",
    "documents": [
        {"type": "selfie", "url": "https://cdn.example/selfie.jpg"},
        {"type": "govt_id", "url": "https://cdn.example/govt-id.jpg?token=secret"},
    ],
}


# ---------------------------------------------------------------------------
# Priority routing
# ---------------------------------------------------------------------------


@pytest.mark.parametrize(
    ("case", "face", "live", "ocr", "ocr_text", "expected"),
    [
        (
            "all scores high",
            0.90,
            0.95,
            0.90,
            "NAME: TEST USER",
            worker.PRIORITY_AUTO_ELIGIBLE,
        ),
        (
            "exact pass thresholds",
            0.60,
            0.90,
            0.80,
            "NAME: TEST USER",
            worker.PRIORITY_AUTO_ELIGIBLE,
        ),
        (
            "face below reject threshold",
            0.34,
            0.95,
            0.90,
            "NAME: TEST USER",
            worker.PRIORITY_AUTO_REJECT,
        ),
        (
            "face exactly at reject threshold is borderline",
            0.35,
            0.95,
            0.90,
            "NAME: TEST USER",
            worker.PRIORITY_MANUAL_REVIEW,
        ),
        (
            "liveness below reject threshold",
            0.90,
            0.49,
            0.90,
            "NAME: TEST USER",
            worker.PRIORITY_AUTO_REJECT,
        ),
        (
            "liveness exactly at reject threshold is borderline",
            0.90,
            0.50,
            0.90,
            "NAME: TEST USER",
            worker.PRIORITY_MANUAL_REVIEW,
        ),
        (
            "borderline face with good liveness",
            0.50,
            0.95,
            0.90,
            "NAME: TEST USER",
            worker.PRIORITY_MANUAL_REVIEW,
        ),
        (
            "OCR confidence below pass",
            0.90,
            0.95,
            0.79,
            "NAME: TEST USER",
            worker.PRIORITY_MANUAL_REVIEW,
        ),
        (
            "missing OCR text",
            0.90,
            0.95,
            0.99,
            "",
            worker.PRIORITY_MANUAL_REVIEW,
        ),
        (
            "missing OCR text key",
            0.90,
            0.95,
            0.99,
            None,
            worker.PRIORITY_MANUAL_REVIEW,
        ),
        (
            "missing OCR confidence",
            0.90,
            0.95,
            None,
            "NAME: TEST USER",
            worker.PRIORITY_MANUAL_REVIEW,
        ),
        (
            "missing face score",
            None,
            0.95,
            0.90,
            "NAME: TEST USER",
            worker.PRIORITY_AUTO_REJECT,
        ),
        (
            "missing liveness score",
            0.90,
            None,
            0.90,
            "NAME: TEST USER",
            worker.PRIORITY_AUTO_REJECT,
        ),
    ],
)
def test_determine_review_priority(case, face, live, ocr, ocr_text, expected):
    ocr_result, face_result, liveness_result = _priority_inputs(
        face=face, live=live, ocr=ocr, ocr_text=ocr_text
    )

    effective, raw = worker.determine_review_priority(
        ocr_result, face_result, liveness_result
    )
    # The fixture pins shadow_mode=False, so the effective decision and the
    # raw AI recommendation must both equal the expected raw policy outcome.
    assert effective == raw == expected, (
        f"{case}: got ({effective!r}, {raw!r}), want {expected!r}"
    )


@pytest.mark.parametrize("value", ["nan", "inf", "-inf", float("nan"), float("inf")])
def test_non_finite_scores_are_treated_as_missing(value):
    assert worker._as_float(value) == 0.0


def test_priority_accepts_numeric_strings():
    effective, raw = worker.determine_review_priority(
        {"ocr_confidence": "0.90", "text": "NAME: TEST USER"},
        {"face_similarity": "0.90"},
        {"liveness_score": "0.95"},
    )
    assert (effective, raw) == (
        worker.PRIORITY_AUTO_ELIGIBLE,
        worker.PRIORITY_AUTO_ELIGIBLE,
    )


def test_ocr_error_forces_manual_review_even_with_high_confidence():
    effective, raw = worker.determine_review_priority(
        {
            "ocr_confidence": 0.99,
            "text": "NAME: TEST USER",
            "error": "ocr_failed",
        },
        {"face_similarity": 0.90},
        {"liveness_score": 0.95},
    )
    assert (effective, raw) == (
        worker.PRIORITY_MANUAL_REVIEW,
        worker.PRIORITY_MANUAL_REVIEW,
    )


def test_hard_fraud_signal_wins_over_missing_ocr_text():
    effective, raw = worker.determine_review_priority(
        {"ocr_confidence": 0.0, "text": "", "error": "no_text_detected"},
        {"face_similarity": 0.10},
        {"liveness_score": 0.95},
    )
    assert (effective, raw) == (
        worker.PRIORITY_AUTO_REJECT,
        worker.PRIORITY_AUTO_REJECT,
    )


def test_missing_selfie_does_not_auto_reject_business_kyc():
    # Business KYC has no selfie. A missing face/liveness stage is missing
    # evidence, not a fraud signal.
    effective, raw = worker.determine_review_priority(
        {"ocr_confidence": 0.95, "text": "GSTIN: 27ABCDE1234F1Z5"},
        {"face_similarity": 0.0, "error": "missing_document"},
        {"liveness_score": 0.0, "is_live": False, "error": "missing_selfie"},
    )
    assert (effective, raw) == (
        worker.PRIORITY_MANUAL_REVIEW,
        worker.PRIORITY_MANUAL_REVIEW,
    )


def test_image_fetch_failure_does_not_auto_reject():
    effective, raw = worker.determine_review_priority(
        {
            "ocr_confidence": 0.0,
            "text": "",
            "error": "image_fetch_failed",
        },
        {"face_similarity": 0.0, "error": "image_fetch_failed"},
        {
            "liveness_score": 0.0,
            "is_live": False,
            "error": "image_fetch_failed",
        },
    )
    assert (effective, raw) == (
        worker.PRIORITY_MANUAL_REVIEW,
        worker.PRIORITY_MANUAL_REVIEW,
    )


def test_failed_face_check_still_auto_rejects_on_low_liveness():
    # A missing face stage does not mask a successful liveness inference that
    # is below the fraud threshold.
    effective, raw = worker.determine_review_priority(
        {"ocr_confidence": 0.95, "text": "NAME: TEST USER"},
        {"face_similarity": 0.0, "error": "no_face_in_govt_id"},
        {"liveness_score": 0.10},
    )
    assert (effective, raw) == (
        worker.PRIORITY_AUTO_REJECT,
        worker.PRIORITY_AUTO_REJECT,
    )


# ---------------------------------------------------------------------------
# Redis payload parsing
# ---------------------------------------------------------------------------


def test_parse_job_payload_from_json_string():
    job = worker.parse_job_payload(json.dumps(GOOD_PAYLOAD))

    assert job["job_id"] == "kyc-job-abc123"
    assert job["verification_id"] == "3f2b6c1e-1111-4a2b-9c3d-1234567890ab"
    assert job["user_id"] == "8a7b6c5d-2222-4e3f-8a9b-0987654321fe"
    assert job["type"] == "individual"
    assert [doc["url"] for doc in job["documents"]] == [
        "https://cdn.example/selfie.jpg",
        "https://cdn.example/govt-id.jpg?token=secret",
    ]
    assert [doc["type"] for doc in job["documents"]] == ["selfie", "govt_id"]


def test_parse_job_payload_from_bytes():
    job = worker.parse_job_payload(json.dumps(GOOD_PAYLOAD).encode("utf-8"))
    assert job["job_id"] == GOOD_PAYLOAD["job_id"]
    assert job["verification_id"] == GOOD_PAYLOAD["verification_id"]


def test_parse_job_payload_from_dict_normalizes_fields():
    job = worker.parse_job_payload(
        {
            "job_id": 123,
            "verification_id": "  vid-1  ",
            "user_id": None,
            "documents": [
                {"type": "SELFIE", "url": "https://cdn.example/s.jpg"},
                "ignored-non-object",
                {"type": None, "url": None},
            ],
        }
    )

    assert job["job_id"] == "123"
    assert job["verification_id"] == "vid-1"
    assert job["user_id"] == ""
    assert job["type"] == ""
    assert job["documents"] == [
        {"type": "selfie", "url": "https://cdn.example/s.jpg"},
        {"type": "", "url": ""},
    ]


@pytest.mark.parametrize(
    "payload",
    [
        "not-json",
        json.dumps([]),
        json.dumps({"documents": []}),  # missing verification_id
        json.dumps({"verification_id": "v", "documents": "not-a-list"}),
        123,
        b"\xff\xfe",  # invalid UTF-8
    ],
)
def test_parse_job_payload_rejects_invalid_input(payload):
    with pytest.raises(worker.JobPayloadError):
        worker.parse_job_payload(payload)


def test_unpack_stream_message_supports_redis_py_tuple_shape():
    message_id, fields = worker._unpack_stream_message(
        ("1-0", {"payload": "{}"})
    )
    assert message_id == "1-0"
    assert fields == {"payload": "{}"}


def test_unpack_stream_message_supports_dict_wrapper_shape():
    message_id, fields = worker._unpack_stream_message(
        {"id": "2-0", "fields": {"payload": "{}"}}
    )
    assert message_id == "2-0"
    assert fields == {"payload": "{}"}


def test_unpack_stream_message_supports_flat_field_list():
    message_id, fields = worker._unpack_stream_message(
        ("3-0", ["payload", "{}", "other", "value"])
    )
    assert message_id == "3-0"
    assert fields == {"payload": "{}", "other": "value"}


def test_unpack_stream_message_rejects_garbage():
    assert worker._unpack_stream_message("garbage") == ("", {})


# ---------------------------------------------------------------------------
# Aadhaar OCR text post-processing
# ---------------------------------------------------------------------------

AADHAAR_OCR_TEXT = (
    "Issue Date15/01/2014\n"
    "H H y\n"
    "XXXX XXXX 5235\n"
    "Government of India\n"
    "y?/Male\n"
    "PatelFujel Gulan\n"
    "458\n"
    "/DOB\n"
    "2d\n"
    "16/05/2xxx"
)


def test_extract_name_and_dob_from_real_aadhaar_ocr_text():
    assert worker._extract_name(AADHAAR_OCR_TEXT) == "Patel Fujel Gulan"
    assert worker._extract_dob(AADHAAR_OCR_TEXT) == "16/05/2xxx"


def test_extract_dob_does_not_confuse_issue_date_with_dob():
    # The issue date appears first in the text but must not be returned.
    assert worker._extract_dob(AADHAAR_OCR_TEXT) != "15/01/2014"


def test_extract_name_ignores_mixed_hindi_noise():
    text = "Government of India\nपुरुष / Male\nRavi Kumar\nDOB\n01/02/1985"
    assert worker._extract_name(text) == "Ravi Kumar"


def test_extract_name_prefers_explicit_label():
    text = "Government of India\nRavi Kumar\nName: Alice M. D'Souza\nDOB"
    assert worker._extract_name(text) == "Alice M. D'Souza"


def test_extract_name_rejects_mask_and_gender_lines():
    assert worker._extract_name("Government of India\nXXXX XXXX 5235\n") == ""
    assert worker._extract_name("Government of India\ny?/Male\n") == ""


def test_extract_dob_supports_iso_and_same_line_formats():
    assert worker._extract_dob("NAME: MOCK USER\nDOB: 1990-01-01") == "1990-01-01"
    assert worker._extract_dob("Date of Birth: 03/04/1990") == "03/04/1990"


def test_extract_dob_uses_masked_year_without_label():
    assert worker._extract_dob("Holder\n16/05/2xxx\n") == "16/05/2xxx"


class _FakeRedis:
    def __init__(self) -> None:
        self.acked: list[tuple[str, str, str]] = []

    def xack(self, stream: str, group: str, message_id: str) -> int:
        self.acked.append((stream, group, message_id))
        return 1


def test_handle_message_extracts_payload_and_acks(monkeypatch):
    fake_redis = _FakeRedis()
    consumer = worker.KYCWorker(model_loader=worker.ModelLoader())
    consumer._redis_client = fake_redis

    captured: dict[str, object] = {}

    def fake_process_job(payload, model_loader):
        captured["payload"] = payload
        job = worker.parse_job_payload(payload)
        return {
            "verification_id": job["verification_id"],
            "review_priority": worker.PRIORITY_AUTO_ELIGIBLE,
        }

    monkeypatch.setattr(worker, "process_job", fake_process_job)

    raw_payload = json.dumps(GOOD_PAYLOAD)
    consumer._handle_message(("1-0", {"payload": raw_payload}))

    assert captured["payload"] == raw_payload
    assert fake_redis.acked == [
        (
            worker.settings.kyc_stream,
            worker.settings.kyc_consumer_group,
            "1-0",
        )
    ]


def test_handle_message_accepts_data_field_fallback(monkeypatch):
    fake_redis = _FakeRedis()
    consumer = worker.KYCWorker(model_loader=worker.ModelLoader())
    consumer._redis_client = fake_redis

    captured: dict[str, object] = {}

    def fake_process_job(payload, model_loader):
        captured["payload"] = payload
        return {
            "verification_id": GOOD_PAYLOAD["verification_id"],
            "review_priority": worker.PRIORITY_AUTO_ELIGIBLE,
        }

    monkeypatch.setattr(worker, "process_job", fake_process_job)

    raw_payload = json.dumps(GOOD_PAYLOAD)
    consumer._handle_message(("2-0", {"data": raw_payload}))

    assert captured["payload"] == raw_payload
    assert fake_redis.acked == [
        (
            worker.settings.kyc_stream,
            worker.settings.kyc_consumer_group,
            "2-0",
        )
    ]


def test_handle_message_without_payload_is_not_processed_or_acked(monkeypatch):
    fake_redis = _FakeRedis()
    consumer = worker.KYCWorker(model_loader=worker.ModelLoader())
    consumer._redis_client = fake_redis

    called = False

    def fake_process_job(payload, model_loader):
        nonlocal called
        called = True
        return {}

    monkeypatch.setattr(worker, "process_job", fake_process_job)

    consumer._handle_message(("3-0", {"unexpected": "value"}))

    assert called is False
    assert fake_redis.acked == []


# ---------------------------------------------------------------------------
# Graceful failure paths
# ---------------------------------------------------------------------------


def _install_corrupt_image_stack(monkeypatch) -> None:
    """Make ``cv2.imdecode`` return None, i.e. an undecodable image."""
    fake_numpy = types.ModuleType("numpy")
    fake_numpy.uint8 = "uint8"
    fake_numpy.frombuffer = lambda data, dtype=None: data

    fake_cv2 = types.ModuleType("cv2")
    fake_cv2.IMREAD_COLOR = 1
    fake_cv2.imdecode = lambda data, flags: None

    monkeypatch.setitem(sys.modules, "numpy", fake_numpy)
    monkeypatch.setitem(sys.modules, "cv2", fake_cv2)


def test_fetch_image_corrupted_file_raises_clean_error(monkeypatch, tmp_path):
    _install_corrupt_image_stack(monkeypatch)

    corrupted = Path(tmp_path) / "corrupted.jpg"
    corrupted.write_bytes(b"\xff\xd8this-is-not-a-real-jpeg")

    with pytest.raises(worker.ImageFetchError, match="could not decode image"):
        worker.fetch_image(corrupted.as_uri())


def test_run_liveness_corrupted_selfie_returns_zero_not_crash(
    monkeypatch, tmp_path
):
    _install_corrupt_image_stack(monkeypatch)

    corrupted = Path(tmp_path) / "selfie.jpg"
    corrupted.write_bytes(b"broken-image-bytes")

    loader = worker.ModelLoader()
    loader.liveness_session = object()
    loader.face_detector = object()

    result = loader.run_liveness(
        [{"type": "selfie", "url": corrupted.as_uri()}], worker.ImageCache()
    )

    assert result["liveness_score"] == 0.0
    assert result["is_live"] is False
    assert result["error"] == "image_fetch_failed"
    assert result["selfie_present"] is True
    assert result["face_detected"] is False


def test_run_face_match_corrupted_images_returns_zero(monkeypatch, tmp_path):
    _install_corrupt_image_stack(monkeypatch)

    corrupted = Path(tmp_path) / "face.jpg"
    corrupted.write_bytes(b"broken-image-bytes")

    loader = worker.ModelLoader()
    loader.face_detector = object()
    loader.face_recognizer = object()

    result = loader.run_face_match(
        [
            {"type": "govt_id", "url": corrupted.as_uri()},
            {"type": "selfie", "url": corrupted.as_uri()},
        ],
        worker.ImageCache(),
    )

    assert result["face_similarity"] == 0.0
    assert result["error"] == "image_fetch_failed"


def test_run_ocr_corrupted_document_returns_zero_confidence(monkeypatch, tmp_path):
    _install_corrupt_image_stack(monkeypatch)

    corrupted = Path(tmp_path) / "document.jpg"
    corrupted.write_bytes(b"broken-image-bytes")

    loader = worker.ModelLoader()
    loader.ocr_engine = object()

    result = loader.run_ocr(
        [{"type": "govt_id", "url": corrupted.as_uri()}], worker.ImageCache()
    )

    assert result["ocr_confidence"] == 0.0
    assert result["confidence"] == 0.0
    assert result["error"] == "image_fetch_failed"


def test_corrupted_documents_route_to_manual_review(monkeypatch, tmp_path):
    _install_corrupt_image_stack(monkeypatch)

    corrupted = Path(tmp_path) / "document.jpg"
    corrupted.write_bytes(b"broken-image-bytes")
    documents = [
        {"type": "govt_id", "url": corrupted.as_uri()},
        {"type": "selfie", "url": corrupted.as_uri()},
    ]

    loader = worker.ModelLoader()
    loader.ocr_engine = object()
    loader.face_detector = object()
    loader.face_recognizer = object()
    loader.liveness_session = object()

    ocr = loader.run_ocr(documents, worker.ImageCache())
    face = loader.run_face_match(documents, worker.ImageCache())
    liveness = loader.run_liveness(documents, worker.ImageCache())

    effective, raw = worker.determine_review_priority(ocr, face, liveness)
    assert (effective, raw) == (
        worker.PRIORITY_MANUAL_REVIEW,
        worker.PRIORITY_MANUAL_REVIEW,
    )


def test_model_loader_missing_models_reports_clean_errors(monkeypatch, tmp_path):
    real_settings = dataclasses.replace(
        worker.settings,
        ml_mode="real",
        model_dir=str(tmp_path),
        ocr_backend="none",
    )
    monkeypatch.setattr(worker, "settings", real_settings)

    loader = worker.ModelLoader(model_dir=tmp_path)
    loader.load_models()

    assert loader.core_ready is False
    assert loader.ready is False
    assert loader.status()["state"] == "unavailable"
    assert loader.errors, "missing models should be recorded in loader.errors"


def test_process_job_real_mode_fails_cleanly_when_models_missing(
    monkeypatch, tmp_path
):
    real_settings = dataclasses.replace(
        worker.settings,
        ml_mode="real",
        model_dir=str(tmp_path),
        ocr_backend="none",
    )
    monkeypatch.setattr(worker, "settings", real_settings)

    loader = worker.ModelLoader(model_dir=tmp_path)

    with pytest.raises(worker.JobProcessingError, match="AI models are not loaded"):
        worker.process_job(json.dumps(GOOD_PAYLOAD), loader)


def test_inference_fallbacks_when_models_unavailable():
    loader = worker.ModelLoader()
    documents = [
        {"type": "govt_id", "url": "file:///does-not-exist-id.jpg"},
        {"type": "selfie", "url": "file:///does-not-exist-selfie.jpg"},
    ]

    ocr = loader.run_ocr(documents, worker.ImageCache())
    assert ocr["ocr_confidence"] == 0.0
    assert ocr["error"] == "ocr_unavailable"

    face = loader.run_face_match(documents, worker.ImageCache())
    assert face["face_similarity"] == 0.0
    assert face["error"] == "model_unavailable"

    liveness = loader.run_liveness(documents, worker.ImageCache())
    assert liveness["liveness_score"] == 0.0
    assert liveness["is_live"] is False
    assert liveness["error"] == "liveness_model_unavailable"


# ---------------------------------------------------------------------------
# Full mock pipeline (no DB / no real models)
# ---------------------------------------------------------------------------


def test_process_job_mock_mode_runs_pipeline_and_persists(monkeypatch):
    calls: dict[str, object] = {}

    def fake_mark_processing(verification_id: str) -> bool:
        calls["processing"] = verification_id
        return True

    def fake_save_results(**kwargs):
        calls["save"] = kwargs
        return True

    monkeypatch.setattr(worker, "mark_processing", fake_mark_processing)
    monkeypatch.setattr(worker, "save_results", fake_save_results)

    result = worker.process_job(json.dumps(GOOD_PAYLOAD), worker.ModelLoader())

    assert result["job_id"] == GOOD_PAYLOAD["job_id"]
    assert result["verification_id"] == GOOD_PAYLOAD["verification_id"]
    assert result["ml_status"] == "ml_completed"
    assert result["saved"] is True
    assert result["review_priority"] == worker.PRIORITY_AUTO_ELIGIBLE
    assert result["ai_recommended_priority"] == worker.PRIORITY_AUTO_ELIGIBLE
    assert result["scores"]["is_live"] is True

    assert calls["processing"] == GOOD_PAYLOAD["verification_id"]
    saved = calls["save"]
    assert saved["verification_id"] == GOOD_PAYLOAD["verification_id"]
    assert saved["review_priority"] == worker.PRIORITY_AUTO_ELIGIBLE
    assert saved["ai_recommended_priority"] == worker.PRIORITY_AUTO_ELIGIBLE

    # Phase 5: mock mode keeps the multi-reference result shape. The legacy
    # govt_id fixture scores as a single reference with no Aadhaar similarity,
    # so the router's evidence gate guarantees manual review after shadow-off.
    face_result = saved["face_result"]
    assert face_result["error"] is None
    assert face_result["winning_reference"] == "govt_id"
    assert face_result["reference_count"] == 1
    assert face_result["reference_scores"] == {
        "govt_id": face_result["face_similarity"]
    }
    assert face_result["aadhaar_similarity"] is None
    assert face_result["second_reference_divergence"] is False


# ---------------------------------------------------------------------------
# Phase 3: shadow tuple + DB guards (routing handoff to the Go router)
# ---------------------------------------------------------------------------


class _RecordingCursor:
    """Minimal psycopg2 cursor double: records SQL + params, reports rowcount."""

    def __init__(self, rowcount: int, captured: list) -> None:
        self.rowcount = rowcount
        self._captured = captured

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        return False

    def execute(self, sql, params=None):
        self._captured.append((sql, params))


class _RecordingConnection:
    def __init__(self, rowcount: int, captured: list) -> None:
        self._rowcount = rowcount
        self._captured = captured

    def cursor(self):
        return _RecordingCursor(self._rowcount, self._captured)


class _RecordingDB:
    """``worker.db_connection`` double yielding ``_RecordingConnection``."""

    def __init__(self, rowcount: int) -> None:
        self.captured: list = []
        self.conn = _RecordingConnection(rowcount, self.captured)

    def __enter__(self):
        return self.conn

    def __exit__(self, *exc):
        return False


def test_shadow_returns_raw_and_effective(monkeypatch):
    """Shadow on: raw threshold decision is preserved, only the effective
    write-down to manual_review affects routing."""
    shadow_settings = dataclasses.replace(worker.settings, shadow_mode=True)
    monkeypatch.setattr(worker, "settings", shadow_settings)

    ocr_result, face_result, liveness_result = _priority_inputs()
    effective, raw = worker.determine_review_priority(
        ocr_result, face_result, liveness_result
    )

    assert effective == worker.PRIORITY_MANUAL_REVIEW
    assert raw == worker.PRIORITY_AUTO_ELIGIBLE


def test_save_results_writes_pending_routing_state(monkeypatch):
    """The completion UPDATE must hand the record to the Go router and guard
    against overwriting an already-routed/escalated row."""
    fake = _RecordingDB(rowcount=1)
    monkeypatch.setattr(worker, "db_connection", lambda: fake)

    saved = worker.save_results(
        verification_id=GOOD_PAYLOAD["verification_id"],
        ocr_result={
            "ocr_confidence": 0.95,
            "text": "NAME: TEST USER",
            "document_validation": {"overall": "verified"},
        },
        face_result={"face_similarity": 0.90},
        liveness_result={"is_live": True, "liveness_score": 0.95},
        review_priority=worker.PRIORITY_MANUAL_REVIEW,
        ai_recommended_priority=worker.PRIORITY_AUTO_ELIGIBLE,
        processing_time_ms=12,
    )

    assert saved is True
    sql, params = fake.captured[0]
    assert "routing_state = 'pending'" in sql
    assert "ai_recommended_priority = " in sql
    assert "review_priority = " in sql
    assert "policy_version" in sql
    # Stale-escalation race guards (ChatGPT P0 #1): a late/duplicate
    # completion can never overwrite escalation or a routed record.
    assert "ml_status <> 'ml_completed'" in sql
    assert "system_escalated_at IS NULL" in sql
    assert "routing_state = 'pending'" in sql
    assert params["ai_recommended_priority"] == worker.PRIORITY_AUTO_ELIGIBLE
    assert params["review_priority"] == worker.PRIORITY_MANUAL_REVIEW
    assert params["policy_version"] == worker.settings.policy_version


def test_save_results_noop_returns_false(monkeypatch):
    """rowcount=0 (duplicate/escalated/routed) -> False, no exception."""
    fake = _RecordingDB(rowcount=0)
    monkeypatch.setattr(worker, "db_connection", lambda: fake)

    saved = worker.save_results(
        verification_id=GOOD_PAYLOAD["verification_id"],
        ocr_result={"ocr_confidence": 0.95, "text": "NAME: TEST USER"},
        face_result={"face_similarity": 0.90},
        liveness_result={"is_live": True, "liveness_score": 0.95},
        review_priority=worker.PRIORITY_AUTO_ELIGIBLE,
        ai_recommended_priority=worker.PRIORITY_AUTO_ELIGIBLE,
        processing_time_ms=12,
    )

    assert saved is False
    assert len(fake.captured) == 1


def test_mark_processing_noop_when_completed(monkeypatch):
    """A redelivered message for an ml_completed record returns rowcount=0 and
    mark_processing reports False: no state change, caller acks the message."""
    fake = _RecordingDB(rowcount=0)
    monkeypatch.setattr(worker, "db_connection", lambda: fake)

    moved = worker.mark_processing(GOOD_PAYLOAD["verification_id"])

    assert moved is False
    sql, params = fake.captured[0]
    assert "ml_status IN ('queued', 'processing')" in sql
    # Escalated records are terminal at the DB level (ChatGPT P0 #1).
    assert "system_escalated_at IS NULL" in sql
    assert params == (GOOD_PAYLOAD["verification_id"],)


# ---------------------------------------------------------------------------
# Phase 4.5: single-download reuse in process_job
# ---------------------------------------------------------------------------


def test_process_job_fetches_primary_document_once(monkeypatch):
    real_settings = dataclasses.replace(worker.settings, ml_mode="real")
    monkeypatch.setattr(worker, "settings", real_settings)
    monkeypatch.setattr(worker, "mark_processing", lambda verification_id: True)
    monkeypatch.setattr(worker, "save_results", lambda **kwargs: True)
    monkeypatch.setattr(
        worker,
        "determine_review_priority",
        lambda *args, **kwargs: ("auto_eligible", "auto_eligible"),
    )

    fetch_calls: list[object] = []

    def fake_fetch_primary(documents):
        fetch_calls.append(documents)
        return b"primary-bytes"

    monkeypatch.setattr(worker, "_fetch_primary_document_bytes", fake_fetch_primary)

    ocr_primary: list[object] = []

    def fake_run_ocr(documents, cache, primary_bytes=None):
        ocr_primary.append(primary_bytes)
        return {"ocr_confidence": 0.0}

    monkeypatch.setattr(
        worker,
        "run_document_validation",
        lambda job, ocr_result, primary_bytes=None: {"overall": "not_attempted"},
    )

    class _FakeLoader:
        core_ready = True
        model_version = "test-version"

        def load_models(self):
            return None

        run_ocr = staticmethod(fake_run_ocr)
        run_face_match = staticmethod(lambda documents, cache: {"face_similarity": 0.0})
        run_liveness = staticmethod(
            lambda documents, cache: {"liveness_score": 0.0, "is_live": False}
        )

    result = worker.process_job(json.dumps(GOOD_PAYLOAD), _FakeLoader())

    assert len(fetch_calls) == 1
    assert ocr_primary == [b"primary-bytes"]
    assert result["ml_status"] == "ml_completed"
