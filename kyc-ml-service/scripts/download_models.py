#!/usr/bin/env python3
"""Download the ONNX weights used by the Phase H KYC pipeline.

Models
------
* ``face_detection_yunet_2023mar.onnx``  - OpenCV Zoo YuNet face detector
* ``face_recognition_sface_2021dec.onnx`` - OpenCV Zoo SFace recognizer
* ``MiniFASNetV2.onnx``                   - Silent-Face-Anti-Spoofing liveness

The three files are small enough to keep outside the Docker image. The compose
stack mounts ``kyc-ml-service/models`` at ``/app/models`` read-only.

PaddleOCR downloads its own detector/recognizer weights on first use. Pass
``--paddleocr`` to warm that cache now (useful before starting an offline
worker). In Docker::

    docker compose run --rm ml-worker \
        python download_models.py --paddleocr --skip-onnx

Usage
-----
    python download_models.py                 # download the 3 ONNX models
    python download_models.py --force         # re-download even if present
    python download_models.py --only yunet    # one model
    python download_models.py --paddleocr     # also warm PaddleOCR cache
    python download_models.py --skip-onnx     # PaddleOCR cache only
"""

from __future__ import annotations

import argparse
import hashlib
import os
import shutil
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Source:
    url: str
    sha256: str
    size: int


@dataclass(frozen=True)
class ModelSpec:
    key: str
    filename: str
    description: str
    sources: tuple[Source, ...]


MODELS: tuple[ModelSpec, ...] = (
    ModelSpec(
        key="yunet",
        filename="face_detection_yunet_2023mar.onnx",
        description="YuNet face detector (OpenCV Zoo, 2023mar)",
        sources=(
            Source(
                url=(
                    "https://github.com/opencv/opencv_zoo/raw/main/"
                    "models/face_detection_yunet/"
                    "face_detection_yunet_2023mar.onnx"
                ),
                sha256="8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4",
                size=232_589,
            ),
            Source(
                url=(
                    "https://media.githubusercontent.com/media/opencv/"
                    "opencv_zoo/main/models/face_detection_yunet/"
                    "face_detection_yunet_2023mar.onnx"
                ),
                sha256="8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4",
                size=232_589,
            ),
        ),
    ),
    ModelSpec(
        key="sface",
        filename="face_recognition_sface_2021dec.onnx",
        description="SFace face recognizer (OpenCV Zoo, 2021dec)",
        sources=(
            Source(
                url=(
                    "https://github.com/opencv/opencv_zoo/raw/main/"
                    "models/face_recognition_sface/"
                    "face_recognition_sface_2021dec.onnx"
                ),
                sha256="0ba9fbfa01b5270c96627c4ef784da859931e02f04419c829e83484087c34e79",
                size=38_696_353,
            ),
            Source(
                url=(
                    "https://media.githubusercontent.com/media/opencv/"
                    "opencv_zoo/main/models/face_recognition_sface/"
                    "face_recognition_sface_2021dec.onnx"
                ),
                sha256="0ba9fbfa01b5270c96627c4ef784da859931e02f04419c829e83484087c34e79",
                size=38_696_353,
            ),
        ),
    ),
    ModelSpec(
        key="liveness",
        filename="MiniFASNetV2.onnx",
        description="MiniFASNetV2 passive liveness (2.7x / 80x80)",
        sources=(
            Source(
                url=(
                    "https://huggingface.co/garciafido/"
                    "minifasnet-v2-anti-spoofing-onnx/resolve/main/"
                    "minifasnet_v2.onnx"
                ),
                sha256="d7b3cd9ba8a7ceb13baa8c4720902e27ca3112eff52f926c08804af6b6eecc7b",
                size=1_744_116,
            ),
            Source(
                url=(
                    "https://github.com/yakhyo/face-anti-spoofing/"
                    "releases/download/weights/MiniFASNetV2.onnx"
                ),
                sha256="b32929adc2d9c34b9486f8c4c7bc97c1b69bc0ea9befefc380e4faae4e463907",
                size=1_743_581,
            ),
        ),
    ),
)

CHUNK_SIZE = 256 * 1024


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(CHUNK_SIZE), b""):
            digest.update(chunk)
    return digest.hexdigest()


def download_one(source: Source, destination: Path, timeout: int = 120) -> None:
    """Stream one URL to ``destination`` with a basic progress meter."""
    request = urllib.request.Request(
        source.url,
        headers={
            "User-Agent": "techguild-kyc-model-downloader/1.0",
            "Accept": "application/octet-stream,*/*",
        },
    )

    partial = destination.with_suffix(destination.suffix + ".part")
    partial.unlink(missing_ok=True)

    started = time.monotonic()
    downloaded = 0
    last_report = 0.0

    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            total = source.size
            declared = response.headers.get("Content-Length")
            if declared and declared.isdigit():
                total = int(declared)

            with partial.open("wb") as out:
                while True:
                    chunk = response.read(CHUNK_SIZE)
                    if not chunk:
                        break
                    out.write(chunk)
                    downloaded += len(chunk)
                    now = time.monotonic()
                    if now - last_report >= 0.5:
                        last_report = now
                        pct = (downloaded / total * 100.0) if total else 0.0
                        sys.stdout.write(
                            f"\r    {downloaded / 1_048_576:6.1f} MiB "
                            f"({pct:5.1f}%)"
                        )
                        sys.stdout.flush()

        sys.stdout.write("\r" + " " * 40 + "\r")
        sys.stdout.flush()

        actual_size = partial.stat().st_size
        actual_sha = sha256_file(partial)

        if actual_size != source.size:
            raise ValueError(
                f"size mismatch: expected {source.size} bytes, got {actual_size}"
            )
        if actual_sha != source.sha256:
            raise ValueError(
                "sha256 mismatch: expected "
                f"{source.sha256}, got {actual_sha}"
            )

        partial.replace(destination)
        elapsed = max(time.monotonic() - started, 0.001)
        print(
            f"    verified {actual_size / 1_048_576:.1f} MiB "
            f"in {elapsed:.1f}s"
        )
    except Exception:
        partial.unlink(missing_ok=True)
        raise


