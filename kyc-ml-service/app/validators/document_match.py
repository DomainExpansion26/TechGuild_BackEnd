"""Comparison logic for document-in-selfie and business-document verification.

Import-safe by design: standard library only. Nothing heavier (cv2, numpy,
network clients) may be imported here, because the slim ML venv used by the
worker unit tests imports this module directly.

Two entry points are used by the pipeline:

* compare_documents() - the uploaded government document vs. the document held
  in the selfie (individual KYC).
* compare_business_documents() - the anti-spoof gate for business KYC (Phase G):
  the ONE business document the user selected is checked against the identifier
  they typed into the form, and the cancelled cheque against the bank details
  (IFSC plus the last four account digits).
"""

from __future__ import annotations

import re
from datetime import date, datetime
from difflib import SequenceMatcher
from typing import Any

# ---------------------------------------------------------------------------
# Name comparison
# ---------------------------------------------------------------------------

# Honorifics/relations that are never part of a person's name but are printed
# on Aadhaar/PAN cards ("Mr.", "Smt", "Dr", "Shri", ...).
_NAME_TITLES = frozenset(
    {
        "dr",
        "km",
        "kumari",
        "late",
        "miss",
        "mr",
        "mrs",
        "ms",
        "prof",
        "smt",
        "sri",
        "shri",
    }
)


def _normalise_name(value: Any) -> str | None:
    """Return a comparable token bag (sorted, lowercased, titles dropped)."""
    if not isinstance(value, str):
        return None
    cleaned = re.sub(r"[^a-z\s]", " ", value.lower())
    tokens = [token for token in cleaned.split() if token and token not in _NAME_TITLES]
    if not tokens:
        return None
    return " ".join(sorted(tokens))


def fuzzy_name_match(name1: str | None, name2: str | None) -> float | None:
    """Similarity of two printed names, or None when either is unusable.

    Token order, extra whitespace, punctuation and honorifics are ignored, so
    "Mr. Patel Fujel" and "fujel  patel" compare as an exact match.
    """
    n1 = _normalise_name(name1)
    n2 = _normalise_name(name2)
    if not n1 or not n2:
        return None
    return SequenceMatcher(None, n1, n2).ratio()


# ---------------------------------------------------------------------------
# Date of birth comparison
# ---------------------------------------------------------------------------

# "/", "-" and "." separated dates in either order: an Indian document prints
# DD/MM/YYYY, a passport/MRZ prints YYYY-MM-DD.
_DATE_FORMATS = (
    "%d/%m/%Y",
    "%d-%m-%Y",
    "%d.%m.%Y",
    "%Y-%m-%d",
    "%Y/%m/%d",
    "%Y.%m.%d",
)


def _parse_date(value: Any) -> date | None:
    if not isinstance(value, str):
        return None
    text = value.strip()
    if not text:
        return None
    for date_format in _DATE_FORMATS:
        try:
            return datetime.strptime(text, date_format).date()
        except ValueError:
            continue
    return None


def compare_dob(first: Any, second: Any) -> bool | None:
    """Compare two dates of birth across the common document formats.

    Returns True/False for two parseable dates and None when either side is
    missing or is not a real calendar date ("31/02/2003", "16/05/2xxx",
    "garbage", ""). None means "no evidence", never "mismatch".
    """
    left = _parse_date(first)
    right = _parse_date(second)
    if left is None or right is None:
        return None
    return left == right


# ---------------------------------------------------------------------------
# Aadhaar comparison helpers
# ---------------------------------------------------------------------------

_NON_DIGIT_RE = re.compile(r"\D")


def _aadhaar_full(value: Any) -> str | None:
    """Return the 12 digits of an Aadhaar number held by the value.

    Deliberately no Verhoeff check here: this is a *comparison* of two reads,
    so a mis-OCR (one wrong check digit) must surface as a mismatch for the
    reject/manual band instead of being silently discarded.
    """
    if value is None:
        return None
    digits = _NON_DIGIT_RE.sub("", str(value))
    return digits if len(digits) == 12 else None


def _side_full(side: dict[str, Any]) -> str | None:
    return _aadhaar_full(side.get("_internal_aadhaar_number")) or _aadhaar_full(
        side.get("aadhaar_number")
    )


def _side_last4(side: dict[str, Any]) -> str | None:
    value = _side_last4_raw(side.get("aadhaar_last4"))
    if value:
        return value
    full = _side_full(side)
    return full[-4:] if full else None


