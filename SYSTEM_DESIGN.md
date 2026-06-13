# Webhook System — Production Design

## Overview

A multi-tenant webhook platform that receives events from any source, fans them out to registered subscriber endpoints, and guarantees at-least-once delivery with full observability.

---

## Core Concepts

| Term | Definition |
|---|---|
| **Event** | A typed JSON payload ingested from a producer (e.g. `order.created`) |
| **Endpoint** | A subscriber's HTTPS URL registered to receive events |
| **Subscription** | A mapping between event types and an endpoint (with optional filters) |
| **Delivery** | One attempt to POST an event to an endpoint |
| **Attempt** | A single HTTP request within a delivery (retried on failure) |

---

## Architecture

```
Producer
  │
  ▼
┌─────────────────────────────────────────────────┐
│  Ingestion Layer                                 │
│  POST /ingest/:source                            │
│  • Authenticate source (HMAC / API key)          │
│  • Validate payload schema                       │
│  • Persist Event to DB                           │
│  • Enqueue fan-out job → Queue                   │
│  • Return 202 Accepted immediately               │
└──────────────────────┬──────────────────────────┘
                       │
                       ▼
              ┌────────────────┐
              │   Event Queue  │  (durable, at-least-once)
              └───────┬────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────┐
│  Fan-out Worker                                  │
│  • Load all matching Subscriptions               │
│  • Create one Delivery record per endpoint       │
│  • Enqueue individual Delivery jobs              │
└──────────────────────┬──────────────────────────┘
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
   ┌────────────┐ ┌────────────┐ ┌────────────┐
   │ Delivery   │ │ Delivery   │ │ Delivery   │
   │ Worker     │ │ Worker     │ │ Worker     │
   │ (endpoint A│ │ (endpoint B│ │ (endpoint C│
   └─────┬──────┘ └─────┬──────┘ └─────┬──────┘
         │              │              │
         ▼              ▼              ▼
   POST to URL     POST to URL   POST to URL
   + HMAC sig      + HMAC sig    + HMAC sig
```

---

## Data Model

### `events`
```sql
id            UUID PRIMARY KEY
source        TEXT NOT NULL          -- producer identifier
type          TEXT NOT NULL          -- e.g. "order.created"
payload       JSONB NOT NULL
headers       JSONB                  -- raw inbound headers for passthrough
idempotency_key TEXT UNIQUE          -- dedup key from producer
status        ENUM(received, processing, delivered, failed, dead_lettered)
created_at    TIMESTAMPTZ DEFAULT now()
```

### `endpoints`
```sql
id            UUID PRIMARY KEY
tenant_id     UUID NOT NULL REFERENCES tenants
name          TEXT NOT NULL
url           TEXT NOT NULL
secret        TEXT NOT NULL          -- used to sign outgoing payloads (HMAC-SHA256)
enabled       BOOLEAN DEFAULT true
timeout_ms    INT DEFAULT 30000
max_retries   INT DEFAULT 5
rate_limit_rps INT DEFAULT 100
created_at    TIMESTAMPTZ
updated_at    TIMESTAMPTZ
```

### `subscriptions`
```sql
id            UUID PRIMARY KEY
endpoint_id   UUID NOT NULL REFERENCES endpoints
event_types   TEXT[]                 -- glob patterns e.g. "order.*", "*"
filter_rules  JSONB                  -- JSONPath filters on payload
created_at    TIMESTAMPTZ
```

### `deliveries`
```sql
id            UUID PRIMARY KEY
event_id      UUID NOT NULL REFERENCES events
endpoint_id   UUID NOT NULL REFERENCES endpoints
status        ENUM(pending, in_flight, success, failed, dead_lettered)
attempt_count INT DEFAULT 0
next_attempt_at TIMESTAMPTZ
last_attempt_at TIMESTAMPTZ
created_at    TIMESTAMPTZ
```

### `delivery_attempts`
```sql
id            UUID PRIMARY KEY
delivery_id   UUID NOT NULL REFERENCES deliveries
attempt_number INT NOT NULL
request_headers  JSONB
request_body     TEXT
response_status  INT
response_headers JSONB
response_body    TEXT
duration_ms      INT
error            TEXT
created_at       TIMESTAMPTZ
```

---

## Retry Strategy

