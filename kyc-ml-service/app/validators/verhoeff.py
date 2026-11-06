"""Verhoeff checksum and Aadhaar number validation.

The Verhoeff algorithm uses three tables: the dihedral group multiplication
table ``D``, the permutation table ``P``, and the multiplicative inverse table
``INV``. Only ASCII digits are accepted; Unicode digit characters are rejected
because ``str.isdigit()`` would otherwise treat them as valid.
"""

from __future__ import annotations

import re

_D = (
    (0, 1, 2, 3, 4, 5, 6, 7, 8, 9),
    (1, 2, 3, 4, 0, 6, 7, 8, 9, 5),
    (2, 3, 4, 0, 1, 7, 8, 9, 5, 6),
    (3, 4, 0, 1, 2, 8, 9, 5, 6, 7),
    (4, 0, 1, 2, 3, 9, 5, 6, 7, 8),
    (5, 9, 8, 7, 6, 0, 4, 3, 2, 1),
    (6, 5, 9, 8, 7, 1, 0, 4, 3, 2),
    (7, 6, 5, 9, 8, 2, 1, 0, 4, 3),
    (8, 7, 6, 5, 9, 3, 2, 1, 0, 4),
    (9, 8, 7, 6, 5, 4, 3, 2, 1, 0),
)

_P = (
    (0, 1, 2, 3, 4, 5, 6, 7, 8, 9),
    (1, 5, 7, 6, 2, 8, 3, 0, 9, 4),
    (5, 8, 0, 3, 7, 9, 6, 1, 4, 2),
    (8, 9, 1, 6, 0, 4, 3, 5, 2, 7),
    (9, 4, 5, 3, 1, 2, 6, 8, 7, 0),
    (4, 2, 8, 6, 5, 7, 3, 9, 0, 1),
    (2, 7, 9, 3, 8, 0, 6, 4, 1, 5),
    (7, 0, 4, 6, 9, 1, 3, 2, 5, 8),
)

_INV = (0, 4, 3, 2, 1, 5, 6, 7, 8, 9)

_ASCII_DIGITS_RE = re.compile(r"^[0-9]+$")
_SEPARATORS_RE = re.compile(r"[\s-]+")


def _normalize(number: str) -> str:
    """Remove only whitespace and hyphens, preserving every other character."""
    return _SEPARATORS_RE.sub("", number)


def verhoeff_checksum(number: str) -> int:
    """Return check digit for ``number`` (which excludes the digit)."""
    if not isinstance(number, str):
        raise ValueError("number must be a string of ASCII digits")

    normalized = _normalize(number)
    if not normalized or _ASCII_DIGITS_RE.fullmatch(normalized) is None:
        raise ValueError("number must contain only ASCII digits")

    checksum = 0
    for index, char in enumerate(reversed(normalized)):
        checksum = _D[checksum][_P[(index + 1) % 8][int(char)]]
    return _INV[checksum]


def is_valid_verhoeff(number: str) -> bool:
    """True iff ``number`` (digits incl. trailing check) is valid."""
    if not isinstance(number, str):
        return False

    normalized = _normalize(number)
    if not normalized or _ASCII_DIGITS_RE.fullmatch(normalized) is None:
        return False

    checksum = 0
    for index, char in enumerate(reversed(normalized)):
        checksum = _D[checksum][_P[index % 8][int(char)]]
    return checksum == 0


def is_valid_aadhaar(digits: str) -> bool:
    """Validate a 12-digit Aadhaar number and its Verhoeff check digit.

    Printed grouping (``1234 5678 9012`` or ``1234-5678-9012``) is accepted.
    Aadhaar numbers never start with 0 or 1, per UIDAI rules.
    """
    if not isinstance(digits, str):
        return False

    normalized = _normalize(digits)
    if len(normalized) != 12 or normalized[0] in "01":
        return False
    return is_valid_verhoeff(normalized)
