"""Phase 5 unit tests: multi-reference face matching (worker.run_face_match).

The SFace embedding function and the YuNet detector are replaced with a
scripted double: each document image carries the Cosine it must produce, so
``run_face_match`` runs end-to-end (fetch -> detect -> quality gate -> embed ->
score) without numpy/cv2/ONNX. Synthetic score maps drive the multi-reference
decision logic; no real images are used.
"""

from __future__ import annotations

import dataclasses
import json
import math

import pytest


# ---------------------------------------------------------------------------
# Import bootstrap (same style as test_worker.py: no heavy deps required).
# ---------------------------------------------------------------------------

import importlib.util
import sys
import types


def _install_stub(name: str, **attrs):
    module = types.ModuleType(name)
    for key, value in attrs.items():
        setattr(module, key, value)
    sys.modules[name] = module
    return module


def _ensure_test_imports() -> None:
    if importlib.util.find_spec("dotenv") is None:
        _install_stub("dotenv", load_dotenv=lambda *args, **kwargs: False)

    if importlib.util.find_spec("redis") is None:
        redis_module = _install_stub("redis")

        class _StubRedis:  # pragma: no cover
            def __init__(self, *args, **kwargs):
                pass

        redis_module.Redis = _StubRedis
        exceptions_module = _install_stub("redis.exceptions")

        class ResponseError(Exception):
            pass

        exceptions_module.ResponseError = ResponseError
        redis_module.exceptions = exceptions_module

    if importlib.util.find_spec("psycopg2") is None:

        def _no_database(*args, **kwargs):
            raise RuntimeError("psycopg2 is stubbed for unit tests")

        _install_stub("psycopg2", connect=_no_database)


_ensure_test_imports()

from app import worker  # noqa: E402  (must come after the dependency stubs)


# ---------------------------------------------------------------------------
# Settings fixture: pin the Phase 5 gates so tests never depend on .env.
# ---------------------------------------------------------------------------


@pytest.fixture(autouse=True)
def stable_settings(monkeypatch):
    test_settings = dataclasses.replace(
        worker.settings,
        ml_mode="mock",
        threshold_face_auto_pass=0.60,
        threshold_face_review=0.40,
        threshold_face_reject=0.35,
        shadow_mode=False,
        face_min_reference_side=80,
        face_min_detection_score=0.80,
        face_min_sharpness=80.0,
        second_reference_divergence_threshold=0.25,
    )
    monkeypatch.setattr(worker, "settings", test_settings)
    return test_settings


# ---------------------------------------------------------------------------
# Scripted doubles
# ---------------------------------------------------------------------------


class _FakeImage:
    """Stands in for a decoded numpy image inside the quality gate."""

    def __init__(self, sharpness: float = 250.0, height: int = 600, width: int = 400):
        self.shape = (height, width, 3)
        self.sharpness = sharpness

    def __getitem__(self, item):
        return self  # the stub cv2 ignores the crop region


class _FakeVar:
    def __init__(self, value: float) -> None:
        self._value = value

    def var(self) -> float:
        return float(self._value)


class _FakeCV2:
    COLOR_BGR2GRAY = 6
    CV_64F = 6

    def cvtColor(self, image, flag):  # noqa: N802 - mirrors cv2 spelling
        return image

    def Laplacian(self, image, dtype):  # noqa: N802 - mirrors cv2 spelling
        return _FakeVar(image.sharpness)


class _FakeNumpy:
    """Minimal np.dot / np.linalg.norm for flat python-vector embeddings."""

    @staticmethod
    def dot(a, b) -> float:
        return float(sum(float(x) * float(y) for x, y in zip(a, b)))

    class linalg:
        @staticmethod
        def norm(v) -> float:
            return math.sqrt(sum(float(x) * float(x) for x in v))


class _FakeCache:
    def __init__(self, images: dict[str, _FakeImage]) -> None:
        self._images = images

    def get(self, url: str):
        image = self._images.get(url)
        if image is None:
            raise worker.ImageFetchError(f"no scripted image for {url}")
        return image


def _unit_vector(cosine: float | None) -> list[float] | None:
    """Unit 2D vector whose cosine against [1, 0] is exactly *cosine*."""
    if cosine is None:
        return None
    sine = math.sqrt(max(0.0, 1.0 - float(cosine) * float(cosine)))
    return [float(cosine), sine, 0.0, 0.0]


