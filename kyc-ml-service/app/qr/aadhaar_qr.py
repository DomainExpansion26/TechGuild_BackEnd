"""Aadhaar Secure QR decoding, signature verification, and V2 parsing.

This module is intentionally standalone: it takes the numeric QR payload as
text plus a list of pinned certificates and returns verified demographics and
the embedded photo bytes. It does not touch the database, the worker, or the
Go backend.

Payload envelope (V2)::

    decimal digits
      -> minimal big-endian integer bytes
      -> zlib stream
      -> b"V2" 0xFF <17 text fields separated by 0xFF> 0xFF <photo> <256-byte RSA signature>

The signature is PKCS#1 v1.5 SHA-256 over everything except the trailing 256
bytes.
"""

from __future__ import annotations

import logging
import re
import sys
import zlib
from dataclasses import dataclass

from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import padding

from app.qr.cert_loader import TrustedCert

# Python 3.11+ caps int(str) at 4300 digits by default.
# Aadhaar QR payloads are 5000-15000 digits.
sys.set_int_max_str_digits(20_000)

logger = logging.getLogger(__name__)

MAX_QR_TEXT_DIGITS = 20_000
MAX_DECOMPRESSED_BYTES = 512 * 1024
RSA_SIGNATURE_BYTES = 256

_JPEG_MAGIC = b"\xff\xd8\xff"
_JPEG2000_MAGIC = b"\x00\x00\x00\x0c\x6a\x50\x20\x20\x0d\x0a\x87\x0a"

_ASCII_DIGITS_RE = re.compile(r"^[0-9]+$")

# Order matters: this is the on-card V2 field order.
_FIELD_NAMES: tuple[str, ...] = (
    "email_mobile_status",
    "reference_id",
    "name",
    "dob",
    "gender",
    "careof",
    "district",
    "landmark",
    "house",
    "location",
    "pincode",
    "postoffice",
    "state",
    "street",
    "subdistrict",
    "vtc",
    "phone_last4",
)
_FIELD_COUNT = len(_FIELD_NAMES)


class QRVerificationError(Exception):
    """Fail-closed QR error. ``code`` is a machine-readable reason."""

    def __init__(self, code: str) -> None:
        self.code = code
        super().__init__(code)


@dataclass(frozen=True)
class QRResult:
    """Verified contents of an Aadhaar Secure QR code."""

    signature_valid: bool
    matched_cert_name: str | None
    matched_cert_fingerprint: str | None
    qr_version: str
    fields: dict[str, str]
    photo_bytes: bytes
    photo_format: str
    qr_reference_last4: str | None


def decode_envelope(qr_text: str) -> bytes:
    """Turn the numeric QR text into the raw decompressed payload bytes."""
    if not isinstance(qr_text, str):
        raise QRVerificationError("qr_payload_not_numeric")

    text = qr_text.strip()
    if not text:
        raise QRVerificationError("qr_payload_empty")
    if not _ASCII_DIGITS_RE.fullmatch(text):
        raise QRVerificationError("qr_payload_not_numeric")
    if len(text) > MAX_QR_TEXT_DIGITS:
        raise QRVerificationError("qr_payload_too_long")

    try:
        number = int(text)
    except ValueError as exc:  # pragma: no cover - guarded by the regex
        raise QRVerificationError("qr_payload_not_numeric") from exc

    if number <= 0:
        raise QRVerificationError("qr_payload_empty")

    buf = number.to_bytes((number.bit_length() + 7) // 8, "big")

    try:
        decompressor = zlib.decompressobj()
        raw = decompressor.decompress(buf, MAX_DECOMPRESSED_BYTES)
    except zlib.error as exc:
        raise QRVerificationError("qr_decompress_failed") from exc

    if decompressor.unconsumed_tail or not decompressor.eof:
        raise QRVerificationError("qr_decompressed_too_large")

    return raw


def verify_signature(
    raw: bytes, certs: list[TrustedCert]
) -> tuple[TrustedCert, bytes]:
    """Verify ``raw`` against each pinned cert; return the matching cert."""
    if len(raw) <= RSA_SIGNATURE_BYTES + 18:
        raise QRVerificationError("qr_payload_too_short")

    signed = raw[:-RSA_SIGNATURE_BYTES]
    signature = raw[-RSA_SIGNATURE_BYTES:]

    for trusted in certs:
        public_key = trusted.cert.public_key()
        try:
            public_key.verify(
                signature,
                signed,
                padding.PKCS1v15(),
                hashes.SHA256(),
            )
        except InvalidSignature:
            continue
        logger.debug("qr signature verified cert=%s", trusted.name)
        return trusted, signed

    raise QRVerificationError("qr_signature_invalid")


def parse_signed_data(signed: bytes) -> dict:
    """Parse strict V2 signed data into fields plus the embedded photo."""
    first_field = signed.split(b"\xff", 1)[0]
    if first_field != b"V2":
        raise QRVerificationError("qr_unknown_version")

    # There are 18 separators for 17 fields + "V2" + photo: one after V2, 16
    # between fields, and one before the photo. maxsplit=18 (FIELD_COUNT + 1)
    # keeps the trailing photo (which may contain 0xFF bytes in JPEG headers
    # and entropy data) as a single final part.
    parts = signed.split(b"\xff", _FIELD_COUNT + 1)
    if len(parts) != _FIELD_COUNT + 2:
        raise QRVerificationError("qr_fields_truncated")

    values = [
        parts[index].decode("utf-8", errors="replace").strip()
        for index in range(1, _FIELD_COUNT + 1)
    ]
    fields = dict(zip(_FIELD_NAMES, values))

    photo_bytes = parts[_FIELD_COUNT + 1]
    if not photo_bytes:
        raise QRVerificationError("qr_photo_missing")

    if photo_bytes.startswith(_JPEG_MAGIC):
        photo_format = "jpeg"
    elif photo_bytes.startswith(_JPEG2000_MAGIC):
        photo_format = "jpeg2000"
    else:
        raise QRVerificationError("qr_photo_invalid_format")

    logger.debug(
        "qr parsed fields=%d photo_format=%s", _FIELD_COUNT, photo_format
    )

    return {
        "version": "V2",
        **fields,
        "photo_bytes": photo_bytes,
        "photo_format": photo_format,
    }


def verify_aadhaar_qr(qr_text: str, certs: list[TrustedCert]) -> QRResult:
    """Decode, verify, and parse an Aadhaar Secure QR payload end to end."""
    raw = decode_envelope(qr_text)
    trusted, signed = verify_signature(raw, certs)

    try:
        parsed = parse_signed_data(signed)
    except QRVerificationError as exc:
        logger.debug("qr parse failed code=%s", exc.code)
        raise

    fields = {name: parsed[name] for name in _FIELD_NAMES}
    reference_id = fields.get("reference_id") or ""
    reference_last4 = reference_id[:4] if reference_id else None

    return QRResult(
        signature_valid=True,
        matched_cert_name=trusted.name,
        matched_cert_fingerprint=trusted.fingerprint_sha256,
        qr_version=parsed["version"],
        fields=fields,
        photo_bytes=parsed["photo_bytes"],
        photo_format=parsed["photo_format"],
        qr_reference_last4=reference_last4,
    )
