# KYC ML Service

Python worker that consumes KYC verification jobs from **Redis Streams**, runs
OCR / face-match / liveness inference, and writes the results into
**PostgreSQL**.

The worker is the only service in this directory. PostgreSQL and Redis are
**external cloud services** (Neon, Supabase, Upstash, Redis Cloud, ElastiCache,
...). There are no local database or cache containers.

```
Go backend ──> Cloud Redis Stream ──> ml-worker (this service)
     │                                      │
     └──────────> Cloud PostgreSQL <────────┘
```

The Go backend and the worker must use the **same** cloud PostgreSQL database
and the **same** Redis instance.

---

## 1. Configure cloud credentials

```bash
cd kyc-ml-service
cp .env.example .env
```

Edit `.env`:

```dotenv
# Neon / Supabase / any managed Postgres (sslmode=require is required)
DATABASE_URL=postgresql://USER:PASSWORD@ep-xxx.aws.neon.tech/neondb?sslmode=require

# Redis Cloud Free tier (no TLS) — use the values from your dashboard
REDIS_HOST=redis-12345.c1.us-east-1-2.ec2.cloud.redislabs.com
REDIS_PORT=15012
REDIS_USERNAME=default
REDIS_PASSWORD=your-password
REDIS_SSL=false

# Upstash / paid TLS plan: use port 6380 and REDIS_SSL=true instead
```

Notes:

- Keep `?sslmode=require` (or `verify-full`) in `DATABASE_URL`. The worker
  passes the URI directly to libpq, which handles cloud TLS.
- Set `REDIS_SSL=true` **only** when the Redis endpoint/plan requires TLS.
  Redis Cloud Free tier does **not** support TLS, so keep
  `REDIS_SSL=false` (the default) for it. Local Redis also uses
  `REDIS_SSL=false`.
- If a TLS provider uses a self-signed certificate, also set
  `REDIS_SSL_CERT_REQS=none`. Prefer `required` everywhere else.
- `.env` is git-ignored. Never commit real credentials.

## 2. Download the models (once)

```bash
python3 download_models.py
```

This places the YuNet/SFace/MiniFASNet ONNX files in `./models`.

## 3A. Run the worker with Docker

```bash
docker compose up -d --build
docker compose logs -f ml-worker
```

The compose file contains **only** `ml-worker`; it reads all credentials from
`.env` via `env_file`. The container-specific model paths are set in the
compose `environment` block, so the same `.env` also works for a bare-metal
run.

Health endpoint: <http://localhost:8000/health>

```json
{
  "status": "ok",
  "redis": "connected",
  "db": "connected",
  "models": {"mode": "real", "state": "ready", "...": "..."}
}
```

`503 / degraded` means the worker cannot reach cloud Postgres or Redis (or the
models failed to load).

Stop it with:

```bash
docker compose down
```

## 3B. Run the worker directly (no Docker)

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python worker.py            # consumer only
# or, with the health endpoint:
python -m uvicorn main:app --host 0.0.0.0 --port 8000
```

## 4. End-to-end test (cloud DB + Redis)

The Go backend and the ML worker must already be running and pointed at the
same cloud services. The script no longer starts or stops containers.

```bash
./test_e2e.sh
```

It reads `DATABASE_URL` / `REDIS_*` from the shell first, then from `.env`.
The same Python dependencies as the worker are required; use a virtualenv or
`E2E_PYTHON=/path/to/python`.

Useful overrides:

```bash
# Match the backend's real JWT secret
E2E_JWT_ACCESS_SECRET=... ./test_e2e.sh

# Full real-model run (images must be readable by the worker process)
E2E_ML_MODE=real \
E2E_GOVT_URL=file:///tmp/kyc-id.jpg \
E2E_SELFIE_URL=file:///tmp/kyc-selfie.jpg \
./test_e2e.sh

# Also require the worker HTTP health endpoint
E2E_REQUIRE_ML_HEALTH=1 ./test_e2e.sh
```

Expected final line: `✅ E2E PASS (cloud PostgreSQL + Redis)`.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| `DATABASE_URL is not set` | `.env` exists and is in `kyc-ml-service/`, or export the variable |
| `connection refused` to Postgres | Use the provider's **pooler/direct** host and `sslmode=require` |
| Redis timeout / reset by peer | Set `REDIS_SSL=true` for TLS endpoints |
| TLS certificate error | Set `REDIS_SSL_CERT_REQS=none` only for self-signed certs |
| `/health` degraded | Worker cannot ping cloud DB/Redis; check credentials and egress rules |
