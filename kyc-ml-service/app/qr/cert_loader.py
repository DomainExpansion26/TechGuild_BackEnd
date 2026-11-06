"""Pinned certificate loading for Aadhaar Secure QR verification.

The Aadhaar Secure QR payload is signed with UIDAI's RSA-2048 key. Only
certificates explicitly pinned in a local directory (matching a SHA256SUMS
manifest) are trusted. Certificates are never fetched at runtime.

Expiry is deliberately NOT used as a gate: a card printed in 2019 was signed
with a certificate that may have expired since, but the signature over the
card data is still cryptographically valid. Expiry is recorded as data so the
caller can decide whether it matters.
"""

from __future__ import annotations

import hashlib
import hmac
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

from cryptography import x509
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.primitives.serialization import Encoding


@dataclass(frozen=True)
class TrustedCert:
    """A pinned, cryptographically trusted Aadhaar QR signing certificate."""

    name: str
    cert: x509.Certificate
    fingerprint_sha256: str
    not_valid_before_utc: datetime
    not_valid_after_utc: datetime


def _read_sha256sums(path: Path) -> dict[str, str]:
    """Parse a ``sha256sum``-style manifest into ``{filename: lowercase_hex}``."""
    sums: dict[str, str] = {}
    if not path.is_file():
        return sums

    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split(None, 1)
        if len(parts) != 2:
            continue
        digest, filename = parts
        filename = filename.strip().lstrip("*").strip()
        if not filename:
            continue
        sums[filename] = digest.strip().lower()
    return sums


def _utc_validity_bounds(cert: x509.Certificate) -> tuple[datetime, datetime]:
    """Return ``(not_valid_before, not_valid_after)`` as timezone-aware UTC."""
    before = getattr(cert, "not_valid_before_utc", None)
    after = getattr(cert, "not_valid_after_utc", None)
    if before is None or after is None:
        # cryptography < 42 fallback. The deprecated attributes are naive UTC.
        before = cert.not_valid_before.replace(tzinfo=timezone.utc)
        after = cert.not_valid_after.replace(tzinfo=timezone.utc)
    return before, after


def load_pinned_certs(cert_dir: Path) -> list[TrustedCert]:
    """Load every ``*.cer`` in ``cert_dir`` and verify it against SHA256SUMS.

    Raises ``ValueError`` when the directory is missing/empty, a certificate
    has no manifest entry, the DER fingerprint does not match the manifest, or
    the certificate's public key is not RSA. Expired certificates are accepted
    and their validity window is recorded in :class:`TrustedCert`.

    The result is sorted by ``not_valid_after_utc`` descending, so the
    most-recently-valid certificate is tried first during verification.
    """
    cert_dir = Path(cert_dir)

    if not cert_dir.is_dir():
        raise ValueError(f"certificate directory does not exist: {cert_dir}")

    cer_files = sorted(cert_dir.glob("*.cer"))
    if not cer_files:
        raise ValueError(f"no .cer files found in {cert_dir}")

    sums = _read_sha256sums(cert_dir / "SHA256SUMS")
    trusted: list[TrustedCert] = []

    for cer_path in cer_files:
        raw = cer_path.read_bytes()

        try:
            cert = x509.load_der_x509_certificate(raw)
        except ValueError:
            cert = x509.load_pem_x509_certificate(raw)

        # Fingerprint the canonical DER encoding, not the file bytes, so a PEM
        # re-encoding of the same certificate yields the same fingerprint.
        canonical_der = cert.public_bytes(Encoding.DER)
        fingerprint = hashlib.sha256(canonical_der).hexdigest()

        expected = sums.get(cer_path.name)
        if expected is None:
            raise ValueError(
                f"{cer_path.name}: no entry in SHA256SUMS"
            )
        if not hmac.compare_digest(fingerprint, expected):
            raise ValueError(
                f"{cer_path.name}: DER fingerprint does not match SHA256SUMS"
            )

        public_key = cert.public_key()
        if not isinstance(public_key, rsa.RSAPublicKey):
            raise ValueError(f"{cer_path.name}: public key is not RSA")

        not_before, not_after = _utc_validity_bounds(cert)
        trusted.append(
            TrustedCert(
                name=cer_path.name,
                cert=cert,
                fingerprint_sha256=fingerprint,
                not_valid_before_utc=not_before,
                not_valid_after_utc=not_after,
            )
        )

    trusted.sort(key=lambda item: item.not_valid_after_utc, reverse=True)
    return trusted
