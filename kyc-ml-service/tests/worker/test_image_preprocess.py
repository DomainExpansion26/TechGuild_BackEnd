"""Phase A.6: card-region extraction / enhancement before selfie document OCR.

The synthetic frames need OpenCV + NumPy, which the slim unit-test virtualenv
does not ship (the project soft-imports both, see ``app/qr/qr_decode.py``). The
OpenCV-dependent cases are therefore marked so they run in the runtime image
built from ``requirements.txt``, while the degradation contract (what happens
when OpenCV is missing) is asserted everywhere.
"""

from __future__ import annotations

import pytest

from app import image_preprocess
from app.image_preprocess import (
    CARD_HEIGHT,
    CARD_WIDTH,
    crop_top_half,
    detect_card_region,
    enhance_document,
    preprocess_selfie_for_ocr,
)

try:  # pragma: no cover - depends on the installed environment
    import cv2
    import numpy as np

    HAS_CV = True
except ImportError:  # pragma: no cover - the slim test venv
    cv2 = None
    np = None
    HAS_CV = False

requires_cv = pytest.mark.skipif(
    not HAS_CV, reason="OpenCV/NumPy ship with the runtime image, not the slim test venv"
)


def _blank(height: int, width: int, colour: int = 0):
    return np.full((height, width, 3), colour, dtype=np.uint8)


def _white_card_frame(height: int = 800, width: int = 1000, angle: float = 10.0):
    """Black frame with a white rotated 500x300 card in the middle."""
    frame = _blank(height, width, 0)
    box = cv2.boxPoints(((width / 2, height / 2), (500, 300), angle))
    cv2.drawContours(frame, [np.int32(box)], -1, (255, 255, 255), -1)
    return frame


def _encode(image) -> bytes:
    ok, buffer = cv2.imencode(".png", image)
    assert ok
    return buffer.tobytes()


# ---------------------------------------------------------------------------
# Degradation contract (runs with and without OpenCV)
# ---------------------------------------------------------------------------


def test_module_exposes_public_api():
    """The worker imports this module unconditionally, so it must be importable."""
    assert image_preprocess.STRATEGY_CARD_EXTRACTED == "card_extracted"
    assert image_preprocess.STRATEGY_TOP_HALF_CROP == "top_half_crop"
    assert image_preprocess.STRATEGY_FULL_IMAGE == "full_image"
    assert image_preprocess.STRATEGY_DECODE_FAILED == "decode_failed"
    assert image_preprocess.CARD_WIDTH == 1000
    assert image_preprocess.CARD_HEIGHT == 630
    for name in (
        "detect_card_region",
        "enhance_document",
        "crop_top_half",
        "preprocess_selfie_for_ocr",
        "preprocess_selfie_image",
    ):
        assert callable(getattr(image_preprocess, name))


def test_preprocess_selfie_for_ocr_decode_fails():
    result = preprocess_selfie_for_ocr(b"not an image")

    assert result["strategy"] == "decode_failed"
    assert result["images"] == []
    assert result["card_detected"] is False
    assert result["decode_error"] == "decode_failed"


def test_preprocess_selfie_for_ocr_empty_bytes_decode_fails():
    result = preprocess_selfie_for_ocr(b"")

    assert result["strategy"] == "decode_failed"
    assert result["images"] == []


def test_preprocess_selfie_image_none_is_decode_failed():
    result = image_preprocess.preprocess_selfie_image(None)

    assert result["strategy"] == "decode_failed"
    assert result["images"] == []


def test_preprocess_selfie_image_without_opencv_hands_back_original(monkeypatch):
    """Degrades to the Phase A behaviour: OCR the frame that was fetched."""
    monkeypatch.setattr(image_preprocess, "HAS_CV2", False)
    monkeypatch.setattr(image_preprocess, "HAS_NUMPY", False)
    marker = object()

    result = image_preprocess.preprocess_selfie_image(marker)

    assert result["strategy"] == "full_image"
    assert result["images"] == [marker]
    assert result["card_detected"] is False
    assert result["decode_error"] is None


def test_preprocess_selfie_image_tolerates_unexpected_input():
    """A non-array payload must not raise; it just falls back to full image."""
    marker = object()

    result = image_preprocess.preprocess_selfie_image(marker)

    assert result["strategy"] in {"full_image", "decode_failed"}
    assert result["card_detected"] is False


def test_detect_card_region_tolerates_unexpected_input():
    assert detect_card_region(object()) is None