Exponential backoff with jitter. Max 5 attempts by default (configurable per endpoint).

```
Attempt 1 → immediate
Attempt 2 → 30s  + jitter
Attempt 3 → 5m   + jitter
Attempt 4 → 30m  + jitter
Attempt 5 → 2h   + jitter
→ Dead letter queue
```

A delivery is considered successful on **any 2xx** response. Non-2xx, connection error, and timeout all trigger a retry. Responses > 30s (configurable) are treated as timeouts.

---

## Security

### Inbound (Producer → Ingestion)
- API key per producer, stored hashed (SHA-256) in DB
- Optional: HMAC-SHA256 signature on request body (`X-Webhook-Signature-256`)
- TLS required; reject non-HTTPS in production
- Rate limit per producer key (token bucket)

### Outbound (Delivery Worker → Subscriber)
Every delivery is signed so subscribers can verify authenticity:

```
X-Webhook-Id: <delivery_id>
X-Webhook-Timestamp: <unix_seconds>
X-Webhook-Signature-256: sha256=HMAC(secret, "<id>.<timestamp>.<body>")
```

Subscribers validate:
1. Parse `X-Webhook-Timestamp` — reject if > 5 minutes old
2. Recompute HMAC with their stored secret
3. Compare signatures with constant-time comparison

### Secrets
- Endpoint secrets stored encrypted at rest (AES-256-GCM, envelope key in KMS)
- Secret rotation supported without downtime (dual-validation window)

---

## API Surface

### Ingestion
```
POST /ingest/:source
Authorization: Bearer <api_key>
Content-Type: application/json

→ 202 { event_id: "..." }
→ 400 invalid payload
→ 401 bad key
→ 409 duplicate idempotency_key
→ 429 rate limited
```

### Management API (authenticated, tenant-scoped)
```
# Endpoints
GET    /v1/endpoints
POST   /v1/endpoints
GET    /v1/endpoints/:id
PATCH  /v1/endpoints/:id
DELETE /v1/endpoints/:id
POST   /v1/endpoints/:id/rotate-secret

# Subscriptions
GET    /v1/endpoints/:id/subscriptions
POST   /v1/endpoints/:id/subscriptions
DELETE /v1/endpoints/:id/subscriptions/:sub_id

# Events
GET    /v1/events?type=&source=&from=&to=&limit=
GET    /v1/events/:id

# Deliveries
GET    /v1/deliveries?endpoint_id=&status=&event_id=
GET    /v1/deliveries/:id
GET    /v1/deliveries/:id/attempts
POST   /v1/deliveries/:id/retry    -- force immediate retry
POST   /v1/events/:id/redeliver    -- create new deliveries for all subs
```

---

## Queue Design

Two logical queues:

**`webhook.fanout`** — Low volume, high priority. One message per event. Fan-out worker creates Delivery records and enqueues to `webhook.delivery`.

**`webhook.delivery`** — High volume. One message per Delivery attempt. Workers are horizontally scaled. Visibility timeout = endpoint timeout + buffer. On failure: message returned to queue with delay matching retry schedule.

Use a **dead letter queue** after max retries; alerts fire on DLQ depth.

For local/small deployments: PostgreSQL-backed queue (SELECT ... FOR UPDATE SKIP LOCKED pattern) works well and avoids an external dependency.

```sql
-- Poll pending deliveries
SELECT * FROM deliveries
WHERE status = 'pending'
  AND next_attempt_at <= now()
ORDER BY next_attempt_at
LIMIT 50
FOR UPDATE SKIP LOCKED;
```

---

## Observability

### Metrics (exposed via `/metrics` — Prometheus format)
```
webhook_events_ingested_total{source, type}
webhook_events_fanout_duration_seconds{source, type}
webhook_deliveries_total{status, endpoint_id}
webhook_delivery_duration_seconds{endpoint_id}
webhook_delivery_attempt_duration_seconds{attempt_number, endpoint_id}
webhook_queue_depth{queue}
webhook_dlq_depth
```

### Structured Logs
Every ingestion and delivery attempt emits a JSON log line:
```json
{
  "ts": "2026-06-12T10:00:00Z",
  "level": "info",
  "event": "delivery.attempt",
  "delivery_id": "...",
  "event_id": "...",
  "endpoint_id": "...",
  "attempt": 2,
  "status": 200,
  "duration_ms": 142,
  "tenant_id": "..."
}
```