def _make_face_row(
    *, width: float = 200.0, height: float = 200.0, score: float = 0.99, x=0.0, y=0.0
) -> list[float]:
    """A YuNet-shaped row: [x, y, w, h, 10 landmarks, detection_score]."""
    return [float(x), float(y), float(width), float(height)] + [0.0] * 10 + [
        float(score)
    ]


class _ScriptedFaceLoader(worker.ModelLoader):
    """run_face_match with scripted detection rows and embedding cosines."""

    def __init__(self) -> None:
        super().__init__()
        self.face_detector = object()
        self.face_recognizer = object()
        self._np = _FakeNumpy
        self._cv2 = _FakeCV2()
        self.images: dict[str, _FakeImage] = {}
        self._rows: dict[int, list[float]] = {}
        self._vectors: dict[int, list[float]] = {}
        self._type_by_id: dict[int, str] = {}
        self._keepalive: list[_FakeImage] = []
        self.embedded_types: list[str] = []

    def add_document(
        self,
        doc_type: str,
        *,
        cosine: float | None = None,
        width: float = 200.0,
        height: float = 200.0,
        score: float = 0.99,
        sharpness: float = 250.0,
    ) -> dict[str, str]:
        url = f"https://cdn.example/{doc_type}-{len(self._keepalive) + 1}.jpg"
        image = _FakeImage(sharpness=sharpness)
        self._keepalive.append(image)  # keep ids stable for the whole test
        self.images[url] = image
        self._rows[id(image)] = _make_face_row(width=width, height=height, score=score)
        self._vectors[id(image)] = _unit_vector(cosine)
        self._type_by_id[id(image)] = doc_type
        return {"type": doc_type, "url": url}

    def _detect_largest_face(self, image):
        return image, self._rows[id(image)]

    def _face_embedding(self, image, face):
        self.embedded_types.append(self._type_by_id[id(image)])
        vector = self._vectors[id(image)]
        if vector is None:
            raise ValueError("zero-length face embedding")
        return vector

    def run(self, documents: list[dict[str, str]]) -> dict:
        return self.run_face_match(documents, _FakeCache(self.images))


# ---------------------------------------------------------------------------
# Multi-reference scoring (synthetic cosines through the real pipeline)
# ---------------------------------------------------------------------------


def test_single_reference_aadhaar_only():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.55)

    result = loader.run([selfie, aadhaar])

    assert result["error"] is None
    assert result["reference_count"] == 1
    assert result["reference_scores"] == {"aadhaar": pytest.approx(0.55)}
    assert result["winning_reference"] == "aadhaar"
    assert result["aadhaar_similarity"] == pytest.approx(0.55)
    assert result["face_similarity"] == pytest.approx(0.55)
    assert result["cosine_similarity"] == pytest.approx(0.55)
    assert result["second_reference_divergence"] is False


def test_multi_reference_pan_wins():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.32)
    pan = loader.add_document("pan_card", cosine=0.68)

    result = loader.run([selfie, aadhaar, pan])

    assert result["error"] is None
    assert result["reference_count"] == 2
    assert result["reference_scores"] == {
        "aadhaar": pytest.approx(0.32),
        "pan_card": pytest.approx(0.68),
    }
    assert result["winning_reference"] == "pan_card"
    assert result["face_similarity"] == pytest.approx(0.68)
    assert result["cosine_similarity"] == pytest.approx(0.68)
    assert result["aadhaar_similarity"] == pytest.approx(0.32)
    assert result["second_reference_divergence"] is True  # 0.68 - 0.32 > 0.25


def test_multi_reference_aadhaar_wins():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.55)
    pan = loader.add_document("pan_card", cosine=0.32)

    result = loader.run([selfie, aadhaar, pan])

    assert result["error"] is None
    assert result["winning_reference"] == "aadhaar"
    assert result["face_similarity"] == pytest.approx(0.55)
    assert result["aadhaar_similarity"] == pytest.approx(0.55)
    assert result["second_reference_divergence"] is False


def test_multi_reference_both_strong():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.55)
    pan = loader.add_document("pan_card", cosine=0.60)

    result = loader.run([selfie, aadhaar, pan])

    assert result["error"] is None
    assert result["face_similarity"] == pytest.approx(0.60)
    assert result["winning_reference"] == "pan_card"
    assert result["aadhaar_similarity"] == pytest.approx(0.55)
    assert result["second_reference_divergence"] is False  # 0.60 - 0.55 <= 0.25


