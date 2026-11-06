#!/usr/bin/env bash
#
# KYC end-to-end smoke test against EXTERNAL (cloud) PostgreSQL + Redis.
#
# Flow exercised:
#   1. Seed a draft KYC user + record + two documents in the cloud database.
#   2. Generate a valid access token for that user (same secret as Go backend).
#   3. POST /v1/verification/submit -> the Go backend queues a job on Redis.
#   4. Confirm the job appears on the configured Redis stream.
#   5. Wait for the separately running Python ML worker to complete it.
#   6. Assert ml_status='ml_completed' in the cloud database.
#
# This script does NOT start or stop Docker containers. The ML worker and the
# Go backend must already be running, and both must point at the same cloud
# PostgreSQL / Redis instances configured here.
#
# Local workflow:
#   Terminal 1 (Go)     : cd backend && go run ./cmd/server
#   Terminal 2 (Docker) : cd kyc-ml-service && docker compose up -d --build
#   Terminal 3 (Test)   : cd kyc-ml-service && ./test_e2e.sh
#
# Configuration resolution order:
#   1. Exported environment variables (highest priority)
#   2. `kyc-ml-service/.env` (copy from .env.example)
#
# The test JWT is signed with JWT_ACCESS_SECRET. Resolution order:
#   1. E2E_JWT_ACCESS_SECRET
#   2. JWT_ACCESS_SECRET read from the backend env file
#      (E2E_BACKEND_ENV_FILE, default: ../backend/.env)
#   3. a local-only fallback value
#
# Required connection settings:
#   DATABASE_URL   cloud PostgreSQL URI, e.g. ...?sslmode=require
#   REDIS_HOST     cloud Redis hostname
#   REDIS_PORT     cloud Redis port (default 6379)
#   REDIS_PASSWORD cloud Redis password (if required)
#   REDIS_SSL      optional: true only for TLS endpoints. Leave unset/false
#                  for Redis Cloud Free tier and local Redis.
#
# Usage:
#   ./test_e2e.sh
#   E2E_JWT_ACCESS_SECRET=... ./test_e2e.sh
#   E2E_ML_MODE=real E2E_GOVT_URL=file:///tmp/kyc-id.jpg \
#     E2E_SELFIE_URL=file:///tmp/kyc-selfie.jpg ./test_e2e.sh
#
# The script needs the same Python dependencies as the worker
# (`pip install -r requirements.txt`) because it uses psycopg2 + redis-py to
# talk to the cloud services. Override the interpreter with E2E_PYTHON.
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${E2E_ENV_FILE:-$SCRIPT_DIR/.env}"
export E2E_ENV_FILE="$ENV_FILE"

BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
ML_URL="${ML_URL:-http://localhost:8000}"
# Resolved from .env below (falls back to the default stream name).
KYC_STREAM="${KYC_STREAM:-}"

# The generated token must be signed with the SAME secret the Go backend uses.
# Precedence: E2E_JWT_ACCESS_SECRET -> backend env file -> local fallback.
JWT_ACCESS_SECRET="${E2E_JWT_ACCESS_SECRET:-}"
BACKEND_ENV_FILE="${E2E_BACKEND_ENV_FILE:-$SCRIPT_DIR/../backend/.env}"
if [[ -z "$JWT_ACCESS_SECRET" && -f "$BACKEND_ENV_FILE" ]]; then
  JWT_FROM_BACKEND="$(
    grep -E '^[[:space:]]*JWT_ACCESS_SECRET=' "$BACKEND_ENV_FILE" \
      | tail -n 1 \
      | cut -d= -f2- \
      | tr -d '\r' \
      | tr -d '"' \
      | tr -d "'" \
      | xargs
  )"
  if [[ -n "$JWT_FROM_BACKEND" ]]; then
    JWT_ACCESS_SECRET="$JWT_FROM_BACKEND"
    echo "==> Using JWT_ACCESS_SECRET from $BACKEND_ENV_FILE"
  fi
  unset JWT_FROM_BACKEND
fi
JWT_ACCESS_SECRET="${JWT_ACCESS_SECRET:-e2e-jwt-access-secret-do-not-use-in-prod}"

# Fixed IDs make the test idempotent and easy to inspect.
USER_ID="11111111-1111-4111-8111-111111111111"
VERIFICATION_ID="22222222-2222-4222-8222-222222222222"
GOVT_DOC_ID="33333333-3333-4333-8333-333333333333"
SELFIE_DOC_ID="44444444-4444-4444-8444-444444444444"

