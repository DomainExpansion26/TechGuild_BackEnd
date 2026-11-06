# TechGuild Backend

Two independently runnable services:

| Service | Directory | What it is |
| --- | --- | --- |
| Go API | `backend/` | Main TechGuild backend (REST API, auth, KYC orchestration) |
| KYC ML worker | `kyc-ml-service/` | Python worker (OCR / face match / liveness) |

PostgreSQL and Redis are **external cloud services** (Neon, Supabase, Upstash,
Redis Cloud, ...). Neither service runs a local database or cache container.

```
Go backend ──> Cloud Redis Stream (kyc:verification:jobs) ──> ml-worker
     │                                                            │
     └──────────────> Cloud PostgreSQL <──────────────────────────┘
```

Both services must point at the same cloud PostgreSQL database and Redis
instance.

> **Local development:** see [`LOCAL_TESTING.md`](LOCAL_TESTING.md) for the
> three-terminal workflow and troubleshooting.

---

## Terminal 1 — KYC ML worker

```bash
cd kyc-ml-service
cp .env.example .env     # first time only; fill in cloud credentials
docker compose up -d --build
docker compose logs -f ml-worker
```

`kyc-ml-service/docker-compose.yml` contains **only** the `ml-worker` service.
All connection settings come from `kyc-ml-service/.env`.

Health: <http://localhost:8000/health>

See `kyc-ml-service/README.md` for bare-metal run instructions, model download,
and troubleshooting.

## Terminal 2 — Go backend

The backend reads `backend/.env` by default. Point it at the same cloud
PostgreSQL/Redis instances (either edit `.env` or export overrides):

```dotenv
DATABASE_URL=postgresql://USER:PASSWORD@ep-xxx.aws.neon.tech/neondb?sslmode=require
REDIS_HOST=your-redis-host
REDIS_PORT=15012
REDIS_USERNAME=default
REDIS_PASSWORD=your-redis-password
REDIS_SSL=false
```

Then run:

```bash
cd backend
go run ./cmd/server
```

The API listens on `http://localhost:8080` (health: `GET /health`).

> `REDIS_SSL` is optional and defaults to `false`. Leave it `false` for Redis
> Cloud Free tier and local Redis; set `REDIS_SSL=true` only if your
> provider/plan requires TLS. Both the Go backend and the Python worker honor
> the same variable.

## End-to-end KYC test

With both services running:

```bash
cd kyc-ml-service
./test_e2e.sh
```

The script talks directly to the configured cloud PostgreSQL and Redis (from
the environment or `.env`) and does not start/stop any containers. It inserts a
dummy record, submits it through the Go API, waits for the worker, and verifies
the update in the cloud database.

Expected final line: `✅ E2E PASS (cloud PostgreSQL + Redis)`.