### Traces
Distributed trace spans:
- `ingest` → `fanout` → `delivery.attempt`
- Propagate `traceparent` header to subscriber endpoints

### Alerts
| Condition | Severity |
|---|---|
| DLQ depth > 0 | Warning |
| DLQ depth > 100 | Critical |
| Delivery success rate < 95% (5m window) | Warning |
| P99 delivery latency > 10s | Warning |
| Ingestion error rate > 1% | Critical |
| Queue lag > 5m | Critical |

---

## Multi-Tenancy

- Every resource is scoped to `tenant_id`
- Row-level security in Postgres enforces tenant isolation
- Rate limits, delivery quotas, and max endpoints are per-tenant (plan-based)
- Tenant secrets (for KMS envelope encryption) are isolated

---

## Scalability

| Layer | Scaling Approach |
|---|---|
| Ingestion API | Stateless — horizontal scale behind load balancer |
| Fan-out workers | Autoscale on queue depth |
| Delivery workers | Autoscale on queue depth, one pool per tenant tier |
| Database | Primary + read replicas; partition `delivery_attempts` by month |
| Queue | Managed queue service (SQS, Pub/Sub, Vercel Queues) |

**Throughput targets (single region):**
- Ingestion: 10,000 events/sec
- Fan-out: 50,000 delivery jobs/sec
- Delivery: 5,000 concurrent HTTP requests

---

## Failure Modes & Mitigations

| Failure | Mitigation |
|---|---|
| Subscriber endpoint down | Retry with backoff → DLQ; customer notified |
| Delivery worker crash | Job visibility timeout returns job to queue |
| Database overload | Queue absorbs burst; workers self-throttle |
| Fan-out produces duplicate deliveries | Idempotency key on Delivery (event_id + endpoint_id) |
| Producer sends duplicate event | `idempotency_key` dedup at ingestion |
| Clock skew on signatures | 5-minute tolerance window on timestamp check |
| Network partition to subscriber | Timeout + retry; circuit breaker after sustained failure |

---

## Circuit Breaker (per endpoint)

Track consecutive failures per endpoint. After **10 consecutive failures** in a 10-minute window:
1. Open circuit — stop attempting delivery, queue jobs are held
2. After 10 minutes, send a single **probe request**
3. On probe success: close circuit and resume
4. On probe failure: reset open timer (exponential extension, max 1h)

Endpoint owners are notified by email/webhook on circuit open.

---

## Dashboard

Pages:
- **Overview** — event rate, delivery success rate, active endpoints, DLQ depth
- **Endpoints** — list, create, edit, rotate secret, enable/disable
- **Event Log** — searchable by type, source, time range; inspect raw payload
- **Delivery Log** — filter by endpoint/status; view all attempts, request/response bodies
- **Dead Letter Queue** — inspect and redeliver failed events

---

## Deployment Topology

```
                    ┌─────────────┐
                    │   CDN/WAF   │
                    └──────┬──────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
   │  Ingestion  │  │  Mgmt API   │  │  Dashboard  │
   │  Service    │  │  Service    │  │  (Next.js)  │
   └──────┬──────┘  └──────┬──────┘  └─────────────┘
          │                │
          ▼                ▼
   ┌──────────────────────────┐
   │        Message Queue     │
   └────────────┬─────────────┘
                │
        ┌───────┴────────┐
        ▼                ▼
  ┌───────────┐   ┌─────────────┐
  │  Fan-out  │   │  Delivery   │
  │  Workers  │   │  Workers    │
  └─────┬─────┘   └──────┬──────┘
        │                │
        └───────┬─────────┘
                ▼
        ┌───────────────┐
        │  PostgreSQL   │
        │  (primary +   │
        │   replicas)   │
        └───────────────┘
```

---

## Tech Choices — Selected Stack (Option B: Go API + Next.js Dashboard)

