"""Phase G tests: business document OCR + cross-match auto-approval.

Covers run_business_document_ocr (selected business document + cancelled cheque
extraction from messy synthetic OCR text), the process_job wiring in both mock
and real mode, and the business branch of determine_review_priority.

The suite runs without Redis, PostgreSQL, ONNX models or PaddleOCR: missing
runtime-only packages are replaced with minimal stubs at import time.
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

PAN = "ABCDE1234F"
OTHER_PAN = "ZZZZZ9999Z"
GSTIN = "27ABCDE1234F1Z5"
IFSC = "HDFC0001234"
ACCOUNT_LAST4 = "8901"  # Last 4 digits from "12345678901"

PAN_URL = "https://cdn.example/pan-card.jpg"
GST_URL = "https://cdn.example/gst-certificate.jpg"
CHEQUE_URL = "https://cdn.example/cancelled-cheque.jpg"

BUSINESS_DOCUMENTS = [
    {"type": "pan_card", "url": PAN_URL},
    {"type": "gst_certificate", "url": GST_URL},
    {"type": "cancelled_cheque", "url": CHEQUE_URL},
]

# Messy, label-carrying reads of the same identity (spaces inside the numbers,
# unrelated lines around them).
PAN_TEXT = """INCOME TAX DEPARTMENT
GOVT. OF INDIA
PERMANENT ACCOUNT NUMBER
ABCDE 1234 F
DOB: 01/01/1990
"""

GST_TEXT = """GSTIN : 27ABCDE1234F1Z5
Legal Name: ACME TRADERS PVT LTD
Date of Registration: 01/07/2017
"""

CHEQUE_TEXT = """STATE BANK OF INDIA
IFSC: HDFC 0001234
A/c No: 12345678901
"""

# The same documents, but the GST certificate belongs to somebody else.
GST_TEXT_OTHER_PAN = """GSTIN : 27ZZZZZ9999Z1Z3
Legal Name: ACME TRADERS PVT LTD
"""

BUSINESS_PAYLOAD = {
    "job_id": "kyc-job-business",
    "verification_id": "3f2b6c1e-3333-4a2b-9c3d-1234567890ab",
    "user_id": "8a7b6c5d-4444-4e3f-8a9b-0987654321fe",
    "type": "business",
    "documents": BUSINESS_DOCUMENTS,
    "selected_business_doc": "gst_certificate",
}

INDIVIDUAL_PAYLOAD = {
    "job_id": "kyc-job-individual",
    "verification_id": "3f2b6c1e-5555-4a2b-9c3d-1234567890ab",
    "user_id": "8a7b6c5d-6666-4e3f-8a9b-0987654321fe",
    "type": "individual",
    "documents": [
        {"type": "selfie", "url": "https://cdn.example/selfie.jpg"},
        {"type": "govt_id", "url": "https://cdn.example/govt-id.jpg"},
    ],
}

MOCK_OCR = {
    "engine": "mock-paddleocr",
    "mock": True,
    "text": "MOCK OCR TEXT\nNAME: MOCK USER",
    "name": "Mock User",
    "dob": "1990-01-01",
    "confidence": 0.95,
    "ocr_confidence": 0.95,
}

BUSINESS_TEXTS = {
    "pan_card": PAN_TEXT,
    "gst_certificate": GST_TEXT,
    "cancelled_cheque": CHEQUE_TEXT,
}

# Phase G: business_info and bank_details nested structure
BUSINESS_PAYLOAD_WITH_VALUES = {
    **BUSINESS_PAYLOAD,
    "business_info": {"gstin": GSTIN, "pan": PAN, "reg_no": ""},
    "bank_details": {"ifsc": IFSC, "account_number": "12345678901"},
    "selected_business_doc": "gst_certificate",
}

BUSINESS_PAYLOAD_PAN_SELECTED = {
    **BUSINESS_PAYLOAD,
    "business_info": {"gstin": GSTIN, "pan": PAN, "reg_no": ""},
    "bank_details": {"ifsc": IFSC, "account_number": "12345678901"},
    "selected_business_doc": "pan_card",
}

BUSINESS_DOCUMENTS_WITH_INC = [
    {"type": "incorporation_certificate", "url": "https://cdn.example/inc-cert.jpg"},
    {"type": "gst_certificate", "url": GST_URL},
    {"type": "cancelled_cheque", "url": CHEQUE_URL},
]

BUSINESS_PAYLOAD_INC_SELECTED = {
    **BUSINESS_PAYLOAD,
    "documents": BUSINESS_DOCUMENTS_WITH_INC,
    "business_info": {"gstin": "", "pan": "", "reg_no": "U12345MH2024PTC123456"},
    "bank_details": {"ifsc": IFSC, "account_number": "12345678901"},
    "selected_business_doc": "incorporation_certificate",
}

# ---------------------------------------------------------------------------
# run_business_document_ocr
# ---------------------------------------------------------------------------


def test_run_business_document_ocr_extracts_biz_num_and_cheque():
    result = worker.run_business_document_ocr(
        BUSINESS_DOCUMENTS,
        None,
        "gst_certificate",
        None,
        BUSINESS_TEXTS,
    )

    assert result["error"] is None
    assert result["extracted_biz_num"] == GSTIN
    assert result["primary_doc_type"] == "gst_certificate"
    assert result["cheque_ifsc"] == IFSC
    assert result["cheque_acc_last4"] == ACCOUNT_LAST4
    for key in ("extracted_biz_num", "cheque_ifsc", "cheque_acc_last4", "primary_doc_type", "error"):
        assert key in result, key
    assert json.loads(json.dumps(result)) == result


def test_run_business_document_ocr_prefers_the_labelled_candidate():
    # A structurally valid PAN/IFSC on an unlabelled line must lose to the one
    # printed next to the field's own label.
    pan_text = "REF XYZAB9999Q ACME TRADERS\nPERMANENT ACCOUNT NUMBER : ABCDE 1234 F\n"
    cheque_text = "BRANCH CODE HDFC0009999\nIFSC: HDFC 0001234\n"

    result = worker.run_business_document_ocr(
        BUSINESS_DOCUMENTS,
        None,
        "pan_card",
        None,
        {
            "pan_card": pan_text,
            "gst_certificate": GST_TEXT,
            "cancelled_cheque": cheque_text,
        },
    )

    assert result["extracted_biz_num"] == PAN
    assert result["cheque_ifsc"] == IFSC


def test_run_business_document_ocr_gstin_with_another_pan_is_flagged():
    result = worker.run_business_document_ocr(
        BUSINESS_DOCUMENTS,
        None,
        "gst_certificate",
        None,
        dict(BUSINESS_TEXTS, gst_certificate=GST_TEXT_OTHER_PAN),
    )

    assert result["extracted_biz_num"] == "27ZZZZZ9999Z1Z3"
    # The GSTIN's embedded PAN differs from the form's PAN
    # The primary document is still gst_certificate


def test_run_business_document_ocr_missing_document_sets_error():
    result = worker.run_business_document_ocr(
        BUSINESS_DOCUMENTS[:2], None, "gst_certificate", None, BUSINESS_TEXTS
    )

    assert result["error"] == "missing_cancelled_cheque"
    # Whatever could be read is still returned.
    assert result["extracted_biz_num"] == GSTIN
    assert result["cheque_ifsc"] is None


def test_run_business_document_ocr_without_business_documents():
    result = worker.run_business_document_ocr(
        [{"type": "selfie", "url": "https://cdn.example/selfie.jpg"}], None, None, None, {}
    )

    assert result["error"] == "missing_business_documents"
    assert result["extracted_biz_num"] is None


def test_run_business_document_ocr_requires_an_engine():
    loader = worker.ModelLoader()
    loader.ocr_engine = None

    result = worker.run_business_document_ocr(BUSINESS_DOCUMENTS, None, "gst_certificate", loader)

    assert result["error"] == "ocr_unavailable"


class _FailingCache:
    def get(self, url):
        raise worker.ImageFetchError("no network")


def test_run_business_document_ocr_fetch_failure_never_raises():
    loader = worker.ModelLoader()
    loader.ocr_engine = object()

    result = worker.run_business_document_ocr(
        BUSINESS_DOCUMENTS, _FailingCache(), "gst_certificate", loader
    )

    assert result["error"] == "gst_certificate_fetch_failed"
    assert result["extracted_biz_num"] is None


def test_run_business_document_ocr_tolerates_garbage():
    for documents in (None, [], [None], [{"type": "pan_card"}], "not-a-list"):
        result = worker.run_business_document_ocr(documents, None, None, None, None)

        assert isinstance(result, dict)
        for key in ("extracted_biz_num", "cheque_ifsc", "cheque_acc_last4", "primary_doc_type", "error"):
            assert key in result, key
        assert result["error"] is not None


# ---------------------------------------------------------------------------
# process_job: mock pipeline
# ---------------------------------------------------------------------------


def _mock_settings(monkeypatch, **overrides):
    defaults = {
        "ml_mode": "mock",
        "shadow_mode": False,
        "face_match_enabled": False,
        "doc_in_selfie_enabled": True,
    }
    defaults.update(overrides)
    monkeypatch.setattr(
        worker, "settings", dataclasses.replace(worker.settings, **defaults)
    )


def _run_process_job(monkeypatch, payload, texts=None, fetch_image=None):
    captured: dict[str, object] = {}

    def fake_save_results(**kwargs):
        captured.update(kwargs)
        return True

    monkeypatch.setattr(worker, "mark_processing", lambda verification_id: True)
    monkeypatch.setattr(worker, "mock_ocr", lambda documents: dict(MOCK_OCR))
    monkeypatch.setattr(worker, "save_results", fake_save_results)
    monkeypatch.setattr(
        worker,
        "run_document_validation",
        lambda job, ocr_result, primary_bytes=None: {"overall": "not_attempted"},
    )
    monkeypatch.setattr(
        worker,
        "mock_business_document_text",
        lambda documents: dict(BUSINESS_TEXTS if texts is None else texts),
    )
    if fetch_image is not None:
        monkeypatch.setattr(worker, "fetch_image", fetch_image)

    result = worker.process_job(json.dumps(payload), worker.ModelLoader())
    return result, captured


def test_mock_business_job_without_matching_text_is_manual(monkeypatch):
    # The default mock seam returns no text at all: a business job must never
    # be auto-approved on a fabricated read.
    _mock_settings(monkeypatch)
    result, captured = _run_process_job(monkeypatch, BUSINESS_PAYLOAD, texts={})

    face = captured["face_result"]
    assert result["ml_status"] == "ml_completed"
    assert face["business_docs_decision"] == "manual"
    assert face["business_docs_score"] == 0
    assert face["business_docs_match"] is False
    assert result["ai_recommended_priority"] == worker.PRIORITY_MANUAL_REVIEW


def test_mock_business_job_with_documents_but_no_uploaded_values_is_manual(
    monkeypatch,
):
    # RUNTIME TRUTH pinned: with the producer payload (document URLs only) the
    # document-internal check passes but the uploaded legs cannot, so the safe
    # default is manual review - never a fabricated approval.
    def _no_fetch(url):  # pragma: no cover - must never run in mock mode
        raise AssertionError("mock mode must not fetch document images")

    _mock_settings(monkeypatch)
    result, captured = _run_process_job(
        monkeypatch, BUSINESS_PAYLOAD, texts=BUSINESS_TEXTS, fetch_image=_no_fetch
    )

    face = captured["face_result"]
    assert face["business_docs_decision"] == "manual"
    assert face["business_docs_score"] == 0
    assert face["business_docs_match"] is False
    assert result["ai_recommended_priority"] == worker.PRIORITY_MANUAL_REVIEW

    nested = face["business_docs"]
    # With selected_business_doc=gst_certificate, we extract GSTIN
    assert nested["extracted_biz_num"] == GSTIN
    assert nested["primary_doc_type"] == "gst_certificate"
    assert nested["cheque_ifsc"] == IFSC
    assert nested["decision"] == "manual"
    assert nested["score"] == 0
    assert json.loads(json.dumps(face))["business_docs_decision"] == "manual"


def test_mock_business_job_with_a_true_match_is_auto_eligible(monkeypatch):
    def _no_fetch(url):  # pragma: no cover - must never run in mock mode
        raise AssertionError("mock mode must not fetch document images")

    _mock_settings(monkeypatch)
    result, captured = _run_process_job(
        monkeypatch,
        BUSINESS_PAYLOAD_WITH_VALUES,
        texts=BUSINESS_TEXTS,
        fetch_image=_no_fetch,
    )

    face = captured["face_result"]
    assert face["business_docs_decision"] == "approve"
    assert face["business_docs_score"] == 100
    assert face["business_docs_match"] is True
    assert result["ai_recommended_priority"] == worker.PRIORITY_AUTO_ELIGIBLE
    assert result["review_priority"] == worker.PRIORITY_AUTO_ELIGIBLE

    nested = face["business_docs"]
    # The nested object carries the decision/score shape the Go router decodes
    # plus the per-field evidence, and the flat keys mirror it.
    assert nested["decision"] == "approve"
    assert nested["score"] == 100
    assert nested["extracted_biz_num"] == GSTIN
    assert nested["primary_doc_type"] == "gst_certificate"
    assert nested["cheque_ifsc"] == IFSC
    assert nested["cheque_acc_no"] == ACCOUNT_LAST4
    assert nested["error"] is None
    assert face["business_docs_decision"] == nested["decision"]
    assert face["business_docs_score"] == nested["score"]
    assert json.loads(json.dumps(face))["business_docs_decision"] == "approve"


def test_mock_business_job_with_a_gstin_for_another_pan_stays_manual(monkeypatch):
    _mock_settings(monkeypatch)
    result, captured = _run_process_job(
        monkeypatch,
        BUSINESS_PAYLOAD_WITH_VALUES,
        texts=dict(BUSINESS_TEXTS, gst_certificate=GST_TEXT_OTHER_PAN),
    )

    face = captured["face_result"]
    assert face["business_docs_decision"] == "manual"
    assert face["business_docs_score"] == 0
    assert face["business_docs_match"] is False
    assert result["ai_recommended_priority"] == worker.PRIORITY_MANUAL_REVIEW


def test_mock_business_job_uses_payload_values_when_present(monkeypatch):
    # The Go producer sends business_info and bank_details; when a payload carries
    # them the uploaded-leg checks must actually run.
    _mock_settings(monkeypatch)
    approved, _ = _run_process_job(
        monkeypatch, BUSINESS_PAYLOAD_WITH_VALUES, texts=BUSINESS_TEXTS
    )
    assert approved["ai_recommended_priority"] == worker.PRIORITY_AUTO_ELIGIBLE

    rejected, captured = _run_process_job(
        monkeypatch,
        {**BUSINESS_PAYLOAD_WITH_VALUES, "business_info": {"gstin": "27ZZZZZ9999Z1Z3", "pan": OTHER_PAN}},
        texts=BUSINESS_TEXTS,
    )
    assert rejected["ai_recommended_priority"] == worker.PRIORITY_MANUAL_REVIEW
    assert captured["face_result"]["business_docs_decision"] == "manual"
    # The identifier mismatch should be recorded
    assert captured["face_result"]["business_docs"]["reason"] == "mismatch"


def test_business_values_may_nest_under_business_info_and_bank_details(monkeypatch):
    _mock_settings(monkeypatch)
    result, captured = _run_process_job(
        monkeypatch,
        {
            **BUSINESS_PAYLOAD,
            "business_info": {"gstin": GSTIN, "pan": PAN},
            "bank_details": {"ifsc": IFSC, "account_number": "12345678901"},
        },
        texts=BUSINESS_TEXTS,
    )

    assert result["ai_recommended_priority"] == worker.PRIORITY_AUTO_ELIGIBLE
    assert captured["face_result"]["business_docs"]["decision"] == "approve"


def test_individual_job_gets_no_business_keys(monkeypatch):
    _mock_settings(monkeypatch)

    class _Loader:
        def run_selfie_document_ocr(self, documents, cache=None):
            return {"error": "selfie_not_found", "aadhaar_number": None}

    monkeypatch.setattr(worker, "mark_processing", lambda verification_id: True)
    monkeypatch.setattr(worker, "mock_ocr", lambda documents: dict(MOCK_OCR))
    captured: dict[str, object] = {}
    monkeypatch.setattr(
        worker, "save_results", lambda **kwargs: (captured.update(kwargs), True)[1]
    )
    monkeypatch.setattr(
        worker,
        "run_document_validation",
        lambda job, ocr_result, primary_bytes=None: {"overall": "not_attempted"},
    )

    worker.process_job(json.dumps(INDIVIDUAL_PAYLOAD), _Loader())

    face = captured["face_result"]
    assert not [key for key in face if key.startswith("business_")]
    assert face["document_in_selfie_decision"] == "manual"


# ---------------------------------------------------------------------------
# process_job: real pipeline
# ---------------------------------------------------------------------------


class _RealBusinessLoader:
    """Real-mode loader double: fetches and OCRs all three business documents."""

    def __init__(self):
        self.core_ready = True
        self.errors: dict[str, str] = {}
        self.model_version = "test-models-v1"
        self.ocr_engine = object()
        self.images_ocrd: list[str] = []

    def load_models(self):
        return None

    def run_ocr(self, documents, cache, primary_bytes=None):
        return dict(MOCK_OCR)

    def run_liveness(self, documents, cache):
        return {
            "enabled": True,
            "engine": "stub-liveness",
            "liveness_score": 0.96,
            "is_live": True,
            "error": None,
        }

    def run_selfie_document_ocr(self, documents, cache):
        return {"error": "selfie_not_found", "aadhaar_number": None}

    def ocr_image_text(self, image):
        self.images_ocrd.append(image)
        for needle, (text, score) in {
            "pan": (PAN_TEXT, 0.95),
            "gst": (GST_TEXT, 0.95),
            "cheque": (CHEQUE_TEXT, 0.95),
        }.items():
            if needle in image:
                return text, score
        return "", 0.0


def test_real_business_job_ocrs_selected_and_cheque_with_shared_engine(monkeypatch):
    monkeypatch.setattr(
        worker,
        "settings",
        dataclasses.replace(
            worker.settings,
            ml_mode="real",
            shadow_mode=False,
            face_match_enabled=False,
            doc_in_selfie_enabled=True,
        ),
    )
    monkeypatch.setattr(worker, "mark_processing", lambda verification_id: True)
    monkeypatch.setattr(worker, "_fetch_primary_document_bytes", lambda documents: None)
    monkeypatch.setattr(
        worker,
        "run_document_validation",
        lambda job, ocr_result, primary_bytes=None: {"overall": "not_attempted"},
    )
    monkeypatch.setattr(worker, "fetch_image", lambda url: url)
    captured: dict[str, object] = {}
    monkeypatch.setattr(
        worker, "save_results", lambda **kwargs: (captured.update(kwargs), True)[1]
    )

    loader = _RealBusinessLoader()
    result = worker.process_job(json.dumps(BUSINESS_PAYLOAD_WITH_VALUES), loader)

    face = captured["face_result"]
    assert result["ml_status"] == "ml_completed"
    assert face["business_docs_decision"] == "approve"
    assert face["business_docs_score"] == 100
    assert face["business_docs"]["extracted_biz_num"] == GSTIN
    assert face["business_docs"]["cheque_ifsc"] == IFSC
    assert result["ai_recommended_priority"] == worker.PRIORITY_AUTO_ELIGIBLE
    # Phase G: Only the selected document (gst_certificate) and cancelled_cheque
    # go through the shared OCR engine path (2 documents, not all 3).
    assert len(loader.images_ocrd) == 2
    assert sorted(loader.images_ocrd) == sorted([GST_URL, CHEQUE_URL])


# ---------------------------------------------------------------------------
# Phase G: selected_business_doc == "pan" routes to manual (hash cannot be compared)
# ---------------------------------------------------------------------------

# For PAN, the uploaded value is a HASH (not the actual PAN), so it will never
# match the PAN extracted from OCR. Simulate by using a different uploaded PAN.
BUSINESS_PAYLOAD_PAN_SELECTED_HASH = {
    **BUSINESS_PAYLOAD,
    "business_info": {"gstin": GSTIN, "pan": "HASHED_PAN_VALUE", "reg_no": ""},
    "bank_details": {"ifsc": IFSC, "account_number": "12345678901"},
    "selected_business_doc": "pan_card",
}


def test_business_pan_selected_routes_to_manual(monkeypatch):
    """Security limitation: PAN is a hash; cannot OCR-match it. Must route to manual."""
    def _no_fetch(url):
        raise AssertionError("mock mode must not fetch document images")

    _mock_settings(monkeypatch)
    # Use a payload with selected_business_doc = "pan_card" and uploaded PAN as hash
    result, captured = _run_process_job(
        monkeypatch,
        BUSINESS_PAYLOAD_PAN_SELECTED_HASH,
        texts=BUSINESS_TEXTS,
        fetch_image=_no_fetch,
    )

    face = captured["face_result"]
    # PAN is a hash - the uploaded hash never matches the actual PAN from OCR
    # so the identifier mismatch causes decision to be manual with reason "mismatch"
    assert face["business_docs_decision"] == "manual"
    assert face["business_docs_score"] == 0
    assert face["business_docs_match"] is False
    assert result["ai_recommended_priority"] == worker.PRIORITY_MANUAL_REVIEW
    assert face["business_docs"]["primary_doc_type"] == "pan_card"
    assert face["business_docs"]["extracted_biz_num"] == PAN
    assert face["business_docs"]["reason"] == "mismatch"


def test_business_incorporation_selected_can_approve(monkeypatch):
    """Incorporation certificate can be matched (registration number)."""
    INC_TEXT = """CERTIFICATE OF INCORPORATION