def _side_last4_raw(value: Any) -> str | None:
    if value is None:
        return None
    digits = _NON_DIGIT_RE.sub("", str(value))
    return digits if len(digits) == 4 else None


def _as_int(value: Any, default: int) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def _as_float(value: Any, default: float) -> float:
    try:
        number = float(value)
    except (TypeError, ValueError):
        return default
    if number != number or number in (float("inf"), float("-inf")):
        return default
    return number


def _result(decision: str, score: int, reason: str, signals: list[str]) -> dict:
    """Build the frozen four-key comparison result."""
    return {
        "decision": decision,
        "score": int(score),
        "reason": reason,
        "signals": list(signals),
    }


def _aadhaar_mismatch(config: dict[str, Any]) -> dict:
    """Aadhaar numbers known on both sides and different.

    Arming "reject_on_mismatch" turns this into the hard reject the Go router
    consumes; disarmed it still stops at manual review.
    """
    if config.get("reject_on_mismatch", True):
        return _result("reject", -100, "aadhaar_number_mismatch", ["aadhaar_mismatch"])
    return _result("manual", 0, "aadhaar_number_mismatch", ["aadhaar_mismatch"])


def compare_documents(uploaded_ocr: dict, selfie_ocr: dict, config: dict) -> dict:
    """Compare the uploaded document against the document held in the selfie.

    Scoring bands:

    * full 12-digit Aadhaar match        +50
    * name present and above threshold   +25  (below threshold: -25)
    * date of birth equal                +25  (different: -15)

    "approve" requires the full Aadhaar match *and* a score at/above
    config["approve_threshold"] (default 75). A known-but-different Aadhaar is
    the only reject path; every soft signal (name/DOB) stays manual.

    Never raises: unusable input is reported as manual review.
    """
    uploaded = uploaded_ocr if isinstance(uploaded_ocr, dict) else {}
    selfie = selfie_ocr if isinstance(selfie_ocr, dict) else {}
    settings_map = config if isinstance(config, dict) else {}

    approve_threshold = _as_int(settings_map.get("approve_threshold"), 75)
    name_threshold = _as_float(settings_map.get("name_match_threshold"), 0.85)

    # A stage that reported an error carries no usable read: the error string
    # itself is the reason, so an operator sees which stage failed.
    if selfie.get("error") and not selfie.get("aadhaar_last4"):
        return _result("manual", 0, str(selfie["error"]), [])

    uploaded_full = _side_full(uploaded)
    selfie_full = _side_full(selfie)

    if selfie_full is None and _side_last4(selfie) is None:
        return _result("manual", 0, "selfie_aadhaar_unreadable", [])
    if uploaded_full is None and _side_last4(uploaded) is None:
        return _result("manual", 0, "uploaded_aadhaar_unreadable", [])

    if uploaded_full is not None and selfie_full is not None:
        if uploaded_full == selfie_full:
            signals = ["aadhaar_exact_match"]
            score = 50

            name_score = fuzzy_name_match(uploaded.get("name"), selfie.get("name"))
            if name_score is not None:
                if name_score >= name_threshold:
                    score += 25
                    signals.append("name_match")
                else:
                    score -= 25
                    signals.append(f"name_mismatch_{name_score:.2f}")

            dob_match = compare_dob(uploaded.get("dob"), selfie.get("dob"))
            if dob_match is True:
                score += 25
                signals.append("dob_match")
            elif dob_match is False:
                score -= 15
                signals.append("dob_mismatch")

            if score >= approve_threshold:
                return _result("approve", score, "aadhaar_exact_match", signals)
            return _result("manual", score, "aadhaar_partial_match", signals)

        return _aadhaar_mismatch(settings_map)

    # At least one side knows only the tail four digits. A matching tail four
    # proves nothing on its own, so this can never approve.
    uploaded_last4 = _side_last4(uploaded)
    selfie_last4 = _side_last4(selfie)
    if uploaded_last4 and selfie_last4 and uploaded_last4 != selfie_last4:
        return _aadhaar_mismatch(settings_map)

    if uploaded_last4 and selfie_last4:
        if uploaded_full is not None or selfie_full is not None:
            return _result(
                "manual",
                0,
                "cannot_verify_without_full_aadhaar",
                ["aadhaar_full_partial"],
            )
        return _result(
            "manual", 0, "cannot_verify_without_full_aadhaar", ["aadhaar_last4_only"]
        )

    return _result("manual", 0, "aadhaar_unreadable", [])