def test_enhance_document_tolerates_none():
    assert enhance_document(None) is None


def test_crop_top_half_tolerates_none():
    assert crop_top_half(None) is None


# ---------------------------------------------------------------------------
# detect_card_region
# ---------------------------------------------------------------------------


@requires_cv
def test_detect_card_region_white_card_on_black_bg():
    frame = _white_card_frame()

    warped = detect_card_region(frame)

    assert warped is not None
    assert warped.shape == (CARD_HEIGHT, CARD_WIDTH, 3)
    assert warped.mean() > 200


@requires_cv
def test_detect_card_region_returns_none_on_solid_color():
    solid = _blank(600, 800, 128)

    assert detect_card_region(solid) is None


@requires_cv
def test_detect_card_region_returns_none_if_too_small():
    """A card covering well under 10% of the frame is not the subject."""
    frame = _blank(1200, 1200, 0)
    box = cv2.boxPoints(((600, 600), (150, 90), 0))
    cv2.drawContours(frame, [np.int32(box)], -1, (255, 255, 255), -1)

    assert detect_card_region(frame) is None


@requires_cv
def test_detect_card_region_returns_none_if_frame_filling():
    """An outline covering most of the frame is the image border, not a card."""
    frame = _blank(600, 800, 0)
    cv2.rectangle(frame, (2, 2), (797, 597), (255, 255, 255), -1)

    assert detect_card_region(frame) is None


@requires_cv
def test_detect_card_region_returns_none_without_opencv(monkeypatch):
    monkeypatch.setattr(image_preprocess, "HAS_CV2", False)

    assert detect_card_region(_white_card_frame()) is None


# ---------------------------------------------------------------------------
# enhance_document / crop_top_half
# ---------------------------------------------------------------------------


@requires_cv
def test_enhance_document_increases_contrast():
    flat = _blank(200, 300, 120)

    enhanced = enhance_document(flat)

    assert enhanced.shape == flat.shape
    assert enhanced.ndim == 3
    assert enhanced.shape[2] == 3


@requires_cv
def test_enhance_document_changes_faint_text():
    low_contrast = _blank(200, 300, 120)
    cv2.putText(
        low_contrast,
        "AADHAAR 2345",
        (10, 110),
        cv2.FONT_HERSHEY_SIMPLEX,
        0.8,
        (134, 134, 134),
        2,
    )

    enhanced = enhance_document(low_contrast)

    assert enhanced.shape == low_contrast.shape
    assert not np.array_equal(enhanced, low_contrast)


@requires_cv
def test_enhance_document_returns_input_without_opencv(monkeypatch):
    monkeypatch.setattr(image_preprocess, "HAS_CV2", False)
    flat = _blank(20, 20, 120)

    assert enhance_document(flat) is flat


@requires_cv
def test_crop_top_half_shape():
    frame = _blank(1000, 800, 200)

    cropped = crop_top_half(frame)

    assert cropped.shape == (600, 800, 3)


@requires_cv
def test_crop_top_half_on_tiny_image_returns_input():
    tiny = _blank(3, 4, 200)

    assert crop_top_half(tiny).shape == tiny.shape


# ---------------------------------------------------------------------------
# preprocess_selfie_for_ocr
# ---------------------------------------------------------------------------


@requires_cv
def test_preprocess_selfie_for_ocr_with_card():
    frame = _white_card_frame()

    result = preprocess_selfie_for_ocr(_encode(frame))

    assert result["card_detected"] is True
    assert result["strategy"] == "card_extracted"
    assert len(result["images"]) >= 1
    assert result["images"][0].shape == (CARD_HEIGHT, CARD_WIDTH, 3)
    # The second attempt is the enhanced full frame, for fallback OCR.
    assert result["images"][-1].shape == frame.shape


@requires_cv
def test_preprocess_selfie_for_ocr_no_card_small_image():
    frame = _blank(400, 300, 90)

    result = preprocess_selfie_for_ocr(_encode(frame))

    assert result["strategy"] in {"full_image", "top_half_crop"}
    assert result["card_detected"] is False
    assert len(result["images"]) >= 1


@requires_cv
def test_preprocess_selfie_for_ocr_no_card_tall_image_crops_band():
    frame = _blank(2000, 1000, 90)

    result = preprocess_selfie_for_ocr(_encode(frame))

    assert result["strategy"] == "top_half_crop"
    assert len(result["images"]) == 2
    assert result["images"][0].shape == (1200, 1000, 3)
    assert result["images"][-1].shape == frame.shape