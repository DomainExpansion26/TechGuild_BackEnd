"""Unit tests for :mod:`app.validators.document_match` (Phase A).

Pure-Python: no worker import, no models, no network.
"""

from __future__ import annotations

import json

import pytest

from app.validators.document_match import (
    business_docs_detail,
    compare_business_documents,
    compare_documents,
    compare_dob,
    fuzzy_name_match,
    gstin_pan_number,
)

UPLOADED_AADHAAR = "234567890124"
SELFIE_AADHAAR = "234567890124"
OTHER_AADHAAR = "999999999999"

BASE_CONFIG = {
    "approve_threshold": 75,
    "manual_threshold": 25,
    "reject_on_mismatch": True,
    "name_match_threshold": 0.85,
}


def _config(**overrides):
    return {**BASE_CONFIG, **overrides}


def _uploaded(**overrides):
    fields = {
        "aadhaar_number": UPLOADED_AADHAAR,
        "aadhaar_last4": UPLOADED_AADHAAR[-4:],
        "name": "Patel Fujel",
        "dob": "16/05/2003",
    }
    fields.update(overrides)
    return fields


def _selfie(**overrides):
    fields = {
        "error": None,
        "aadhaar_number": SELFIE_AADHAAR,
        "name": "Patel Fujel",
        "dob": "16/05/2003",
        "confidence": 0.94,
        "text_redacted": "XXXX XXXX 0124",
    }
    fields.update(overrides)
    return fields


# ---------------------------------------------------------------------------
# fuzzy_name_match
# ---------------------------------------------------------------------------


@pytest.mark.parametrize(
    ("first", "second"),
    [
        ("Patel Fujel", "Patel Fujel"),
        ("patel fujel", "Patel Fujel"),
    ],
)
def test_fuzzy_name_match_identical(first, second):
    assert fuzzy_name_match(first, second) == 1.0


def test_fuzzy_name_match_ignores_token_order():
    assert fuzzy_name_match("Fujel Patel", "Patel Fujel") > 0.9


@pytest.mark.parametrize("title", ["Mr.", "Smt", "Dr", "Shri"])
def test_fuzzy_name_match_strips_titles(title):
    assert fuzzy_name_match(f"{title} Patel Fujel", "Patel Fujel") > 0.9


def test_fuzzy_name_match_ignores_extra_whitespace():
    assert fuzzy_name_match("  Patel   Fujel ", "Patel Fujel") == 1.0


@pytest.mark.parametrize("value", [None, "", "   ", "Mr."])
def test_fuzzy_name_match_none_when_unusable(value):
    assert fuzzy_name_match(value, "Patel Fujel") is None
    assert fuzzy_name_match("Patel Fujel", value) is None


def test_fuzzy_name_match_different_names():
    assert fuzzy_name_match("John Smith", "Rajesh Kumar") < 0.40


# ---------------------------------------------------------------------------
# compare_dob
# ---------------------------------------------------------------------------


def test_compare_dob_same_format():
    assert compare_dob("16/05/2003", "16/05/2003") is True


def test_compare_dob_across_formats():
    assert compare_dob("16/05/2003", "2003-05-16") is True


def test_compare_dob_different_dates():
    assert compare_dob("16/05/2003", "17/05/2003") is False


@pytest.mark.parametrize(
    ("first", "second"),
    [
        (None, "16/05/2003"),
        ("16/05/2003", None),
        ("garbage", "x"),
        ("16/05/2xxx", "16/05/2003"),
        ("31/02/2003", "31/02/2003"),
        ("", ""),
    ],
)
def test_compare_dob_unparseable_returns_none(first, second):
    assert compare_dob(first, second) is None


# ---------------------------------------------------------------------------
# compare_documents: happy paths
# ---------------------------------------------------------------------------


def test_all_signals_match_approves_with_full_score():
    result = compare_documents(_uploaded(), _selfie(), _config())

    assert result["decision"] == "approve"
    assert result["score"] == 100
    assert "aadhaar_exact_match" in result["signals"]
    assert "dob_match" in result["signals"]


def test_aadhaar_and_name_without_dob_approves_at_75():
    result = compare_documents(_uploaded(dob=None), _selfie(dob=None), _config())

    assert result["decision"] == "approve"
    assert result["score"] == 75


def test_aadhaar_only_is_manual_at_50():
    result = compare_documents(
        _uploaded(dob=None), _selfie(name=None, dob=None), _config()
    )

    assert result["decision"] == "manual"
    assert result["score"] == 50
    assert result["signals"] == ["aadhaar_exact_match"]


def test_aadhaar_match_with_name_mismatch_stays_manual():
    result = compare_documents(
        _uploaded(dob=None), _selfie(name="Rajesh Kumar", dob=None), _config()
    )

    assert result["decision"] == "manual"
    assert result["score"] < 75
    assert any(s.startswith("name_mismatch_") for s in result["signals"])


