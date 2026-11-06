"""Validation helpers for Indian document identifiers found in OCR text."""

from __future__ import annotations

import re

from .verhoeff import is_valid_aadhaar

_PAN_RE = re.compile(r"^[A-Z]{3}[ABCFGHJLPT][A-Z][0-9]{4}[A-Z]$")
_SEPARATORS_RE = re.compile(r"[\s-]+")
_AADHAAR_SEPARATOR_RE = re.compile(r"(?<=[0-9])[\s-]+(?=[0-9])")
_AADHAAR_CANDIDATE_RE = re.compile(r"(?<![0-9])([0-9]{12})(?![0-9])")


def is_valid_pan(raw: str) -> bool:
    """Validate the PAN character layout (no public check-digit algorithm)."""
    if not isinstance(raw, str):
        return False

    candidate = _SEPARATORS_RE.sub("", raw).upper()
    return _PAN_RE.fullmatch(candidate) is not None


def find_aadhaar_in_text(text: str) -> str | None:
    """Return the first Verhoeff-valid 12-digit Aadhaar run in ``text``."""
    if not isinstance(text, str) or not text:
        return None

    # Collapse separators between digits first. This prevents a 16-digit VID
    # from being sliced into a false 12-digit Aadhaar match.
    collapsed = _AADHAAR_SEPARATOR_RE.sub("", text)
    for match in _AADHAAR_CANDIDATE_RE.finditer(collapsed):
        candidate = match.group(1)
        if is_valid_aadhaar(candidate):
            return candidate
    return None