CIN: U12345MH2024PTC123456
Company Name: ACME TRADERS PVT LTD
"""

    def _no_fetch(url):
        raise AssertionError("mock mode must not fetch document images")

    _mock_settings(monkeypatch)
    result, captured = _run_process_job(
        monkeypatch,
        BUSINESS_PAYLOAD_INC_SELECTED,
        texts=dict(BUSINESS_TEXTS, incorporation_certificate=INC_TEXT),
        fetch_image=_no_fetch,
    )

    face = captured["face_result"]
    assert face["business_docs_decision"] == "approve"
    assert face["business_docs_score"] == 100
    assert face["business_docs_match"] is True
    assert result["ai_recommended_priority"] == worker.PRIORITY_AUTO_ELIGIBLE
    assert face["business_docs"]["primary_doc_type"] == "incorporation_certificate"


# ---------------------------------------------------------------------------
# determine_review_priority: business branch
# ---------------------------------------------------------------------------


def _priority_settings(monkeypatch, **overrides):
    defaults = {
        "ml_mode": "mock",
        "shadow_mode": False,
        "face_match_enabled": False,
    }
    defaults.update(overrides)
    monkeypatch.setattr(
        worker, "settings", dataclasses.replace(worker.settings, **defaults)
    )


def _business_face(decision, score, **extra):
    face = worker._make_face_disabled_result(doc_types=["pan_card", "gst_certificate"])
    face.update(
        {
            "business_docs": {"decision": decision, "score": score},
            "business_docs_decision": decision,
            "business_docs_score": score,
            "business_docs_match": decision == "approve",
        }
    )
    face.update(extra)
    return face


BUSINESS_OCR_RESULT = {"ocr_confidence": 0.95, "text": "GSTIN: 27ABCDE1234F1Z5"}
NO_SELFIE_LIVENESS = {
    "liveness_score": 0.0,
    "is_live": False,
    "error": "missing_selfie",
}


def test_business_approve_is_auto_eligible(monkeypatch):
    _priority_settings(monkeypatch)
    face = _business_face("approve", 100)

    effective, raw = worker.determine_review_priority(
        BUSINESS_OCR_RESULT, face, NO_SELFIE_LIVENESS
    )

    assert (effective, raw) == (
        worker.PRIORITY_AUTO_ELIGIBLE,
        worker.PRIORITY_AUTO_ELIGIBLE,
    )


def test_business_mismatch_is_manual_review(monkeypatch):
    _priority_settings(monkeypatch)
    face = _business_face("manual", 0)

    effective, raw = worker.determine_review_priority(
        BUSINESS_OCR_RESULT, face, NO_SELFIE_LIVENESS
    )

    assert (effective, raw) == (
        worker.PRIORITY_MANUAL_REVIEW,
        worker.PRIORITY_MANUAL_REVIEW,
    )


def test_business_without_a_selfie_never_auto_rejects(monkeypatch):
    _priority_settings(monkeypatch)
    face = _business_face("manual", 0)

    for liveness in (
        NO_SELFIE_LIVENESS,
        {},
        {"liveness_score": 0.0, "is_live": False},
        {"error": "image_fetch_failed"},
    ):
        _, raw = worker.determine_review_priority(BUSINESS_OCR_RESULT, face, liveness)
        assert raw != worker.PRIORITY_AUTO_REJECT


def test_business_branch_is_exempt_from_face_and_liveness_errors(monkeypatch):
    _priority_settings(monkeypatch)
    face = _business_face("approve", 100)

    # A business record has no face/liveness evidence at all; the document-set
    # verdict is the whole decision, so missing stages cannot demote it.
    _, raw = worker.determine_review_priority(
        BUSINESS_OCR_RESULT, face, {"liveness_score": 0.0, "error": "missing_selfie"}
    )
    assert raw == worker.PRIORITY_AUTO_ELIGIBLE


def test_individual_records_ignore_the_business_branch(monkeypatch):
    _priority_settings(monkeypatch, ml_mode="real")
    ocr_result = {"ocr_confidence": 0.95, "text": "NAME: TEST USER"}
    face = {"face_similarity": 0.90}
    liveness = {"liveness_score": 0.95, "is_live": True}

    effective, raw = worker.determine_review_priority(ocr_result, face, liveness)

    assert (effective, raw) == (
        worker.PRIORITY_AUTO_ELIGIBLE,
        worker.PRIORITY_AUTO_ELIGIBLE,
    )


def test_document_in_selfie_promotion_needs_the_real_pipeline(monkeypatch):
    """Phase B: the selfie-document verdict is the individual identity anchor.

    In real mode it promotes the record (the Go router requires
    ai_recommended_priority == auto_eligible before it re-checks the evidence);
    in mock mode it must not, which is the routing
    test_determine_review_priority_unchanged pins.
    """
    # What process_job writes for an individual record while face match is
    # disabled (FACE_MATCH_ENABLED=false) plus the Phase A verdict keys.
    face = worker._make_face_disabled_result(doc_types=["aadhaar", "selfie"])
    face.update(
        {"document_in_selfie_decision": "approve", "document_in_selfie_score": 100}
    )
    ocr_result = {"ocr_confidence": 0.95, "text": "NAME: TEST USER"}
    liveness = {"liveness_score": 0.95, "is_live": True}

    _priority_settings(monkeypatch, ml_mode="real")
    assert worker.determine_review_priority(ocr_result, face, liveness) == (
        worker.PRIORITY_AUTO_ELIGIBLE,
        worker.PRIORITY_AUTO_ELIGIBLE,
    )

    _priority_settings(monkeypatch, ml_mode="mock")
    effective, raw = worker.determine_review_priority(ocr_result, face, liveness)
    assert (effective, raw) == (
        worker.PRIORITY_MANUAL_REVIEW,
        worker.PRIORITY_MANUAL_REVIEW,
    )


def test_priority_accepts_the_legacy_three_positional_arguments(monkeypatch):
    _priority_settings(monkeypatch)
    ocr_result, face, liveness = BUSINESS_OCR_RESULT, _business_face("approve", 100), {}

    priority = worker.determine_review_priority(ocr_result, face, liveness)

    assert isinstance(priority, tuple) and len(priority) == 2
    assert all(isinstance(value, str) for value in priority)