def test_dob_mismatch_penalises_score():
    result = compare_documents(
        _uploaded(name=None), _selfie(name=None, dob="17/05/2003"), _config()
    )

    assert result["decision"] == "manual"
    assert result["score"] == 35
    assert "dob_mismatch" in result["signals"]


def test_soft_signals_never_reject():
    # A name mismatch plus a DOB mismatch is still only worth manual review.
    result = compare_documents(
        _uploaded(), _selfie(name="Rajesh Kumar", dob="17/05/2003"), _config()
    )

    assert result["decision"] == "manual"
    assert result["score"] == 10


def test_higher_approve_threshold_can_force_manual():
    result = compare_documents(
        _uploaded(dob=None), _selfie(dob=None), _config(approve_threshold=90)
    )

    assert result["decision"] == "manual"
    assert result["score"] == 75


# ---------------------------------------------------------------------------
# compare_documents: aadhaar mismatch band
# ---------------------------------------------------------------------------


def test_aadhaar_mismatch_rejects_when_armed():
    result = compare_documents(
        _uploaded(), _selfie(aadhaar_number=OTHER_AADHAAR), _config()
    )

    assert result["decision"] == "reject"
    assert result["score"] == -100
    assert result["reason"] == "aadhaar_number_mismatch"
    assert result["signals"] == ["aadhaar_mismatch"]


def test_aadhaar_mismatch_is_manual_when_disarmed():
    result = compare_documents(
        _uploaded(),
        _selfie(aadhaar_number=OTHER_AADHAAR),
        _config(reject_on_mismatch=False),
    )

    assert result["decision"] == "manual"
    assert result["score"] == 0
    assert result["reason"] == "aadhaar_number_mismatch"
    assert result["signals"] == ["aadhaar_mismatch"]


def test_aadhaar_separators_do_not_cause_a_false_mismatch():
    result = compare_documents(
        _uploaded(aadhaar_number="2345 6789 0124"),
        _selfie(aadhaar_number="2345-6789-0124"),
        _config(),
    )

    assert result["decision"] == "approve"


# ---------------------------------------------------------------------------
# compare_documents: missing evidence
# ---------------------------------------------------------------------------


def test_selfie_aadhaar_missing_is_manual():
    result = compare_documents(
        _uploaded(), _selfie(aadhaar_number=None), _config()
    )

    assert result["decision"] == "manual"
    assert result["score"] == 0
    assert result["reason"] == "selfie_aadhaar_unreadable"
    assert result["signals"] == []


def test_uploaded_aadhaar_missing_is_manual():
    result = compare_documents(
        _uploaded(aadhaar_number=None, aadhaar_last4=None), _selfie(), _config()
    )

    assert result["decision"] == "manual"
    assert result["score"] == 0
    assert result["reason"] == "uploaded_aadhaar_unreadable"
    assert result["signals"] == []


@pytest.mark.parametrize(
    "error",
    ["selfie_not_found", "selfie_fetch_failed", "selfie_ocr_low_confidence"],
)
def test_selfie_error_is_manual(error):
    result = compare_documents(_uploaded(), _selfie(error=error), _config())

    assert result["decision"] == "manual"
    assert result["score"] == 0
    assert result["reason"] == error
    assert result["signals"] == []


# ---------------------------------------------------------------------------
# compare_documents: tail-four fallback (uploaded side carries only last4)
# ---------------------------------------------------------------------------


def test_tail_match_no_longer_approves():
    # Phase A.5: a matching tail four proves nothing (two different people can
    # share it), so the old ``aadhaar_tail_match`` approval must be gone.
    result = compare_documents(
        _uploaded(aadhaar_number=None), _selfie(), _config()
    )

    assert result["decision"] == "manual"
    assert result["score"] == 0
    assert result["reason"] == "cannot_verify_without_full_aadhaar"
    assert result["signals"] == ["aadhaar_full_partial"]


def test_tail_mismatch_rejects_when_armed():
    result = compare_documents(
        _uploaded(aadhaar_number=None),
        _selfie(aadhaar_number=OTHER_AADHAAR),
        _config(),
    )

    assert result["decision"] == "reject"
    assert result["score"] == -100
    assert result["signals"] == ["aadhaar_mismatch"]


def test_tail_mismatch_is_manual_when_disarmed():
    result = compare_documents(
        _uploaded(aadhaar_number=None),
        _selfie(aadhaar_number=OTHER_AADHAAR),
        _config(reject_on_mismatch=False),
    )

    assert result["decision"] == "manual"
    assert result["score"] == 0


# ---------------------------------------------------------------------------
# compare_documents: full-Aadhaar policy (Phase A.5)
# ---------------------------------------------------------------------------


def test_full_aadhaar_match_approves():
    result = compare_documents(
        _uploaded(_internal_aadhaar_number=UPLOADED_AADHAAR),
        _selfie(_internal_aadhaar_number=SELFIE_AADHAAR),
        _config(),
    )

    assert result["decision"] == "approve"
    assert result["score"] >= 75
    assert result["signals"][0] == "aadhaar_exact_match"


