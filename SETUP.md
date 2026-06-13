# Setup Guide

## 1. Install Supabase CLI

```bash
# npm
npm install -g supabase

# or Homebrew
brew install supabase/tap/supabase
```

## 2. Create a Supabase project

Go to https://supabase.com/dashboard → New project. Note your:
- Project ref (e.g. `abcdefghijklmnop`)
- Database password
- Region

## 3. Login and link

```bash
supabase login                              # opens browser
supabase link --project-ref <your-ref>     # links this directory
```

## 4. Push migrations (creates all tables + seeds default tenant)

```bash
supabase db push
```

This runs both migrations:
- `20260613000001_initial.sql` — all tables, indexes, RLS
- `20260613000002_seed.sql` — default tenant row

## 5. Copy connection strings into .env

From Supabase dashboard → Settings → Database → Connection string:

```bash
cp .env.example .env
```

Edit `.env`:

```
# Transaction mode (port 6543) — API handlers
DATABASE_POOLER_URL=postgresql://postgres.[ref]:[password]@aws-0-[region].pooler.supabase.com:6543/postgres?sslmode=require

# Direct (port 5432) — River workers + keygen
DATABASE_DIRECT_URL=postgresql://postgres.[ref]:[password]@db.[ref].supabase.co:5432/postgres?sslmode=require

DEFAULT_TENANT_ID=00000000-0000-0000-0000-000000000001
```

## 6. Generate your first API key

```bash
make keygen NAME="local-dev"
```

Output:
```
✓ API key created

  Key ID   : <uuid>
  Name     : local-dev
  Tenant   : 00000000-0000-0000-0000-000000000001
  Raw key  : whk_<64 hex chars>

  ⚠  This is the only time the raw key is shown. Store it now.

  Usage:
    Authorization: Bearer whk_<64 hex chars>
```

Copy the raw key into `.env`:
```
NEXT_PUBLIC_API_KEY=whk_<your-key>
```

## 7. Start everything

```bash
make dev-api        # Go API on :8080 (runs migrations on startup)
make dev-worker     # River workers
make install        # npm install for dashboard (first time)
make dev-dash       # Next.js dashboard on :3000
```

## 8. Send a test event

```bash
curl -X POST http://localhost:8080/ingest/test \
  -H "X-Event-Type: order.created" \
  -H "Content-Type: application/json" \
  -d '{"order_id": "ord_001", "amount": 4999}'

# → {"event_id":"<uuid>"}
```

## 9. Register a webhook endpoint

```bash
curl -X POST http://localhost:8080/v1/endpoints \
  -H "Authorization: Bearer whk_<your-key>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-service",
    "url":  "https://webhook.site/<your-id>"
  }'
```

Then subscribe it to all events:

```bash
curl -X POST http://localhost:8080/v1/endpoints/<endpoint-id>/subscriptions \
  -H "Authorization: Bearer whk_<your-key>" \
  -H "Content-Type: application/json" \
  -d '{"event_types": ["*"]}'
```

## Supabase Realtime (optional — live dashboard updates)

In the Supabase dashboard → Table Editor → `deliveries` → Enable Realtime.

Add to `.env`:
```
NEXT_PUBLIC_SUPABASE_URL=https://<ref>.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=<anon-key>
```

## Managing API keys via the REST API

```bash
# List all keys (IDs and names — raw key is never returned after creation)
curl http://localhost:8080/v1/apikeys \
  -H "Authorization: Bearer whk_<your-key>"

# Create a new key
curl -X POST http://localhost:8080/v1/apikeys \
  -H "Authorization: Bearer whk_<your-key>" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-pipeline"}'
# → {"id":"...","name":"ci-pipeline","created_at":"...","key":"whk_<raw — shown once>"}

# Delete a key by ID
curl -X DELETE http://localhost:8080/v1/apikeys/<key-id> \
  -H "Authorization: Bearer whk_<different-key>"
# Note: you cannot delete the key you are currently authenticated with.
# Generate a replacement first, then delete the old one.
```