GOVT_URL="${E2E_GOVT_URL:-https://example.test/kyc/govt-id.jpg}"
SELFIE_URL="${E2E_SELFIE_URL:-https://example.test/kyc/selfie.jpg}"

# Prefer an explicit interpreter, then the service virtualenv, then system Python.
if [[ -n "${E2E_PYTHON:-}" ]]; then
  PYTHON_BIN="$E2E_PYTHON"
elif [[ -x "$SCRIPT_DIR/.venv/bin/python" ]]; then
  PYTHON_BIN="$SCRIPT_DIR/.venv/bin/python"
else
  PYTHON_BIN="python3"
fi

if [[ ! -f "$ENV_FILE" && -z "${DATABASE_URL:-}" ]]; then
  echo "❌ E2E FAIL: no .env found at $ENV_FILE and DATABASE_URL is not set." >&2
  echo "   Copy kyc-ml-service/.env.example to .env and fill in your cloud credentials." >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Python helper (cloud PostgreSQL + Redis + JWT).
# Written to a temp file so the test stays a single self-contained script.
# ---------------------------------------------------------------------------
HELPER="$(mktemp "${TMPDIR:-/tmp}/kyc-e2e-helper.XXXXXX.py")"
trap 'rm -f "$HELPER"' EXIT

cat > "$HELPER" <<'PY'
"""Helpers for the KYC E2E test (cloud PostgreSQL + Redis)."""
from __future__ import annotations

import base64
import hashlib
import hmac
import json
import os
import sys
import time
from pathlib import Path


def _load_env() -> None:
    env_file = os.environ.get("E2E_ENV_FILE") or str(
        Path(__file__).resolve().with_name(".env")
    )
    try:
        from dotenv import load_dotenv
    except ImportError:
        raise SystemExit(
            "python-dotenv is missing; run: pip install -r requirements.txt"
        )
    # Existing environment variables win.
    load_dotenv(env_file, override=False)


def _env_bool(name: str, default: bool = False) -> bool:
    raw = os.getenv(name)
    if raw is None or not raw.strip():
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on"}


def _database_url() -> str:
    url = (os.getenv("DATABASE_URL") or "").strip()
    if not url:
        raise SystemExit("DATABASE_URL is not set (environment or .env)")
    if url.lower().startswith("postgres://"):
        url = "postgresql://" + url[len("postgres://") :]
    return url


def _connect():
    try:
        import psycopg2
    except ImportError:
        raise SystemExit("psycopg2 is missing; run: pip install -r requirements.txt")
    return psycopg2.connect(
        _database_url(),
        connect_timeout=int(os.getenv("DB_CONNECT_TIMEOUT", "10")),
        application_name="kyc-e2e-test",
    )


def _redis_client():
    try:
        import redis
    except ImportError:
        raise SystemExit("redis is missing; run: pip install -r requirements.txt")

    host = (os.getenv("REDIS_HOST") or "").strip()
    if not host:
        raise SystemExit("REDIS_HOST is not set (environment or .env)")

    kwargs = {
        "host": host,
        "port": int(os.getenv("REDIS_PORT", "6379")),
        "password": (os.getenv("REDIS_PASSWORD") or None),
        "username": (os.getenv("REDIS_USERNAME") or None),
        "db": int(os.getenv("REDIS_DB", "0")),
        "decode_responses": True,
        "socket_connect_timeout": int(os.getenv("REDIS_CONNECT_TIMEOUT", "5")),
        "socket_timeout": int(os.getenv("REDIS_SOCKET_TIMEOUT", "15")),
    }
    if _env_bool("REDIS_SSL", False):
        kwargs["ssl"] = True
        cert_reqs = (os.getenv("REDIS_SSL_CERT_REQS") or "required").strip()
        kwargs["ssl_cert_reqs"] = cert_reqs
        kwargs["ssl_check_hostname"] = cert_reqs != "none"
    return redis.Redis(**kwargs)


def cmd_env_get(name: str) -> None:
    print(os.getenv(name, ""))


def cmd_check() -> None:
    try:
        with _connect() as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT 1")
                cur.fetchone()
        _redis_client().ping()
    except SystemExit:
        raise
    except Exception as exc:  # noqa: BLE001 - surface a clean E2E error
        raise SystemExit(f"connectivity check failed: {exc}") from None
    print("PostgreSQL and Redis are reachable")


def cmd_db_exec() -> None:
    sql = sys.stdin.read()
    if not sql.strip():
        raise SystemExit("db-exec received empty SQL")
    with _connect() as conn:
        with conn.cursor() as cur:
            cur.execute(sql)
        conn.commit()


