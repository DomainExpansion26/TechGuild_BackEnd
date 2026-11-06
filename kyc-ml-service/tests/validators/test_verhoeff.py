"""Tests for the Verhoeff checksum and Aadhaar validation."""

from __future__ import annotations

import pytest

from app.validators.verhoeff import (
    is_valid_aadhaar,
    is_valid_verhoeff,
    verhoeff_checksum,
)


@pytest.mark.parametrize(
    "number",
    [
        "234567890124",
        "999999999999",
        "2345 6789 0124",
        "2345-6789-0124",
    ],
)
def test_valid_aadhaar_numbers(number: str) -> None:
    assert is_valid_aadhaar(number) is True


@pytest.mark.parametrize(
    "number",
    [
        "234567890123",
        "234567890125",
        "23456789012",
        "2345678901245",
        "123456789012",
        "023456789012",
        "abcdefghijkl",
        "१२३४५६७८९०१२",
        "",
        None,
    ],
)
def test_invalid_aadhaar_numbers(number: str | None) -> None:
    assert is_valid_aadhaar(number) is False


@pytest.mark.parametrize(
    "base",
    [
        "23456789012",
        "98765432109",
        "11223344556",
    ],
)
def test_verhoeff_checksum_roundtrip(base: str) -> None:
    check = verhoeff_checksum(base)

    assert 0 <= check <= 9
    assert is_valid_verhoeff(base + str(check)) is True
    assert is_valid_verhoeff(base + str((check + 1) % 10)) is False


def test_verhoeff_checksum_rejects_non_ascii_digits() -> None:
    with pytest.raises(ValueError):
        verhoeff_checksum("12345abc")


def test_verhoeff_checksum_rejects_unicode_digits() -> None:
    with pytest.raises(ValueError):
        verhoeff_checksum("१२३४५")
