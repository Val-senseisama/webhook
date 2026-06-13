# Webhook Delivery System

A production-grade webhook delivery platform built from scratch. Events are ingested, fanned out to subscribed endpoints, delivered with HMAC-signed payloads, and retried with exponential backoff — all backed by a Postgres job queue with no external message broker.

**Stack:** Go · Supabase (Postgres) · River · Next.js 15

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  Ingest                                                             │
│  POST /ingest/:source                                               │
│  No auth — write-only. Idempotency-Key deduplication.              │
└────────────────────┬────────────────────────────────────────────────┘
                     │ INSERT event + enqueue FanoutJob
                     ▼
┌─────────────────────────────────────────────────────────────────────┐
│  FanoutWorker  (River, "default" queue, 50 workers)                 │
│                                                                     │
│  1. Load subscribed endpoints matching event type or "*"            │
│  2. INSERT delivery row per endpoint (UNIQUE → idempotent)         │
│  3. Enqueue DeliveryJob per delivery                               │
└────────────────────┬────────────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────────────┐
│  DeliveryWorker  (River, "delivery" queue, 500 workers)             │
│                                                                     │
│  1. Sign payload: HMAC-SHA256(secret, deliveryID.timestamp.body)   │
│  2. POST to endpoint URL with signature headers                     │
│  3. Record attempt (status, response, duration)                     │
│  4. On 2xx → mark success                                          │
│  5. On failure → River retries: 30s · 5m · 30m · 2h              │
│  6. After max_retries → dead-letter                                 │
└─────────────────────────────────────────────────────────────────────┘

 Postgres (Supabase)     River job queue         pgx/v5
 ├── tenants             ├── river_job            ├── pooler pool (port 6543)
 ├── api_keys            ├── river_queue          │   transaction mode
 ├── events              └── river_leader         │   simple protocol (*)
 ├── endpoints                                    └── direct pool (port 5432)
 ├── subscriptions                                    session mode
 ├── deliveries                                       advisory locks
 └── delivery_attempts

(*) Supabase's transaction-mode pooler can't share prepared statements
    across connections — pgx must use simple protocol on port 6543.
```

---

## Design Decisions

**Why River instead of a Redis queue?**
River is a Postgres-backed job queue. No extra infrastructure — the jobs live in the same database as the events and delivery records, so fanout + delivery creation is a single logical transaction. Failed jobs are retried by River's scheduler, not by polling.

**Two connection pools, not one**
River requires session-mode connections for advisory locks (used for leader election and queue polling). Supabase's default pooler is transaction-mode — it breaks advisory locks. Solution: two pools from the same process:
- Port **6543** (transaction mode) → API handlers, short-lived queries
- Port **5432** (session mode) → River client and worker process

**Why simple protocol on the transaction-mode pool?**
pgx v5 caches named prepared statements on server connections. In transaction mode, the same backend connection is shared across clients — a prepared statement created by one client is invisible to another but blocks re-creation. `QueryExecModeSimpleProtocol` disables the cache on the pooler pool.

**HMAC-SHA256 signing**
Every delivery is signed with the endpoint's secret:
```
X-Webhook-Signature-256: sha256=HMAC(secret, deliveryID.timestamp.body)
```
Consumers verify the signature to reject replays and spoofed payloads. The secret rotates independently of the endpoint URL via `POST /v1/endpoints/:id/rotate-secret`.

**API key security**
Raw keys are generated as `whk_<64 random hex>`. Only the SHA-256 hash is stored. The raw key is shown exactly once at creation — never retrievable after that. You cannot delete the key you're currently authenticated with (requires generating a replacement first).

**Multi-tenant schema, single-tenant surface**
Every table has `tenant_id`. RLS is enabled. The API key bearer auth resolves which tenant you are — so expanding to multi-tenant SaaS requires only an onboarding flow, not a schema change.

---

## Project Layout

```
webhook/
├── api/
│   ├── cmd/
│   │   ├── api/        HTTP server (chi router, goose migrations, River insert-only)
│   │   ├── worker/     River workers (FanoutWorker + DeliveryWorker)
│   │   └── keygen/     CLI: generate + register an API key
│   └── internal/
│       ├── config/     env-based config
│       ├── db/         pgx query layer (no ORM)
│       ├── handlers/   HTTP handlers (endpoints, events, deliveries, apikeys)
│       ├── jobs/       River workers + custom retry policy
│       ├── middleware/ Bearer token auth
│       ├── migrations/ Embedded goose SQL migrations
│       └── signing/    HMAC-SHA256 signing + API key hashing
├── dashboard/          Next.js 15 App Router dashboard
└── supabase/           Supabase project config + migrations
```

---

## Running Locally

**Prerequisites:** Go 1.22+, Node 20+, a Supabase project

```bash
# 1. Copy and fill in connection strings
cp .env.example .env

# 2. Push DB schema
supabase link --project-ref <ref>
supabase db push

# 3. Generate an admin API key (shown once — copy it into .env)
make keygen NAME="local-dev"

# 4. Start everything
make dev-api      # Go API on :8080  (Terminal 1)
make dev-worker   # River workers    (Terminal 2)
make install      # npm install (first time)
make dev-dash     # Dashboard on :3000 (Terminal 3)
```

---

## API Reference

### Ingest (no auth)

```bash
POST /ingest/:source
X-Event-Type: order.created
Idempotency-Key: evt_abc123   # optional dedup key
Content-Type: application/json

{"order_id": "ord_001", "amount": 4999}
# → {"event_id": "<uuid>"}
```

### Endpoints

```bash
GET    /v1/endpoints
POST   /v1/endpoints          {"name":"my-svc","url":"https://..."}
GET    /v1/endpoints/:id
PATCH  /v1/endpoints/:id
DELETE /v1/endpoints/:id
POST   /v1/endpoints/:id/rotate-secret

GET    /v1/endpoints/:id/subscriptions
POST   /v1/endpoints/:id/subscriptions  {"event_types":["order.*","*"]}
DELETE /v1/endpoints/:id/subscriptions/:subID
```

### Events & Deliveries

```bash
GET /v1/events?type=order.created&source=checkout&limit=50
GET /v1/events/:id

GET /v1/deliveries?status=in_flight&endpoint_id=...&limit=100
GET /v1/deliveries/:id
GET /v1/deliveries/:id/attempts
```

### API Keys

```bash
GET    /v1/apikeys
POST   /v1/apikeys   {"name":"ci-pipeline"}
       # → {"id":"...","key":"whk_..."}  ← raw key shown once
DELETE /v1/apikeys/:id   # cannot delete the key you're authenticated with
```

All `/v1/` routes require `Authorization: Bearer whk_<key>`.

---

## Payload Verification

```python
import hmac, hashlib

def verify(secret: str, delivery_id: str, timestamp: str, body: bytes, sig: str) -> bool:
    msg = f"{delivery_id}.{timestamp}.".encode() + body
    expected = "sha256=" + hmac.new(secret.encode(), msg, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, sig)
```

```go
import "crypto/hmac"; "crypto/sha256"

func Verify(secret, deliveryID, timestamp string, body []byte, sig string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    fmt.Fprintf(mac, "%s.%s.", deliveryID, timestamp)
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(sig))
}
```