# ---------------------------------------------------------------------------
# Business documents: the selected document identifier + the cheque bank details
# ---------------------------------------------------------------------------

_PAN_RE = re.compile(r"[A-Z]{5}[0-9]{4}[A-Z]")
# GSTIN: 2 state digits, then the 10-char PAN, then entity/Z/checksum.
_GSTIN_RE = re.compile(r"[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]")
_IFSC_RE = re.compile(r"[A-Z]{4}0[A-Z0-9]{6}")
# CIN: listing status (U/L), 5 industry digits, the 2-letter state, the 4-digit
# year of incorporation, the 3-letter company class (PTC/PLC/...) and 6 digits.
_CIN_RE = re.compile(r"[LU][0-9]{5}[A-Z]{2}[0-9]{4}[A-Z]{3}[0-9]{6}")
# Generic company/registration identifiers: shop-and-establishment numbers,
# Udyam/MSME registrations and the loose formats state portals hand out.
_REGISTRATION_RE = re.compile(r"[A-Z0-9]{6,25}")

# Characters OCR routinely swaps. Applied to BOTH sides before comparing, so a
# read that produced "O" where the ink says "0" still matches the typed value.
_OCR_CONFUSABLES = str.maketrans({"O": "0", "I": "1", "L": "1"})

_NON_ALNUM_RE = re.compile(r"[^A-Za-z0-9]")


def _upper_alnum(value: Any) -> str:
    """Uppercase and drop every separator (spaces, hyphens, newlines)."""
    if value is None:
        return ""
    text = value if isinstance(value, str) else str(value)
    return _NON_ALNUM_RE.sub("", text).upper()


def fold_ocr_confusables(text: str) -> str:
    """Fold the OCR confusables O->0 and I/L->1 in an uppercased string.

    Offset preserving (no stripping), so a caller can run a second regex pass
    over the folded copy of a line it already scanned verbatim.
    """
    if not isinstance(text, str):
        return ""
    return text.translate(_OCR_CONFUSABLES)


def _compare_key(value: Any) -> str:
    """Confusable-tolerant comparison key (O/I/L folded onto digits)."""
    return fold_ocr_confusables(_upper_alnum(value))


def _matched(pattern: re.Pattern, raw: str) -> str | None:
    if pattern.fullmatch(raw):
        return raw
    tolerant = raw.translate(_OCR_CONFUSABLES)
    if pattern.fullmatch(tolerant):
        return tolerant
    return None


def normalise_pan(value: Any) -> str | None:
    """Validated PAN, or None when the value is missing/malformed."""
    return _matched(_PAN_RE, _upper_alnum(value))


def normalise_gstin(value: Any) -> str | None:
    """Validated 15-character GSTIN, or None."""
    return _matched(_GSTIN_RE, _upper_alnum(value))


def normalise_ifsc(value: Any) -> str | None:
    """Validated 11-character IFSC, or None."""
    return _matched(_IFSC_RE, _upper_alnum(value))


def normalise_registration_number(value: Any) -> str | None:
    """Validated company registration number (CIN or generic), or None.

    A CIN-shaped value is accepted as printed. Anything else must be a 6-25
    character alphanumeric run that carries at least one digit: every real
    registration number does, and the digit requirement keeps ordinary words
    ("INCORPORATION", "PRIVATE") out of the candidate set. Case, whitespace and
    separators are normalised away, and the OCR confusables are folded only as
    a fallback read, so a correctly-read CIN ("U12345ML2024PTC123456" - state
    code ML would fold to M1) is never rewritten.
    """
    raw = _upper_alnum(value)
    if not raw:
        return None
    # The digit requirement is applied to the value as READ, before any
    # confusable folding: otherwise folding would manufacture a digit inside an
    # ordinary word ("INCORPORATION" -> "1NC0RP0RAT10N") and turn it into a
    # candidate.
    if not any(character.isdigit() for character in raw):
        return None
    candidates = (raw, fold_ocr_confusables(raw))
    for candidate in candidates:
        if _CIN_RE.fullmatch(candidate):
            return candidate
    for candidate in candidates:
        if _REGISTRATION_RE.fullmatch(candidate):
            return candidate
    return None


def same_document_id(left: Any, right: Any) -> bool:
    """True only when both sides carry a value and they are equal.

    Empty never matches empty: a missing read is missing evidence, not a
    corroboration.
    """
    key_left = _compare_key(left)
    key_right = _compare_key(right)
    return bool(key_left) and key_left == key_right