def cmd_db_scalar() -> None:
    sql = sys.stdin.read()
    if not sql.strip():
        raise SystemExit("db-scalar received empty SQL")
    with _connect() as conn:
        with conn.cursor() as cur:
            cur.execute(sql)
            row = cur.fetchone()
    if not row or row[0] is None:
        print("")
    else:
        print(row[0])


def cmd_jwt(user_id: str, secret: str) -> None:
    def b64url(raw: bytes) -> str:
        return base64.urlsafe_b64encode(raw).rstrip(b"=").decode("ascii")

    header = b64url(
        json.dumps({"alg": "HS256", "typ": "JWT"}, separators=(",", ":")).encode(
            "utf-8"
        )
    )
    payload = b64url(
        json.dumps(
            {
                "user_id": user_id,
                "session_id": "e2e-session",
                "type": "access",
                "exp": int(time.time()) + 3600,
            },
            separators=(",", ":"),
        ).encode("utf-8")
    )
    signing_input = f"{header}.{payload}".encode("ascii")
    signature = b64url(
        hmac.new(secret.encode("utf-8"), signing_input, hashlib.sha256).digest()
    )
    print(f"{header}.{payload}.{signature}")


def cmd_stream_check(stream: str, verification_id: str) -> None:
    client = _redis_client()
    length = int(client.xlen(stream))
    if length < 1:
        raise SystemExit(f"stream {stream} is empty after submit")

    entries = client.xrevrange(stream, count=1)
    found = any(
        verification_id in str(value)
        for _message_id, fields in entries
        for value in fields.values()
    )
    if not found:
        raise SystemExit(
            f"latest entry in {stream} does not contain "
            f"verification_id={verification_id}"
        )
    print(length)


def cmd_inspect(verification_id: str) -> None:
    sql = """
        SELECT id, status, ml_status, review_priority,
               ocr_ocr_result, face_match_result, liveness_result
          FROM verification_records
         WHERE id = %s::uuid
    """
    with _connect() as conn:
        with conn.cursor() as cur:
            cur.execute(sql, (verification_id,))
            row = cur.fetchone()
            columns = [description[0] for description in cur.description]

    if row is None:
        raise SystemExit(f"verification record not found: {verification_id}")

    payload = {}
    for column, value in zip(columns, row):
        if hasattr(value, "isoformat"):
            value = value.isoformat()
        payload[column] = value
    print(json.dumps(payload, indent=2, sort_keys=True, default=str))


def main() -> None:
    _load_env()

    if len(sys.argv) < 2:
        raise SystemExit("usage: helper <command> [args...]")

    command = sys.argv[1]
    args = sys.argv[2:]

    if command == "env-get":
        cmd_env_get(*args)
    elif command == "check":
        cmd_check()
    elif command == "db-exec":
        cmd_db_exec()
    elif command == "db-scalar":
        cmd_db_scalar()
    elif command == "jwt":
        cmd_jwt(*args)
    elif command == "stream-check":
        cmd_stream_check(*args)
    elif command == "inspect":
        cmd_inspect(*args)
    else:
        raise SystemExit(f"unknown helper command: {command}")


if __name__ == "__main__":
    main()
PY

py() {
  "$PYTHON_BIN" "$HELPER" "$@"
}

if [[ -z "$KYC_STREAM" ]]; then
  KYC_STREAM="$(py env-get KYC_STREAM || true)"
fi
KYC_STREAM="${KYC_STREAM:-kyc:verification:jobs}"

fail() {
  echo
  echo "❌ E2E FAIL: $*" >&2
  echo
  echo "--- Configuration ---" >&2
  echo "  .env file    : $ENV_FILE" >&2
  echo "  Backend URL  : $BACKEND_URL" >&2
  echo "  ML URL       : $ML_URL" >&2
  echo "  ML worker    : check the terminal/container where it is running" >&2
  echo "  Go backend   : check the terminal where it is running" >&2
  exit 1
}

