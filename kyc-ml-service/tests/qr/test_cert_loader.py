"""Tests for :mod:`validators.cert_loader`.

All certificates are synthetic RSA-2048 (or EC for the non-RSA case) self-signed
certs generated at test time. The real UIDAI certificate is not required.
"""

from __future__ import annotations

import hashlib
from datetime import datetime, timedelta, timezone

import pytest
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec, rsa
from cryptography.x509.oid import NameOID

from app.qr.cert_loader import TrustedCert, load_pinned_certs

COMMON_NAME = "Synthetic UIDAI Test Cert"


def _build_cert(key=None) -> tuple[x509.Certificate, bytes]:
    """Build a self-signed certificate and return ``(cert, der_bytes)``."""
    private_key = key or rsa.generate_private_key(
        public_exponent=65537, key_size=2048
    )
    now = datetime.now(timezone.utc)
    subject = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, COMMON_NAME)])
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
    return cert, cert.public_bytes(serialization.Encoding.DER)


def _write_sums(cert_dir, lines: str) -> None:
    (cert_dir / "SHA256SUMS").write_text(lines, encoding="utf-8")


def _sums_line(der: bytes, filename: str) -> str:
    return f"{hashlib.sha256(der).hexdigest()}  {filename}\n"


def test_empty_dir_raises(tmp_path):
    with pytest.raises(ValueError):
        load_pinned_certs(tmp_path)


def test_missing_sums_raises(tmp_path):
    _cert, der = _build_cert()
    (tmp_path / "synthetic-uidai.cer").write_bytes(der)

    with pytest.raises(ValueError):
        load_pinned_certs(tmp_path)


def test_hash_mismatch_raises(tmp_path):
    _cert, der = _build_cert()
    name = "synthetic-uidai.cer"
    (tmp_path / name).write_bytes(der)
    _write_sums(tmp_path, f"{'0' * 64}  {name}\n")

    with pytest.raises(ValueError):
        load_pinned_certs(tmp_path)


def test_non_rsa_cert_raises(tmp_path):
    ec_key = ec.generate_private_key(ec.SECP256R1())
    _cert, der = _build_cert(key=ec_key)
    name = "synthetic-ec.cer"
    (tmp_path / name).write_bytes(der)
    _write_sums(tmp_path, _sums_line(der, name))

    with pytest.raises(ValueError):
        load_pinned_certs(tmp_path)


def test_cert_missing_from_sums_raises(tmp_path):
    _cert, der = _build_cert()
    name = "synthetic-uidai.cer"
    (tmp_path / name).write_bytes(der)
    # Manifest exists but only lists a different file.
    _write_sums(tmp_path, _sums_line(der, "different.cer"))

    with pytest.raises(ValueError):
        load_pinned_certs(tmp_path)


def test_valid_single_cert_loads_returns_trustedcert(tmp_path):
    cert, der = _build_cert()
    name = "synthetic-uidai.cer"
    (tmp_path / name).write_bytes(der)
    _write_sums(tmp_path, _sums_line(der, name))

    loaded = load_pinned_certs(tmp_path)

    assert len(loaded) == 1
    trusted = loaded[0]
    assert isinstance(trusted, TrustedCert)
    assert trusted.name == name
    assert trusted.fingerprint_sha256 == hashlib.sha256(der).hexdigest()
    assert trusted.not_valid_before_utc.tzinfo is not None
    assert trusted.not_valid_after_utc.tzinfo is not None
    assert trusted.not_valid_before_utc < trusted.not_valid_after_utc
    assert trusted.cert.serial_number == cert.serial_number
    assert (
        trusted.cert.subject.get_attributes_for_oid(NameOID.COMMON_NAME)[0].value
        == COMMON_NAME
    )
