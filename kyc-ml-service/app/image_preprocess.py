"""Card extraction + Rotation/Mirror Fix for OCR."""
from __future__ import annotations
import logging
import os
from typing import Any

logger = logging.getLogger(__name__)

try:
    import cv2
    HAS_CV2 = True
except ImportError:
    cv2 = None
    HAS_CV2 = False

try:
    import numpy as np
    HAS_NUMPY = True
except ImportError:
    np = None
    HAS_NUMPY = False

try:
    import onnxruntime as ort
    HAS_ORT = True
except ImportError:
    HAS_ORT = False

CARD_WIDTH = 1000
CARD_HEIGHT = 630
MODEL_PATH = "/app/models/yolov8_doc.onnx"

STRATEGY_CARD_EXTRACTED = "yolo_card_extracted"
STRATEGY_FULL_IMAGE = "full_image"
STRATEGY_ROTATED = "auto_rotated_fallback"

_yolo_session = None

def _get_yolo_session():
    global _yolo_session
    if _yolo_session is None and HAS_ORT and os.path.exists(MODEL_PATH):
        try:
            _yolo_session = ort.InferenceSession(MODEL_PATH, providers=['CPUExecutionProvider'])
        except Exception as e:
            logger.error(f"Failed to load YOLO model: {e}")
    return _yolo_session

def _result(images: list[Any], strategy: str, card_detected: bool) -> dict[str, Any]:
    return {"images": images, "strategy": strategy, "card_detected": card_detected}

def _order_points(points: Any) -> Any:
    rect = np.zeros((4, 2), dtype=np.float32)
    sums = points.sum(axis=1)
    rect[0] = points[np.argmin(sums)]
    rect[2] = points[np.argmax(sums)]
    diffs = np.diff(points, axis=1)
    rect[1] = points[np.argmin(diffs)]
    rect[3] = points[np.argmax(diffs)]
    return rect

def _distance(first: Any, second: Any) -> float:
    return float(np.linalg.norm(first.astype(np.float32) - second.astype(np.float32)))

def detect_card_region_yolo(image: Any) -> Any:
    session = _get_yolo_session()
    if session is None:
        return None
    try:
        h_orig, w_orig = image.shape[:2]
        img_resized = cv2.resize(image, (640, 640))
        img_rgb = cv2.cvtColor(img_resized, cv2.COLOR_BGR2RGB)
        img_norm = img_rgb.astype(np.float32) / 255.0
        img_tensor = np.transpose(img_norm, (2, 0, 1))
        img_tensor = np.expand_dims(img_tensor, axis=0)
        
        inputs = {session.get_inputs()[0].name: img_tensor}
        outputs = session.run(None, inputs)
        output = outputs[0][0]
        
        if output.shape[0] < output.shape[1]:
            scores = output[4:, :]
            max_scores = np.max(scores, axis=0)
            best_idx = np.argmax(max_scores)
            confidence = max_scores[best_idx]
            xc, yc, w, h = output[:4, best_idx]
        else:
            scores = output[:, 4:]
            max_scores = np.max(scores, axis=1)
            best_idx = np.argmax(max_scores)
            confidence = max_scores[best_idx]
            xc, yc, w, h = output[best_idx, :4]

        if confidence < 0.15: 
            return None
            
        x1 = max(0, (xc - w/2) * (w_orig / 640.0))
        y1 = max(0, (yc - h/2) * (h_orig / 640.0))
        x2 = min(w_orig, (xc + w/2) * (w_orig / 640.0))
        y2 = min(h_orig, (yc + h/2) * (h_orig / 640.0))
        
        pts = np.array([[x1, y1], [x2, y1], [x2, y2], [x1, y2]], dtype=np.float32)
        points = _order_points(pts)
        
        destination = np.float32([[0, 0], [CARD_WIDTH, 0], [CARD_WIDTH, CARD_HEIGHT], [0, CARD_HEIGHT]])
        matrix = cv2.getPerspectiveTransform(points, destination)
        return cv2.warpPerspective(image, matrix, (CARD_WIDTH, CARD_HEIGHT))
    except Exception:
        return None

def enhance_document(image: Any) -> Any:
    if not (HAS_CV2 and HAS_NUMPY) or image is None:
        return image
    try:
        grayscale = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
        clahe = cv2.createCLAHE(clipLimit=3.0, tileGridSize=(8, 8)).apply(grayscale)
        blurred = cv2.GaussianBlur(clahe, (0, 0), 3)
        sharpened = cv2.addWeighted(clahe, 1.5, blurred, -0.5, 0)
        return cv2.cvtColor(sharpened, cv2.COLOR_GRAY2BGR)
    except Exception:
        return image

def preprocess_selfie_image(image: Any) -> dict[str, Any]:
    try:
        if image is None:
            return _result([], "decode_failed", False)
        if not (HAS_CV2 and HAS_NUMPY):
            return _result([image], STRATEGY_FULL_IMAGE, False)

        card = detect_card_region_yolo(image)
        if card is not None:
            return _result([enhance_document(card)], STRATEGY_CARD_EXTRACTED, True)

        h, w = image.shape[:2]
        bottom_crop = image[int(h*0.4):h, :]
        enhanced_bottom = enhance_document(bottom_crop)
        enhanced_full = enhance_document(image)

        img_180 = cv2.rotate(enhanced_full, cv2.ROTATE_180)
        img_flip = cv2.flip(enhanced_full, 1)
        bottom_180 = cv2.rotate(enhanced_bottom, cv2.ROTATE_180)
        bottom_flip = cv2.flip(enhanced_bottom, 1)

        images_to_try = [enhanced_full, enhanced_bottom, img_180, img_flip, bottom_180, bottom_flip]
        return _result(images_to_try, STRATEGY_ROTATED, False)
    except Exception:
        return _result([image], STRATEGY_FULL_IMAGE, False)

def preprocess_selfie_for_ocr(image_bytes: bytes) -> dict[str, Any]:
    if not image_bytes or not (HAS_CV2 and HAS_NUMPY):
        return _result([], "decode_failed", False)
    try:
        buffer = np.frombuffer(image_bytes, dtype=np.uint8)
        image = cv2.imdecode(buffer, cv2.IMREAD_COLOR)
    except Exception:
        image = None
    if image is None:
        return _result([], "decode_failed", False)
    return preprocess_selfie_image(image)