def test_full_aadhaar_mismatch_rejects():
    result = compare_documents(
        _uploaded(_internal_aadhaar_number=UPLOADED_AADHAAR),
        _selfie(_internal_aadhaar_number="234567890125"),
        _config(reject_on_mismatch=True),
    )

    assert result["decision"] == "reject"
    assert result["score"] == -100
    assert result["signals"] == ["aadhaar_mismatch"]


def test_last4_only_never_approves():
    result = compare_documents(
        _uploaded(aadhaar_number=None, aadhaar_last4="0124"),
        _selfie(aadhaar_number=None, aadhaar_last4="0124"),
        _config(),
    )

    assert result["decision"] == "manual"
    assert result["score"] == 0
    assert result["signals"] == ["aadhaar_last4_only"]
    assert "cannot_verify" in result["reason"]


def test_partial_full_manual():
    # Uploaded side knows the full number, the selfie side only its tail four.
    result = compare_documents(
        _uploaded(_internal_aadhaar_number=UPLOADED_AADHAAR),
        _selfie(aadhaar_number=None, aadhaar_last4="0124"),
        _config(),
    )

    assert result["decision"] == "manual"
    assert result["score"] == 0
    assert result["signals"] == ["aadhaar_full_partial"]


# ---------------------------------------------------------------------------
# Contract shape
# ---------------------------------------------------------------------------


def test_result_is_json_serializable():
    result = compare_documents(_uploaded(), _selfie(), _config())

    assert json.loads(json.dumps(result)) == result


@pytest.mark.parametrize(
    ("uploaded", "selfie"),
    [
        (_uploaded(), _selfie()),
        (_uploaded(dob=None), _selfie(name=None)),
        ({}, {}),
        (None, None),
    ],
)
def test_result_shape_and_types(uploaded, selfie):
    result = compare_documents(uploaded, selfie, {})

    assert set(result) == {"decision", "score", "reason", "signals"}
    assert result["decision"] in {"approve", "manual", "reject"}
    assert isinstance(result["score"], int)
    assert isinstance(result["reason"], str)
    assert isinstance(result["signals"], list)


def test_decision_score_bands_are_consistent():
    cases = [
        (_uploaded(), _selfie(), "approve"),
        (_uploaded(), _selfie(aadhaar_number=OTHER_AADHAAR), "reject"),
        (_uploaded(), _selfie(error="selfie_not_found"), "manual"),
        (_uploaded(aadhaar_number=None, aadhaar_last4=None), _selfie(), "manual"),
    ]


    for uploaded, selfie, expected in cases:
        assert compare_documents(uploaded, selfie, _config())["decision"] == expected


# ---------------------------------------------------------------------------
# Phase G: one selected business document plus cancelled cheque
# ---------------------------------------------------------------------------

GSTIN = "27ABCDE1234F1Z5"
PAN = "ABCDE1234F"
IFSC = "HDFC0001234"


def _phase_g_payload(**overrides):
    payload = {
        "selected_business_doc": "gst_certificate",
        "business_info": {"gstin": GSTIN, "pan": PAN, "reg_no": ""},
        "bank_details": {"ifsc": IFSC, "account_number": "12345678901234"},
    }
    payload.update(overrides)
    return payload


def _phase_g_ocr(**overrides):
    result = {
        "extracted_biz_num": GSTIN,
        "cheque_ifsc": IFSC,
        "cheque_acc_last4": "1234",
    }
    result.update(overrides)
    return result


def test_phase_g_all_three_matches_approve():
    assert compare_business_documents(_phase_g_payload(), _phase_g_ocr()) == {
        "decision": "approve", "score": 100, "reason": "business_and_bank_verified"
    }


@pytest.mark.parametrize("field", ["extracted_biz_num", "cheque_ifsc", "cheque_acc_last4"])
def test_phase_g_any_mismatch_routes_manual(field):
    ocr = _phase_g_ocr(**{field: "wrong"})
    result = compare_business_documents(_phase_g_payload(), ocr)
    assert result == {"decision": "manual", "score": 0, "reason": "mismatch"}


def test_phase_g_requires_selected_document_identifier():
    payload = _phase_g_payload(selected_business_doc="pan_card")
    assert compare_business_documents(payload, _phase_g_ocr()) == {
        "decision": "manual", "score": 0, "reason": "mismatch"
    }


def test_phase_g_requires_bank_values_and_ocr_error_blocks_approval():
    payload = _phase_g_payload(bank_details={"ifsc": "", "account_number": ""})
    assert compare_business_documents(payload, _phase_g_ocr(error="ocr_failed")) == {
        "decision": "manual", "score": 0, "reason": "mismatch"
    }


def test_phase_g_never_matches_empty_values():
    assert compare_business_documents({}, {}) == {
        "decision": "manual", "score": 0, "reason": "mismatch"
    }


def test_phase_g_result_is_json_serializable():
    result = compare_business_documents(_phase_g_payload(), _phase_g_ocr())
    assert json.loads(json.dumps(result)) == result