def ensure_model(spec: ModelSpec, destination_dir: Path, force: bool) -> bool:
    destination = destination_dir / spec.filename
    expected = spec.sources[0]

    if destination.is_file() and not force:
        actual_size = destination.stat().st_size
        actual_sha = sha256_file(destination)
        if actual_size == expected.size and actual_sha == expected.sha256:
            print(f"[skip] {spec.filename} already present and verified")
            return True
        print(
            f"[warn] {spec.filename} exists but does not match the expected "
            "checksum; re-downloading"
        )

    print(f"\n[{spec.key}] {spec.description}")
    print(f"  -> {destination}")

    for index, source in enumerate(spec.sources, start=1):
        print(f"  source {index}/{len(spec.sources)}: {source.url}")
        try:
            download_one(source, destination)
            return True
        except (urllib.error.URLError, urllib.error.HTTPError) as exc:
            print(f"  [error] download failed: {exc}")
        except Exception as exc:  # noqa: BLE001 - try the next mirror
            print(f"  [error] {exc}")

    print(f"[fail] could not download {spec.filename} from any source")
    return False


def warm_paddleocr(lang: str) -> bool:
    """Instantiate PaddleOCR once so its models are cached locally."""
    print("\n[paddleocr] warming detector/recognizer model cache")
    try:
        import numpy as np
        from paddleocr import PaddleOCR
    except Exception as exc:  # noqa: BLE001 - optional step
        print(f"[fail] PaddleOCR is not importable: {exc}")
        print("       Install the service requirements first.")
        return False

    try:
        try:
            engine = PaddleOCR(lang=lang, use_angle_cls=False, show_log=False)
        except TypeError:
            # PaddleOCR 3.x constructor.
            engine = PaddleOCR(
                lang=lang,
                use_doc_orientation_classify=False,
                use_doc_unwarping=False,
                use_textline_orientation=False,
            )

        probe = np.full((96, 384, 3), 255, dtype=np.uint8)
        try:
            if hasattr(engine, "predict"):
                list(engine.predict(probe))
            else:
                engine.ocr(probe, cls=False)
        except Exception:
            # A blank image may yield no detections; the point of this call is
            # to force the model files to download, so a detection error here
            # is not fatal.
            pass

        print("[ok] PaddleOCR model cache is ready")
        return True
    except Exception as exc:  # noqa: BLE001 - optional step
        print(f"[fail] PaddleOCR warm-up failed: {exc}")
        return False


def parse_args() -> argparse.Namespace:
    default_dest = Path(__file__).resolve().parent.parent / "models"
    parser = argparse.ArgumentParser(
        description="Download ONNX weights for the TechGuild KYC ML worker."
    )
    parser.add_argument(
        "--dest",
        type=Path,
        default=default_dest,
        help=f"destination directory (default: {default_dest})",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="re-download files even when a verified copy exists",
    )
    parser.add_argument(
        "--only",
        action="append",
        choices=[spec.key for spec in MODELS],
        help="download only the given model key (repeatable)",
    )
    parser.add_argument(
        "--skip-onnx",
        action="store_true",
        help="do not download the three ONNX files",
    )
    parser.add_argument(
        "--paddleocr",
        action="store_true",
        help="also instantiate PaddleOCR to warm its model cache",
    )
    parser.add_argument(
        "--lang",
        default=os.getenv("OCR_LANG", "en"),
        help="PaddleOCR language for --paddleocr (default: en)",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    args.dest.mkdir(parents=True, exist_ok=True)

    selected = [
        spec
        for spec in MODELS
        if not args.only or spec.key in set(args.only)
    ]

    ok = True
    if not args.skip_onnx:
        print(f"Downloading ONNX models to {args.dest}")
        for spec in selected:
            ok = ensure_model(spec, args.dest, args.force) and ok

    if args.paddleocr:
        ok = warm_paddleocr(args.lang) and ok

    if not args.skip_onnx:
        print("\nContents:")
        for spec in selected:
            path = args.dest / spec.filename
            if path.is_file():
                print(
                    f"  {path.name:<45} "
                    f"{path.stat().st_size / 1_048_576:7.2f} MiB"
                )
            else:
                print(f"  {path.name:<45} MISSING")

    if not ok:
        print(
            "\nSome models could not be downloaded. You can also place the "
            "files manually:"
        )
        for spec in MODELS:
            print(f"  {spec.filename}  ->  {args.dest / spec.filename}")
        print(
            "\nIf only the liveness mirror is unavailable, the worker will "
            "still start, but liveness will return score=0.0 / is_live=false."
        )
        return 1

    print("\nDone. Rebuild/restart the worker to pick up the models:")
    print("  docker compose build ml-worker")
    print("  docker compose up -d --force-recreate ml-worker")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