def test_multi_reference_both_weak():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.15)
    pan = loader.add_document("pan_card", cosine=0.18)

    result = loader.run([selfie, aadhaar, pan])

    assert result["error"] is None
    assert result["face_similarity"] == pytest.approx(0.18)
    assert result["winning_reference"] == "pan_card"
    assert result["aadhaar_similarity"] == pytest.approx(0.15)
    assert result["reference_count"] == 2
    assert result["second_reference_divergence"] is False  # 0.18 - 0.15 <= 0.25


def test_multi_reference_pan_only_no_aadhaar():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    pan = loader.add_document("pan_card", cosine=0.65)

    result = loader.run([selfie, pan])

    assert result["error"] is None
    assert result["reference_count"] == 1
    assert result["reference_scores"] == {"pan_card": pytest.approx(0.65)}
    assert result["winning_reference"] == "pan_card"
    assert result["aadhaar_similarity"] is None
    assert result["second_reference_divergence"] is False
    assert set(result["quality"]) == {"pan_card"}


def test_single_reference_legacy_govt_id_only():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    govt = loader.add_document("govt_id", cosine=0.44)

    result = loader.run([selfie, govt])

    assert result["error"] is None
    assert result["reference_count"] == 1
    assert result["reference_scores"] == {"govt_id": pytest.approx(0.44)}
    assert result["winning_reference"] == "govt_id"
    # Legacy govt_id is scored as a reference but is NOT an Aadhaar trust
    # anchor: the similarity stays None so the router forces manual review.
    assert result["aadhaar_similarity"] is None
    assert result["second_reference_divergence"] is False


def test_no_usable_reference_returns_error():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.55, width=40.0, height=40.0)
    pan = loader.add_document("pan_card", cosine=0.60, width=60.0, height=60.0)

    result = loader.run([selfie, aadhaar, pan])

    assert result["error"] == "no_usable_reference"
    assert result["reference_scores"] == {}
    assert result["reference_count"] == 0
    assert result["aadhaar_similarity"] is None
    assert result["quality"]["aadhaar"]["usable"] is False
    assert result["quality"]["pan_card"]["usable"] is False


# ---------------------------------------------------------------------------
# Quality gates
# ---------------------------------------------------------------------------


def test_quality_gate_skips_blurry_small_face():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.70, width=50.0, height=120.0)
    pan = loader.add_document("pan_card", cosine=0.50)

    result = loader.run([selfie, aadhaar, pan])

    assert result["error"] is None
    assert set(result["reference_scores"]) == {"pan_card"}
    assert result["aadhaar_similarity"] is None
    assert result["winning_reference"] == "pan_card"
    assert result["reference_count"] == 1
    # Skipped by the SIDE gate, not by an embedding failure.
    quality = result["quality"]["aadhaar"]
    assert quality["usable"] is False
    assert quality["width"] == 50
    assert "error" not in quality
    assert "aadhaar" not in loader.embedded_types


def test_quality_gate_skips_low_detection_score():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.70, score=0.50)
    pan = loader.add_document("pan_card", cosine=0.50)

    result = loader.run([selfie, aadhaar, pan])

    assert result["error"] is None
    assert set(result["reference_scores"]) == {"pan_card"}
    assert result["aadhaar_similarity"] is None
    quality = result["quality"]["aadhaar"]
    assert quality["usable"] is False
    assert quality["detection_score"] == pytest.approx(0.5)
    assert "error" not in quality
    assert "aadhaar" not in loader.embedded_types


def test_quality_gate_skips_low_sharpness():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.70, sharpness=30.0)
    pan = loader.add_document("pan_card", cosine=0.50)

    result = loader.run([selfie, aadhaar, pan])

    assert result["error"] is None
    assert set(result["reference_scores"]) == {"pan_card"}
    assert result["aadhaar_similarity"] is None
    quality = result["quality"]["aadhaar"]
    assert quality["usable"] is False
    assert quality["sharpness"] == pytest.approx(30.0)
    assert "error" not in quality
    assert "aadhaar" not in loader.embedded_types