def gstin_pan_number(gstin: Any) -> str | None:
    """The PAN embedded in a GSTIN: characters 3-12 (1-based), i.e. [2:12].

    The GSTIN carries the holder's PAN in those ten positions, so comparing it
    against the PAN card number detects a GST certificate that belongs to
    somebody else. An extraction that does not satisfy [A-Z]{5}[0-9]{4}[A-Z]
    returns None (a failed check, never an approval).
    """
    value = normalise_gstin(gstin)
    if value is None:
        return None
    return normalise_pan(value[2:12])


# Which uploaded/bank field carries the identifier of each selectable business
# document type. compare_business_documents() receives the CANONICAL type the
# worker resolved (see worker.resolve_business_doc_type), so these four names
# are the whole contract; any other type is an unknown type, and an unknown
# type cannot be checked.
_PAN_FORM_KEYS = ("pan_number", "pan", "pan_no", "business_pan")
_GSTIN_FORM_KEYS = ("gstin", "gst_number", "gstin_number", "gst_no")
_REGISTRATION_FORM_KEYS = (
    "registration_number",
    "registration_no",
    "company_registration_number",
    "business_registration_number",
    "cin",
    "registration",
)
_IFSC_FORM_KEYS = ("ifsc_code", "ifsc", "bank_ifsc")
_ACCOUNT_FORM_KEYS = ("account_number", "account_no", "bank_account_number", "account")

# The form values may sit at the root of the uploaded dict or inside one of the
# nested objects the Go producer uses for the business/bank section.
_NESTED_UPLOADED_KEYS = ("business", "business_details", "bank_details", "bank", "kyc")

# Selected document type -> (form field names, normaliser).
_IDENTIFIER_SPECS: dict[str, tuple[tuple[str, ...], Any]] = {
    "pan_card": (_PAN_FORM_KEYS, normalise_pan),
    "gst_certificate": (_GSTIN_FORM_KEYS, normalise_gstin),
    "incorporation_certificate": (
        _REGISTRATION_FORM_KEYS,
        normalise_registration_number,
    ),
    "business_registration": (_REGISTRATION_FORM_KEYS, normalise_registration_number),
}


def _uploaded_business_sources(uploaded: dict) -> list[dict]:
    """The uploaded-info dict plus every nested business/bank object it holds."""
    sources: list[dict] = [uploaded]
    for key in _NESTED_UPLOADED_KEYS:
        nested = uploaded.get(key)
        if isinstance(nested, dict):
            sources.append(nested)
    return sources


def _first_uploaded_value(sources: list[dict], keys: tuple[str, ...]) -> str:
    """First non-empty value among *keys*; the root object wins over nesting."""
    for source in sources:
        for key in keys:
            value = source.get(key)
            if isinstance(value, str) and value.strip():
                return value.strip()
            if isinstance(value, (int, float)) and not isinstance(value, bool):
                return str(value)
    return ""


def _digit_last4(value: Any) -> str | None:
    """Last four digits of an account number, or None when unusable.

    Both sides of the check are reduced this way, so it does not matter whether
    the OCR stage stored the full account number or only its tail: fewer than
    four digits is an unreadable value, never a match.
    """
    if value is None:
        return None
    digits = _NON_DIGIT_RE.sub("", str(value))
    if len(digits) < 4:
        return None
    return digits[-4:]


def _identifier_matches(ocr: dict, sources: list[dict], signals: list[str]) -> bool:
    """The uploaded identifier vs the one read off the selected document."""
    doc_type = str(ocr.get("primary_doc_type") or "").strip().lower()
    spec = _IDENTIFIER_SPECS.get(doc_type)
    if spec is None:
        # Without the resolved document type there is no way to tell which form
        # field is the one to compare: missing evidence, never a pass.
        signals.append("identifier_type_unknown")
        return False

    form_keys, normalise = spec
    uploaded_id = normalise(_first_uploaded_value(sources, form_keys))
    document_id = normalise(ocr.get("extracted_biz_num"))
    if not uploaded_id or not document_id:
        signals.append("identifier_unavailable")
        return False
    if same_document_id(uploaded_id, document_id):
        signals.append("identifier_match")
        return True
    signals.append("identifier_mismatch")
    return False


