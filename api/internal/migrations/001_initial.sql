-- +goose Up

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    key_hash    TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS events (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source           TEXT NOT NULL,
    type             TEXT NOT NULL,
    payload          JSONB NOT NULL,
    headers          JSONB,
    idempotency_key  TEXT,
    status           TEXT NOT NULL DEFAULT 'received'
                         CHECK (status IN ('received','processing','delivered','failed','dead_lettered')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_events_tenant_type    ON events (tenant_id, type);
CREATE INDEX IF NOT EXISTS idx_events_tenant_created ON events (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_status         ON events (status) WHERE status NOT IN ('delivered');

CREATE TABLE IF NOT EXISTS endpoints (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    url          TEXT NOT NULL,
    secret       TEXT NOT NULL,
    enabled      BOOLEAN NOT NULL DEFAULT true,
    timeout_ms   INT NOT NULL DEFAULT 30000,
    max_retries  INT NOT NULL DEFAULT 5,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_endpoints_tenant ON endpoints (tenant_id);

CREATE TABLE IF NOT EXISTS subscriptions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id  UUID NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    event_types  TEXT[] NOT NULL DEFAULT '{"*"}',
    filter_rules JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS deliveries (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id         UUID NOT NULL REFERENCES events(id),
    endpoint_id      UUID NOT NULL REFERENCES endpoints(id),
    status           TEXT NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending','in_flight','success','failed','dead_lettered')),
    attempt_count    INT NOT NULL DEFAULT 0,
    next_attempt_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_attempt_at  TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, endpoint_id)
);

CREATE INDEX IF NOT EXISTS idx_deliveries_pending  ON deliveries (next_attempt_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_deliveries_endpoint ON deliveries (endpoint_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_deliveries_event    ON deliveries (event_id);

CREATE TABLE IF NOT EXISTS delivery_attempts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id      UUID NOT NULL REFERENCES deliveries(id),
    attempt_number   INT NOT NULL,
    request_headers  JSONB,
    request_body     TEXT,
    response_status  INT,
    response_headers JSONB,
    response_body    TEXT,
    duration_ms      INT,
    error            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_delivery_attempts_delivery ON delivery_attempts (delivery_id, attempt_number);

ALTER TABLE events            ENABLE ROW LEVEL SECURITY;
ALTER TABLE endpoints         ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions     ENABLE ROW LEVEL SECURITY;
ALTER TABLE deliveries        ENABLE ROW LEVEL SECURITY;
ALTER TABLE delivery_attempts ENABLE ROW LEVEL SECURITY;

-- +goose Down

DROP TABLE IF EXISTS delivery_attempts;
DROP TABLE IF EXISTS deliveries;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS endpoints;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS tenants;