def test_usable_reference_passes_all_three_gates():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.55)

    result = loader.run([selfie, aadhaar])

    assert result["error"] is None
    quality = result["quality"]["aadhaar"]
    assert quality["usable"] is True
    assert quality["width"] == 200 and quality["height"] == 200
    assert quality["detection_score"] == pytest.approx(0.99)
    assert quality["sharpness"] >= 80.0


# ---------------------------------------------------------------------------
# Result contract
# ---------------------------------------------------------------------------

REQUIRED_FACE_KEYS = {
    "engine",
    "model_version",
    "error",
    "face_similarity",
    "winning_reference",
    "aadhaar_similarity",
    "reference_scores",
    "reference_count",
    "second_reference_divergence",
    "quality",
    "cosine_similarity",
    "face_detected",
    "document_types",
    "thresholds",
}


def test_return_shape_has_all_required_keys():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.55)
    pan = loader.add_document("pan_card", cosine=0.60)

    result = loader.run([selfie, aadhaar, pan])

    missing = REQUIRED_FACE_KEYS - set(result)
    assert not missing, f"face result is missing keys: {sorted(missing)}"
    assert result["face_detected"]["selfie"] is True
    assert result["face_detected"]["aadhaar"] is True
    assert result["face_detected"]["pan_card"] is True
    assert result["thresholds"] == {
        "auto_pass": pytest.approx(0.60),
        "review": pytest.approx(0.40),
        "reject": pytest.approx(0.35),
    }
    assert [d["type"] for d in [selfie, aadhaar, pan]] == result["document_types"]


def test_result_json_serializable():
    loader = _ScriptedFaceLoader()
    selfie = loader.add_document("selfie", cosine=1.0)
    aadhaar = loader.add_document("aadhaar", cosine=0.55)
    pan = loader.add_document("pan_card", cosine=0.62)

    result = loader.run([selfie, aadhaar, pan])

    serialized = json.dumps(result)  # must not raise
    reparsed = json.loads(serialized)
    assert reparsed["reference_count"] == 2
    assert reparsed["winning_reference"] == "pan_card"


def test_error_json_serializable():
    """Error results (missing/everything-skipped) also go to the DB as JSONB."""
    loader = _ScriptedFaceLoader()
    result = loader.run([{"type": "selfie", "url": "x.jpg"}])

    serialized = json.dumps(result)  # must not raise
    reparsed = json.loads(serialized)
    assert reparsed["error"] == "missing_document"
    assert reparsed["aadhaar_similarity"] is None
    assert reparsed["second_reference_divergence"] is False


# ---------------------------------------------------------------------------
# Mock mode (ML_MODE=mock) mirrors the multi-reference shape
# ---------------------------------------------------------------------------


def test_mock_face_match_multi_reference_shape():
    result = worker.mock_face_match(
        [
            {"type": "selfie", "url": "https://cdn.example/selfie.jpg"},
            {"type": "aadhaar", "url": "https://cdn.example/aadhaar.jpg"},
            {"type": "pan_card", "url": "https://cdn.example/pan.jpg"},
        ]
    )

    assert result["mock"] is True
    assert result["reference_scores"] == {"aadhaar": 0.45, "pan_card": 0.62}
    assert result["winning_reference"] == "pan_card"
    assert result["face_similarity"] == pytest.approx(0.62)
    assert result["aadhaar_similarity"] == pytest.approx(0.45)
    assert result["second_reference_divergence"] is False  # 0.62 - 0.45 <= 0.25
    assert result["reference_count"] == 2
    assert result["error"] is None
    assert not (REQUIRED_FACE_KEYS - set(result))


def test_mock_face_match_legacy_govt_id_keeps_aadhaar_similarity_none():
    result = worker.mock_face_match(
        [
            {"type": "selfie", "url": "https://cdn.example/selfie.jpg"},
            {"type": "govt_id", "url": "https://cdn.example/govt.jpg"},
        ]
    )

    assert result["winning_reference"] == "govt_id"
    assert result["reference_count"] == 1
    assert result["aadhaar_similarity"] is None
    assert result["second_reference_divergence"] is False


def test_mock_face_match_missing_references_errors():
    result = worker.mock_face_match([{"type": "selfie", "url": "s.jpg"}])

    assert result["error"] == "missing_document"
    assert result["reference_count"] == 0
