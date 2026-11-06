# KYC ML worker image (Phase H: ONNX + PaddleOCR)
# Python 3.11 slim is the target runtime; see requirements.txt.
FROM python:3.11-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    MODEL_DIR=/app/models \
    PADDLE_OCR_BASE_DIR=/app/.paddleocr \
    PADDLE_PDX_CACHE_HOME=/app/.paddlex

WORKDIR /app

# OpenCV / PaddleOCR / PaddlePaddle system libraries.
#
# On Debian bookworm the old `libgl1-mesa-glx` package no longer exists; the
# replacement is `libgl1`. `libgomp1` is required by the Paddle CPU wheel, and
# the remaining libraries cover OpenCV's image codecs / X11 shims.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        libgl1 \
        libglib2.0-0 \
        libgomp1 \
        libsm6 \
        libxext6 \
        libxrender1 \
        libjpeg62-turbo \
        libpng16-16 \
        libtiff6 \
        libwebp7 \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# The compose stack mounts ./models here read-only. The OCR cache directories
# must stay writable for the unprivileged runtime user (PaddleOCR downloads its
# detector/recognizer weights on first use).
RUN mkdir -p /app/models /app/.paddleocr /app/.paddlex \
    && useradd --create-home --uid 10001 appuser \
    && chown -R appuser:appuser /app

USER appuser

EXPOSE 8000

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
