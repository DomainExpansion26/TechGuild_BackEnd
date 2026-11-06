"""FastAPI entry point for the KYC ML service.

The HTTP surface is intentionally small (health checks only). The actual work
happens in the Redis Stream consumer started in the FastAPI lifespan hook.
"""

from __future__ import annotations

import logging
import os
from contextlib import asynccontextmanager
from typing import AsyncIterator

from fastapi import FastAPI
from fastapi.responses import JSONResponse

from app.config import settings
from app.worker import KYCWorker, ping_database

logger = logging.getLogger("kyc.main")

# One worker instance per process. Uvicorn's worker processes each start their
# own consumer; Redis consumer groups distribute the jobs between them.
worker = KYCWorker()


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    logging.basicConfig(
        level=os.getenv("LOG_LEVEL", "INFO").upper(),
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    logger.info(
        "Starting KYC ML service (stream=%s, group=%s)",
        settings.kyc_stream,
        settings.kyc_consumer_group,
    )

    app.state.worker = worker
    worker.start()
    try:
        yield
    finally:
        logger.info("Stopping KYC ML service")
        worker.stop()


app = FastAPI(
    title="TechGuild KYC ML Service",
    description=(
        "Phase H AI worker: consumes KYC verification jobs from Redis Streams, "
        "runs PaddleOCR / YuNet+SFace / MiniFASNet inference, and writes results "
        "to PostgreSQL."
    ),
    version="0.2.0",
    lifespan=lifespan,
)


@app.get("/health", tags=["system"])
def health() -> JSONResponse:
    """Liveness/readiness probe for Redis and PostgreSQL.

    Returns HTTP 200 with the expected ``{"status": "ok", ...}`` payload when
    both dependencies are reachable, otherwise HTTP 503 with ``degraded``.
    """
    redis_ok = worker.ping_redis()
    db_ok = ping_database()
    healthy = redis_ok and db_ok

    body = {
        "status": "ok" if healthy else "degraded",
        "redis": "connected" if redis_ok else "disconnected",
        "db": "connected" if db_ok else "disconnected",
        "models": worker.model_loader.status(),
    }
    return JSONResponse(status_code=200 if healthy else 503, content=body)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=int(os.getenv("PORT", "8000")),
        reload=False,
    )
