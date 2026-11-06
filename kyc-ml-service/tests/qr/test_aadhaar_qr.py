"""Tests for Aadhaar Secure QR envelope decoding, verification, and parsing.

All fixtures are synthetic: a fresh RSA-2048 key and self-signed certificate
are generated at test time, and the matching DER is pinned in a temporary
``SHA256SUMS`` file. No real UIDAI certificate is used or required.
"""

from __future__ import annotations

import hashlib
import tempfile
import zlib
from datetime import datetime, timedelta, timezone
from io import BytesIO
from pathlib import Path

import pytest
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import padding, rsa
from cryptography.x509.oid import NameOID
from PIL import Image

from app.qr.aadhaar_qr import (
    MAX_DECOMPRESSED_BYTES,
    QRVerificationError,
    decode_envelope,
    parse_signed_data,
    verify_aadhaar_qr,
    verify_signature,
)
from app.qr.cert_loader import load_pinned_certs

# A minimal 1x1 JPEG. Falls back to PIL if the literal does not validate.
JPEG_BYTES = bytes.fromhex(
    "ffd8ffe000104a46494600010100000100010000"
    "ffdb004300ffffffffffffffffffffffffffffffffffffffffffffffffff"
    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
    "ffffffffffffffffffffffffffffffffffffffffffffffffffc2000b0800010001"
    "01011100ffc40014000100000000000000000000000000000009ffda0008"
    "01010000010000ffd9"
)

JP2_SUPPORTED = ".jp2" in Image.registered_extensions()

FIELD_VALUES: tuple[str, ...] = (
    "0",
    "1234/56789/12345",
    "JOHN DOE",
    "1990-01-01",
    "M",
    "S/O JANE DOE",
    "TEST DISTRICT",
    "NEAR LANDMARK",
    "1-2-3",
    "TEST LOCATION",
    "560001",
    "TEST POST OFFICE",
    "KARNATAKA",
    "TEST STREET",
    "TEST SUBDISTRICT",
    "TEST VTC",
    "9999",
)


def _jpeg_bytes() -> bytes:
    try:
        with Image.open(BytesIO(JPEG_BYTES)) as probe:
            probe.verify()
    except Exception:  # noqa: BLE001 - fixture fallback only
        buffer = BytesIO()
        Image.new("RGB", (1, 1), (255, 255, 255)).save(buffer, "JPEG")
        return buffer.getvalue()
    return JPEG_BYTES


def _jpeg2000_bytes() -> bytes:
    buffer = BytesIO()
    Image.new("RGB", (1, 1), (255, 255, 255)).save(buffer, "JPEG2000")
    return buffer.getvalue()


def _photo_bytes(photo_format: str) -> bytes:
    if photo_format == "jpeg":
        return _jpeg_bytes()
    if photo_format == "jpeg2000":
        return _jpeg2000_bytes()
    raise AssertionError(f"unsupported fixture photo format: {photo_format}")