wait_for_http() {
  local url="$1"
  local name="$2"
  local attempts="${3:-120}"

  echo "==> Waiting for $name at $url"
  for ((i = 1; i <= attempts; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "    $name is ready"
      return 0
    fi
    sleep 1
  done

  fail "$name did not become healthy at $url"
}

echo "==> Local E2E run"
echo "    Go backend : $BACKEND_URL  (start it yourself: cd backend && go run ./cmd/server)"
echo "    ML worker  : $ML_URL  (start it yourself: cd kyc-ml-service && docker compose up -d --build)"
echo "    This script never starts or stops Docker containers."
echo

echo "==> Checking cloud PostgreSQL + Redis connectivity"
py check || fail "cannot reach the configured cloud PostgreSQL/Redis"

echo
echo "==> Waiting for the Go backend"
wait_for_http "$BACKEND_URL/health" "Go backend" 180

if [[ "${E2E_REQUIRE_ML_HEALTH:-0}" == "1" ]]; then
  wait_for_http "$ML_URL/health" "ML worker" 120
elif curl -fsS "$ML_URL/health" >/dev/null 2>&1; then
  echo "    ML worker health OK"
else
  echo "    WARNING: no ML worker health endpoint at $ML_URL/health"
  echo "             (continuing; the test will wait for the worker to write to the DB)"
fi

# Determine the worker's inference mode for the priority assertion:
#   1. explicit E2E_ML_MODE
#   2. /health response (when the worker exposes HTTP)
#   3. unknown -> only require ml_status=ml_completed
ML_MODE="${E2E_ML_MODE:-}"
if [[ -z "$ML_MODE" ]]; then
  ML_MODE="$(
    "$PYTHON_BIN" - "$ML_URL/health" <<'PY' 2>/dev/null || true
import json
import sys
import urllib.request

try:
    with urllib.request.urlopen(sys.argv[1], timeout=3) as response:
        print(json.load(response).get("models", {}).get("mode", ""))
except Exception:
    print("")
PY
  )"
fi

if [[ "$ML_MODE" == "mock" ]]; then
  EXPECT_PRIORITY="auto_eligible"
else
  EXPECT_PRIORITY=""
fi

echo
echo "==> Seeding draft verification record (ML_MODE=${ML_MODE:-unknown})"
GOVT_URL_SQL="${GOVT_URL//\'/\'\'}"
SELFIE_URL_SQL="${SELFIE_URL//\'/\'\'}"
if ! py db-exec <<SQL
-- Idempotent cleanup from previous runs.
-- NOTE: verification_audit_logs and bulk_review_items store UUIDs in text
-- columns (their Go models do not declare type:uuid), so do not cast these.
DELETE FROM verification_audit_logs
 WHERE verification_id = '$VERIFICATION_ID'
    OR user_id = '$USER_ID';
DELETE FROM bulk_review_items
 WHERE verification_id = '$VERIFICATION_ID';
DELETE FROM verification_documents
 WHERE verification_record_id = '$VERIFICATION_ID'::uuid;
DELETE FROM verification_records
 WHERE id = '$VERIFICATION_ID'::uuid;
DELETE FROM govt_id_dedups
 WHERE user_id = '$USER_ID'::uuid;
DELETE FROM users
 WHERE id = '$USER_ID'::uuid
    OR email = 'kyc-e2e@example.test';

INSERT INTO users (
  id, first_name, last_name, email, password_hash, account_type, status,
  email_verified, points, rank, two_factor_enabled, created_at, updated_at
) VALUES (
  '$USER_ID'::uuid,
  'KYC', 'E2E', 'kyc-e2e@example.test', '', 'individual', 'active',
  TRUE, 0, 'F', FALSE, NOW(), NOW()
);

INSERT INTO verification_records (
  id, user_id, type, status, govt_id_hash, vendor, vendor_reference_id,
  ml_status, review_priority, created_at, updated_at
) VALUES (
  '$VERIFICATION_ID'::uuid,
  '$USER_ID'::uuid,
  'individual',
  'draft',
  'e2e-govt-id-hash',
  'e2e',
  'AADHAAR',
  'idle',
  'manual_review',
  NOW(),
  NOW()
);

INSERT INTO verification_documents (
  id, verification_record_id, document_type, file_url, public_id,
  content_type, created_at
) VALUES
  (
    '$GOVT_DOC_ID'::uuid,
    '$VERIFICATION_ID'::uuid,
    'govt_id',
    '$GOVT_URL_SQL',
    '',
    'image/jpeg',
    NOW()
  ),
  (
    '$SELFIE_DOC_ID'::uuid,
    '$VERIFICATION_ID'::uuid,
    'selfie',
    '$SELFIE_URL_SQL',
    '',
    'image/jpeg',
    NOW()
  );
SQL
then
  fail "failed to seed the cloud database (check DATABASE_URL and the schema)"
fi

SEED_STATE="$(py db-scalar <<SQL
SELECT status || '|' || ml_status
  FROM verification_records
 WHERE id = '$VERIFICATION_ID'::uuid;
SQL
)"
if [[ "$SEED_STATE" != "draft|idle" ]]; then
  fail "seed record is not in draft|idle state (got: $SEED_STATE)"
