"""Tests for image-to-QR-text decoding in :mod:`qr_decode`."""

from __future__ import annotations

from io import BytesIO

import pytest
from PIL import Image

from app.qr.aadhaar_qr import QRVerificationError
from app.qr.qr_decode import MAX_IMAGE_BYTES, decode_qr_text

# A long numeric payload with the same shape as an Aadhaar Secure QR string.
QR_PAYLOAD = "1234567890" * 200


def _encode_qr_png(text: str) -> bytes:
    """Encode ``text`` as a QR PNG using whatever encoder is available."""
    image: Image.Image | None = None

    try:  # Preferred when OpenCV is installed.
        import cv2

        encoder = cv2.QRCodeEncoder.create()
        array = encoder.encode(text)
        image = Image.fromarray(array)
    except Exception:  # noqa: BLE001 - try the next encoder
        image = None

    if image is None:
        try:
            import qrcode

            qr = qrcode.QRCode(border=4)
            qr.add_data(text)
            qr.make(fit=True)
            image = qr.make_image(fill_color="black", back_color="white").get_image()
        except Exception:  # noqa: BLE001 - try the next encoder
            image = None

    if image is None:
        try:
            import zxingcpp

            barcode = zxingcpp.create_barcode(
                text, zxingcpp.BarcodeFormat.QRCode
            )
            bitmap = zxingcpp.write_barcode_to_image(
                barcode, scale=4, add_quiet_zones=True
            )
            height, width = bitmap.shape
            image = Image.frombytes(
                "L", (width, height), memoryview(bitmap).tobytes()
            )
        except Exception:  # noqa: BLE001 - no encoder available
            image = None

    if image is None:
        pytest.skip("no QR encoder available (cv2, qrcode, zxing-cpp)")

    buffer = BytesIO()
    image.save(buffer, "PNG")
    return buffer.getvalue()


def test_decodes_synthetic_qr_image() -> None:
    png_bytes = _encode_qr_png(QR_PAYLOAD)

    text, engine = decode_qr_text(png_bytes)

    assert text == QR_PAYLOAD
    assert len(text) > 1000
    assert text.isdigit()
    assert engine.startswith(("zxing:", "opencv"))


def test_empty_bytes_rejected() -> None:
    with pytest.raises(QRVerificationError) as exc_info:
        decode_qr_text(b"")

    assert exc_info.value.code == "qr_image_invalid"


def test_too_large_rejected() -> None:
    oversized = b"\x00" * (MAX_IMAGE_BYTES + 1)

    with pytest.raises(QRVerificationError) as exc_info:
        decode_qr_text(oversized)

    assert exc_info.value.code == "qr_image_invalid"


def test_non_image_bytes_rejected() -> None:
    with pytest.raises(QRVerificationError) as exc_info:
        decode_qr_text(b"this is definitely not an image")

    assert exc_info.value.code == "qr_image_invalid"