| Concern | Choice | Reason |
|---|---|---|
| API runtime | **Go 1.22** | Goroutine-per-delivery: 50k concurrent HTTP requests on one box vs 5k with Node |
| HTTP router | **chi** | Lightweight, idiomatic, great middleware story |
| Database | **Supabase (PostgreSQL)** | Managed Postgres + RLS for tenancy + Realtime for live dashboard |
| DB driver | **pgx/v5** | Best Go Postgres driver; pipeline mode for write batching |
| Job queue | **River** | Postgres-backed queue on pgx; advisory locks, retries, web UI — no extra infra |
| Migrations | **goose** | SQL-first migration files, embedded into binary |
| Crypto | Go `crypto/hmac` (stdlib) | HMAC-SHA256, no external dep |
| Validation | **encoding/json** + manual | Keeps binary small; Zod-style libs not idiomatic in Go |
| Observability | **slog** (stdlib) + OpenTelemetry | JSON structured logs; OTel for traces |
| Dashboard | **Next.js 15** (App Router) | React Server Components for initial load, Supabase Realtime for live updates |
| Dashboard styling | **Tailwind v4** + custom CSS vars | Industrial dark theme, Geist Mono font |
| Dashboard data | Supabase JS client (Realtime) | Live delivery status without polling |
| Auth | API keys (hashed SHA-256 in DB) | Keys for producers; Supabase Auth for dashboard users |

## Connection Strategy

```
API process  →  PgBouncer pooler (port 6543)   — short-lived, burst connections
               + direct connection (port 5432)  — River client (advisory locks need session mode)

Worker process → direct connection (port 5432)  — River workers (session-mode advisory locks)
```

## Project Layout

```
webhook/
├── api/                          # Go backend
│   ├── cmd/
│   │   ├── api/main.go           # HTTP server (ingest + management API)
│   │   └── worker/main.go        # River worker process
│   ├── internal/
│   │   ├── config/config.go      # Env-based config
│   │   ├── db/                   # pgx query layer (no ORM)
│   │   │   ├── db.go             # Pool setup + DB struct
│   │   │   ├── events.go
│   │   │   ├── endpoints.go
│   │   │   ├── subscriptions.go
│   │   │   ├── deliveries.go
│   │   │   └── apikeys.go
│   │   ├── signing/hmac.go       # HMAC sign + verify
│   │   ├── middleware/auth.go    # API key auth middleware
│   │   ├── handlers/             # HTTP handlers
│   │   │   ├── ingest.go
│   │   │   ├── endpoints.go
│   │   │   ├── events.go
│   │   │   └── deliveries.go
│   │   └── jobs/                 # River job definitions
│   │       ├── fanout.go
│   │       └── delivery.go
│   ├── migrations/001_initial.sql
│   └── go.mod
├── dashboard/                    # Next.js 15 frontend
│   ├── app/
│   │   ├── layout.tsx
│   │   ├── globals.css
│   │   ├── page.tsx              # Overview
│   │   ├── endpoints/page.tsx
│   │   ├── events/page.tsx
│   │   └── deliveries/page.tsx
│   ├── components/
│   │   ├── sidebar.tsx
│   │   ├── status-badge.tsx
│   │   └── data-table.tsx
│   └── lib/
│       ├── api.ts
│       └── types.ts
├── docker-compose.yml
├── Makefile
└── .env.example
```

## Throughput (Revised with Go)

| Scale | Events/sec | Bottleneck |
|---|---|---|
| Starter | < 500 | Nothing |
| Growth | 500–5k | Postgres write throughput |
| Scale | 5k–20k | Postgres + write batching needed |
| High | 20k–50k | Add Kafka ingest tier; River stays |

Go shifts the ceiling: the application layer is no longer the bottleneck. Supabase Pro handles ~5k writes/sec. Above that, batch event inserts or move to Supabase Enterprise.

---

## Open Questions / Tradeoffs

1. **Ordering guarantees** — Do events need to be delivered in order per endpoint? If yes, use per-endpoint FIFO queues (more expensive, lower throughput).

2. **Payload size limit** — Default 256KB. Large payloads could be stored in object storage with a signed URL sent in the webhook instead.

3. **Filtering** — JSONPath filters on subscription allow subscribers to receive only relevant events; adds CPU cost at fan-out time.

4. **Replay** — Should historical events be replayable? Requires long event retention (and storage cost) vs. DLQ-only recovery.

5. **Push vs. Pull** — This design is push-based. A pull-based alternative (subscribers poll an event stream) is simpler to implement and eliminates retry complexity, but shifts latency burden to subscribers.