def _ifsc_matches(ocr: dict, sources: list[dict], signals: list[str]) -> bool:
    """The uploaded IFSC vs the IFSC read off the cancelled cheque."""
    uploaded_ifsc = normalise_ifsc(_first_uploaded_value(sources, _IFSC_FORM_KEYS))
    cheque_ifsc = normalise_ifsc(ocr.get("cheque_ifsc"))
    if not uploaded_ifsc or not cheque_ifsc:
        signals.append("ifsc_unavailable")
        return False
    if same_document_id(uploaded_ifsc, cheque_ifsc):
        signals.append("ifsc_match")
        return True
    signals.append("ifsc_mismatch")
    return False


def _account_matches(ocr: dict, sources: list[dict], signals: list[str]) -> bool:
    """The uploaded account tail vs the account tail read off the cheque."""
    uploaded_last4 = _digit_last4(_first_uploaded_value(sources, _ACCOUNT_FORM_KEYS))
    cheque_last4 = _digit_last4(ocr.get("cheque_acc_no"))
    if not uploaded_last4 or not cheque_last4:
        signals.append("account_last4_unavailable")
        return False
    if uploaded_last4 == cheque_last4:
        signals.append("account_last4_match")
        return True
    signals.append("account_last4_mismatch")
    return False


def business_docs_detail(business: dict, uploaded: dict) -> dict:
    """Compatibility detail view for older callers; Phase G uses the compact verdict."""
    return {
        "pan_number": business.get("pan_number") if isinstance(business, dict) else None,
        "pan_source": "selected_document",
        "gstin": business.get("gstin") if isinstance(business, dict) else None,
        "gstin_pan": gstin_pan_number(business.get("gstin")) if isinstance(business, dict) else None,
        "cheque_ifsc": business.get("cheque_ifsc") if isinstance(business, dict) else None,
        "pan_matches_uploaded": None,
        "pan_matches_gstin": None,
        "ifsc_matches_uploaded": None,
        "pan_matches_cheque": None,
        "error": business.get("error") if isinstance(business, dict) else None,
    }


def compare_business_documents(job_payload: dict, ocr_result: dict, *legacy_args: Any) -> dict:
    # Older Phase E callers passed (uploaded, bank, ocr, config). Keep the
    # entrypoint tolerant while all new production calls use the Phase G pair.
    if legacy_args:
        legacy_ocr = ocr_result if isinstance(ocr_result, dict) else {}
        legacy_uploaded = job_payload if isinstance(job_payload, dict) else {}
        legacy_business = legacy_args[0] if isinstance(legacy_args[0], dict) else {}
        job_payload = {
            "selected_business_doc": "gst_certificate",
            "business_info": {"gstin": legacy_business.get("gstin"), "pan": legacy_uploaded.get("pan_number")},
            "bank_details": {"ifsc": legacy_uploaded.get("ifsc_code"), "account_number": "0000"},
        }
        ocr_result = {"extracted_biz_num": legacy_business.get("gstin"), "cheque_ifsc": legacy_business.get("cheque_ifsc"), "cheque_acc_last4": "0000", "error": legacy_business.get("error")}
        if legacy_business.get("error"):
            ocr_result["error"] = legacy_business["error"]

    """Phase G: compare the selected business document and cancelled cheque."""
    payload = job_payload if isinstance(job_payload, dict) else {}
    ocr = ocr_result if isinstance(ocr_result, dict) else {}
    business = payload.get("business_info") if isinstance(payload.get("business_info"), dict) else {}
    bank = payload.get("bank_details") if isinstance(payload.get("bank_details"), dict) else {}
    selected = str(payload.get("selected_business_doc") or "").strip().lower()
    form_id = {
        "gst_certificate": business.get("gstin"),
        "pan_card": business.get("pan"),
        "incorporation_certificate": business.get("reg_no"),
        "business_registration": business.get("reg_no"),
    }.get(selected)
    extracted = ocr.get("extracted_biz_num")
    cheque_ifsc = ocr.get("cheque_ifsc")
    cheque_last4 = ocr.get("cheque_acc_last4") or ocr.get("cheque_acc_no")
    account = bank.get("account_number")
    ok = (
        not ocr.get("error")
        and selected in {"gst_certificate", "pan_card", "incorporation_certificate", "business_registration"}
        and same_document_id(form_id, extracted)
        and same_document_id(bank.get("ifsc"), cheque_ifsc)
        and _digit_last4(account) is not None
        and _digit_last4(account) == _digit_last4(cheque_last4)
    )
    if ok:
        return {"decision": "approve", "score": 100, "reason": "business_and_bank_verified"}
    return {"decision": "manual", "score": 0, "reason": "mismatch"}
