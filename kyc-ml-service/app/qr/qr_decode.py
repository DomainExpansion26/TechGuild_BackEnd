"""QR image decoding with a zxing-cpp first cascade and optional OpenCV.

This module only turns image bytes into the numeric QR text. It does not
verify or parse the Aadhaar payload; see :mod:`aadhaar_qr` for that.

OpenCV and NumPy are optional. When they are absent the zxing-cpp cascade
still works using PIL images; the cv2-only image variants and fallback are
skipped.
"""

from __future__ import annotations

import io
import logging

import zxingcpp
from PIL import Image, ImageOps, UnidentifiedImageError

from app.qr.aadhaar_qr import QRVerificationError

try:
    import cv2

    HAS_CV2 = True
except ImportError:  # pragma: no cover - exercised only without OpenCV
    cv2 = None  # type: ignore[assignment]
    HAS_CV2 = False

try:
    import numpy as np

    HAS_NUMPY = True
except ImportError:  # pragma: no cover - exercised only without NumPy
    np = None  # type: ignore[assignment]
    HAS_NUMPY = False

logger = logging.getLogger(__name__)

MAX_IMAGE_BYTES = 20_000_000
MAX_LONG_SIDE_PX = 2400


def _load_rgb(image_bytes: bytes) -> Image.Image:
    """Decode arbitrary image bytes into an RGB PIL image."""
    try:
        with Image.open(io.BytesIO(image_bytes)) as source:
            source.load()
            return source.convert("RGB")
    except (
        UnidentifiedImageError,
        OSError,
        ValueError,
        Image.DecompressionBombError,
    ) as exc:
        raise QRVerificationError("qr_image_invalid") from exc


def _cap_long_side(image: Image.Image) -> Image.Image:
    if max(image.size) <= MAX_LONG_SIDE_PX:
        return image
    capped = image.copy()
    capped.thumbnail(
        (MAX_LONG_SIDE_PX, MAX_LONG_SIDE_PX), Image.Resampling.LANCZOS
    )
    return capped


def _rgb_to_bgr(rgb: Image.Image):
    """Convert a PIL RGB image to an OpenCV BGR ndarray."""
    array = np.asarray(rgb, dtype=np.uint8)
    return cv2.cvtColor(array, cv2.COLOR_RGB2BGR)


def _build_variants(
    rgb: Image.Image,
) -> list[tuple[str, object]]:
    """Build decodable image variants, adding cv2-only ones when available."""
    gray = rgb.convert("L")
    variants: list[tuple[str, object]] = [
        ("original", rgb),
        ("grayscale", gray),
        ("autocontrast", ImageOps.autocontrast(gray)),
    ]

    if not (HAS_CV2 and HAS_NUMPY):
        return variants

    bgr = _rgb_to_bgr(rgb)
    gray_array = cv2.cvtColor(bgr, cv2.COLOR_BGR2GRAY)

    clahe = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(8, 8))
    variants.append(("clahe", clahe.apply(gray_array)))

    adaptive = cv2.adaptiveThreshold(
        gray_array,
        255,
        cv2.ADAPTIVE_THRESH_GAUSSIAN_C,
        cv2.THRESH_BINARY,
        31,
        7,
    )
    variants.append(("adaptive", adaptive))
    variants.append(("adaptive_inverted", cv2.bitwise_not(adaptive)))

    upscaled = cv2.resize(
        gray_array,
        None,
        fx=2.0,
        fy=2.0,
        interpolation=cv2.INTER_LANCZOS4,
    )
    variants.append(("gray_2x", upscaled))

    return variants


def decode_qr_text(image_bytes: bytes) -> tuple[str, str]:
    """Decode the first QR code found in ``image_bytes``.

    Returns ``(text, engine_name)``. Raises
    :class:`~aadhaar_qr.QRVerificationError` with code ``qr_image_invalid`` for
    unusable input and ``qr_not_decodable`` when the cascade finds nothing.
    """
    if not image_bytes or len(image_bytes) > MAX_IMAGE_BYTES:
        raise QRVerificationError("qr_image_invalid")

    rgb = _cap_long_side(_load_rgb(image_bytes))

    for variant_name, variant in _build_variants(rgb):
        try:
            results = zxingcpp.read_barcodes(
                variant,
                formats=[zxingcpp.BarcodeFormat.QRCode],
                try_rotate=True,
                try_downscale=True,
                try_invert=True,
            )
        except Exception as exc:  # noqa: BLE001 - one bad variant must not abort
            logger.debug(
                "qr variant=%s zxing_error=%s",
                variant_name,
                type(exc).__name__,
            )
            continue

        for result in results:
            text = getattr(result, "text", "") or ""
            if text:
                logger.debug(
                    "qr decoded variant=%s engine=zxing", variant_name
                )
                return text, f"zxing:{variant_name}"

        logger.debug("qr variant=%s no_result", variant_name)

    if HAS_CV2 and HAS_NUMPY:
        try:
            bgr = _rgb_to_bgr(rgb)
            detector = cv2.QRCodeDetector()
            text, _points, _straight = detector.detectAndDecode(bgr)
        except Exception as exc:  # noqa: BLE001 - optional fallback only
            logger.debug("qr opencv_error=%s", type(exc).__name__)
            text = ""
        if text:
            logger.debug("qr decoded engine=opencv")
            return text, "opencv"

    raise QRVerificationError("qr_not_decodable")
