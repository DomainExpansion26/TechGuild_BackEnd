"""Tests for PAN and Aadhaar text extraction."""

from __future__ import annotations

import pytest

from app.validators.id_formats import find_aadhaar_in_text, is_valid_pan


@pytest.mark.parametrize(
    "pan",
    [
        "ABCPE1234F",
        "AAACA1234A",
        "abcpe1234f",
        "ABCPE 1234 F",
        "ABCPE-1234-F",
    ],
)
def test_valid_pan(pan: str) -> None:
    assert is_valid_pan(pan) is True


@pytest.mark.parametrize(
    "pan",
    [
        "ABCDE1234F",
        "ABCPE1234",
        "ABCPE12345",
        "AB1PE1234F",
        "",
        None,
    ],
)
def test_invalid_pan(pan: str | None) -> None:
    assert is_valid_pan(pan) is False


@pytest.mark.parametrize(
    ("text", "expected"),
    [
        ("Aadhaar No: 2345 6789 0124 issued", "234567890124"),
        ("2345-6789-0124", "234567890124"),
        (
            "old 2345 6789 0123 new 2345 6789 0124",
            "234567890124",
        ),
        ("2003\n2345 6789 0124", "234567890124"),
        ("Patel\n16/05/2003\n2345 6789 0124", "234567890124"),
        ("PAN: ABCPE1234F", None),
        ("VID 1234 5678 9012 3456", None),
        ("", None),
        (None, None),
    ],
)
def test_find_aadhaar_in_text(text: str | None, expected: str | None) -> None:
    assert find_aadhaar_in_text(text) == expected


def test_find_aadhaar_across_lines() -> None:
    assert find_aadhaar_in_text("Patel\n16/05/2003\n2345 6789 0124") == (
        "234567890124"
    )


def test_find_aadhaar_not_merged_with_prior_digits() -> None:
    # A line ending in digits must not glue onto the Aadhaar run below it.
    assert find_aadhaar_in_text("2003\n2345 6789 0124") == "234567890124"