def _make_synthetic_v2_payload(
    *,
    photo_format: str = "jpeg",
    tamper_signature: bool = False,
    version: bytes = b"V2",
    fields: tuple[str, ...] | None = None,
    tmp_path: Path | None = None,
) -> tuple[str, x509.Certificate, Path]:
    """Build a signed synthetic V2 payload and a pinned cert directory."""
    cert_dir = (
        Path(tmp_path)
        if tmp_path is not None
        else Path(tempfile.mkdtemp(prefix="aadhaar-qr-test-"))
    )
    cert_dir.mkdir(parents=True, exist_ok=True)

    private_key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    now = datetime.now(timezone.utc)
    subject = x509.Name(
        [x509.NameAttribute(NameOID.COMMON_NAME, "Synthetic UIDAI Test Cert")]
    )
    cert = (
        x509.CertificateBuilder()
        .subject_name(subject)
        .issuer_name(subject)
        .public_key(private_key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(now - timedelta(days=1))
        .not_valid_after(now + timedelta(days=3650))
        .add_extension(
            x509.BasicConstraints(ca=False, path_length=None), critical=True
        )
        .sign(private_key, hashes.SHA256())
    )

    der = cert.public_bytes(serialization.Encoding.DER)
    cert_name = "synthetic-uidai.cer"
    (cert_dir / cert_name).write_bytes(der)
    digest = hashlib.sha256(der).hexdigest()
    (cert_dir / "SHA256SUMS").write_text(
        f"{digest}  {cert_name}\n", encoding="utf-8"
    )

    values = fields if fields is not None else FIELD_VALUES
    photo = _photo_bytes(photo_format)
    signed = (
        version
        + b"\xff"
        + b"\xff".join(value.encode("utf-8") for value in values)
        + b"\xff"
        + photo
    )

    signature = private_key.sign(
        signed, padding.PKCS1v15(), hashes.SHA256()
    )
    if tamper_signature:
        signature = bytes([signature[0] ^ 0xFF]) + signature[1:]

    compressed = zlib.compress(signed + signature)
    number = int.from_bytes(compressed, "big")
    return str(number), cert, cert_dir


def _decode_raw(tmp_path: Path, **kwargs) -> tuple[bytes, Path]:
    qr_text, _cert, cert_dir = _make_synthetic_v2_payload(
        tmp_path=tmp_path, **kwargs
    )
    return decode_envelope(qr_text), cert_dir


def test_decode_envelope_roundtrip(tmp_path: Path) -> None:
    raw, _cert_dir = _decode_raw(tmp_path)

    assert raw.startswith(b"V2\xff")
    signed = raw[:-256]
    assert b"JOHN DOE" in signed
    assert signed.endswith(_jpeg_bytes())
    assert len(raw) - len(signed) == 256


@pytest.mark.parametrize("payload", ["12a4", "12 34", "१२३४", "-123", "1.5"])
def test_rejects_non_numeric(payload: str) -> None:
    with pytest.raises(QRVerificationError) as exc_info:
        decode_envelope(payload)

    assert exc_info.value.code == "qr_payload_not_numeric"


@pytest.mark.parametrize("payload", ["", "   ", "\n\t "])
def test_rejects_empty(payload: str) -> None:
    with pytest.raises(QRVerificationError) as exc_info:
        decode_envelope(payload)

    assert exc_info.value.code == "qr_payload_empty"


def test_rejects_too_long() -> None:
    with pytest.raises(QRVerificationError) as exc_info:
        decode_envelope("1" * 20_001)

    assert exc_info.value.code == "qr_payload_too_long"


def test_rejects_zlib_bomb() -> None:
    bomb = zlib.compress(b"\x00" * (MAX_DECOMPRESSED_BYTES + 1024))
    payload = str(int.from_bytes(bomb, "big"))

    with pytest.raises(QRVerificationError) as exc_info:
        decode_envelope(payload)

    assert exc_info.value.code == "qr_decompressed_too_large"


def test_signature_tamper_rejected(tmp_path: Path) -> None:
    raw, cert_dir = _decode_raw(tmp_path, tamper_signature=True)
    certs = load_pinned_certs(cert_dir)

    with pytest.raises(QRVerificationError) as exc_info:
        verify_signature(raw, certs)

    assert exc_info.value.code == "qr_signature_invalid"


def test_unknown_cert_rejected(tmp_path: Path) -> None:
    raw, _cert_dir = _decode_raw(tmp_path / "signed", version=b"V2")
    _other_qr, _other_cert, other_dir = _make_synthetic_v2_payload(
        tmp_path=tmp_path / "other"
    )
    other_certs = load_pinned_certs(other_dir)

    with pytest.raises(QRVerificationError) as exc_info:
        verify_signature(raw, other_certs)

    assert exc_info.value.code == "qr_signature_invalid"


def test_unknown_version_rejected(tmp_path: Path) -> None:
    raw, cert_dir = _decode_raw(tmp_path, version=b"V9")
    certs = load_pinned_certs(cert_dir)
    _trusted, signed = verify_signature(raw, certs)

    with pytest.raises(QRVerificationError) as exc_info:
        parse_signed_data(signed)

    assert exc_info.value.code == "qr_unknown_version"


def test_truncated_fields_rejected() -> None:
    # No 0xFF bytes in the tail so it cannot be mistaken for field separators.
    signed = b"V2\xff" + b"\xff".join(b"x" for _ in range(5)) + b"\xff" + b"photo"

    with pytest.raises(QRVerificationError) as exc_info:
        parse_signed_data(signed)

    assert exc_info.value.code == "qr_fields_truncated"


def _assert_photo_format(
    tmp_path: Path, photo_format: str, magic: bytes
) -> None:
    raw, cert_dir = _decode_raw(tmp_path, photo_format=photo_format)
    certs = load_pinned_certs(cert_dir)
    _trusted, signed = verify_signature(raw, certs)

    parsed = parse_signed_data(signed)

    assert parsed["photo_format"] == photo_format
    assert parsed["photo_bytes"].startswith(magic)


def test_jpeg_photo_detected(tmp_path: Path) -> None:
    _assert_photo_format(tmp_path, "jpeg", b"\xff\xd8\xff")


@pytest.mark.skipif(
    not JP2_SUPPORTED, reason="Pillow has no JPEG2000 support (.jp2)"
)
def test_jpeg2000_photo_detected(tmp_path: Path) -> None:
    _assert_photo_format(
        tmp_path,
        "jpeg2000",
        b"\x00\x00\x00\x0c\x6a\x50\x20\x20\x0d\x0a\x87\x0a",
    )


def test_empty_photo_rejected() -> None:
    signed = (
        b"V2\xff"
        + b"\xff".join(value.encode("utf-8") for value in FIELD_VALUES)
        + b"\xff"
    )

    with pytest.raises(QRVerificationError) as exc_info:
        parse_signed_data(signed)

    assert exc_info.value.code == "qr_photo_missing"


def test_reference_last4_extracted(tmp_path: Path) -> None:
    qr_text, _cert, cert_dir = _make_synthetic_v2_payload(tmp_path=tmp_path)
    certs = load_pinned_certs(cert_dir)

    result = verify_aadhaar_qr(qr_text, certs)

    assert result.qr_reference_last4 == "1234"


def test_verify_aadhaar_qr_end_to_end(tmp_path: Path) -> None:
    qr_text, cert, cert_dir = _make_synthetic_v2_payload(tmp_path=tmp_path)
    certs = load_pinned_certs(cert_dir)
    expected_digest = hashlib.sha256(
        cert.public_bytes(serialization.Encoding.DER)
    ).hexdigest()

    result = verify_aadhaar_qr(qr_text, certs)

    assert result.signature_valid is True
    assert result.matched_cert_name == "synthetic-uidai.cer"
    assert result.matched_cert_fingerprint == expected_digest
    assert result.qr_version == "V2"
    assert result.fields["name"] == "JOHN DOE"
    assert result.fields["pincode"] == "560001"
    assert len(result.fields) == 17
    assert result.photo_format == "jpeg"
    assert result.photo_bytes.startswith(b"\xff\xd8\xff")
    assert result.qr_reference_last4 == "1234"