fi
echo "    seeded: $VERIFICATION_ID ($SEED_STATE)"

echo
echo "==> Generating JWT for test user"
TOKEN="$(py jwt "$USER_ID" "$JWT_ACCESS_SECRET")"
TOKEN="$(printf '%s' "$TOKEN" | tr -d '\r\n')"

if [[ "$TOKEN" != *.*.* ]]; then
  fail "failed to generate a JWT (got: ${TOKEN:0:40}...)"
fi
echo "    token generated (${#TOKEN} chars)"

echo
echo "==> POST /v1/verification/submit"
SUBMIT_BODY="$(mktemp)"
trap 'rm -f "$HELPER" "$SUBMIT_BODY"' EXIT

set +e
HTTP_CODE="$(
  curl -sS -o "$SUBMIT_BODY" -w '%{http_code}' \
    -X POST "$BACKEND_URL/v1/verification/submit" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{}'
)"
CURL_RC=$?
set -e

cat "$SUBMIT_BODY"
echo

if [[ "$CURL_RC" -ne 0 ]]; then
  fail "could not reach the Go backend at $BACKEND_URL (curl exit $CURL_RC)"
fi
if [[ "$HTTP_CODE" != "200" && "$HTTP_CODE" != "201" && "$HTTP_CODE" != "204" ]]; then
  if [[ "$HTTP_CODE" == "401" ]]; then
    echo "    Hint: 401 means the test token and the Go backend use different" >&2
    echo "          JWT_ACCESS_SECRET values. Set E2E_JWT_ACCESS_SECRET, or make" >&2
    echo "          sure E2E_BACKEND_ENV_FILE points at the backend env file." >&2
  fi
  fail "submit returned HTTP $HTTP_CODE"
fi

echo
echo "==> Checking Redis stream $KYC_STREAM (cloud)"
STREAM_LEN="$(py stream-check "$KYC_STREAM" "$VERIFICATION_ID")" \
  || fail "job was not found on the Redis stream"
echo "    stream length: $STREAM_LEN"
echo "    latest entry contains the job payload"

echo
echo "==> Waiting for the ML worker to update the cloud database"
sleep 2

ML_ROW=""
ML_STATUS=""
REVIEW_PRIORITY=""
OCR_CONFIDENCE=""
FACE_SIMILARITY=""
LIVENESS_SCORE=""

for attempt in $(seq 1 30); do
  ML_ROW="$(py db-scalar <<SQL
SELECT ml_status
       || '|' || COALESCE(review_priority, '')
       || '|' || COALESCE(ocr_ocr_result->>'ocr_confidence', '')
       || '|' || COALESCE(face_match_result->>'face_similarity', '')
       || '|' || COALESCE(liveness_result->>'liveness_score', '')
  FROM verification_records
 WHERE id = '$VERIFICATION_ID'::uuid;
SQL
)"

  IFS='|' read -r ML_STATUS REVIEW_PRIORITY OCR_CONFIDENCE FACE_SIMILARITY LIVENESS_SCORE <<<"$ML_ROW"

  if [[ "$ML_STATUS" == "ml_completed" ]] \
    && { [[ -z "$EXPECT_PRIORITY" ]] || [[ "$REVIEW_PRIORITY" == "$EXPECT_PRIORITY" ]]; }; then
    break
  fi
  sleep 1
done

if [[ "$ML_STATUS" != "ml_completed" ]]; then
  fail "expected ml_status=ml_completed, got '$ML_STATUS' (row: $ML_ROW)"
fi
if [[ -n "$EXPECT_PRIORITY" && "$REVIEW_PRIORITY" != "$EXPECT_PRIORITY" ]]; then
  fail "expected review_priority=$EXPECT_PRIORITY, got '$REVIEW_PRIORITY' (row: $ML_ROW)"
fi

echo
echo "✅ E2E PASS (cloud PostgreSQL + Redis)"
echo "   verification_id : $VERIFICATION_ID"
echo "   ml_status       : $ML_STATUS"
echo "   review_priority : $REVIEW_PRIORITY"
echo "   ocr_confidence  : $OCR_CONFIDENCE"
echo "   face_similarity : $FACE_SIMILARITY"
echo "   liveness_score  : $LIVENESS_SCORE"
echo
echo "Inspect the row with your PostgreSQL client:"
echo "  SELECT id, status, ml_status, review_priority,"
echo "         ocr_ocr_result, face_match_result, liveness_result"
echo "    FROM verification_records"
echo "   WHERE id = '$VERIFICATION_ID';"
echo
echo "Full row:"
py inspect "$VERIFICATION_ID"
