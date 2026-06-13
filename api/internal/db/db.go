package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

// ---- shared model types ----

type Event struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	Source         string          `json:"source"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	Headers        json.RawMessage `json:"headers,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
}

type Endpoint struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Secret     string    `json:"secret"`
	Enabled    bool      `json:"enabled"`
	TimeoutMs  int       `json:"timeout_ms"`
	MaxRetries int       `json:"max_retries"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Subscription struct {
	ID          uuid.UUID       `json:"id"`
	EndpointID  uuid.UUID       `json:"endpoint_id"`
	EventTypes  []string        `json:"event_types"`
	FilterRules json.RawMessage `json:"filter_rules,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Delivery struct {
	ID            uuid.UUID  `json:"id"`
	EventID       uuid.UUID  `json:"event_id"`
	EndpointID    uuid.UUID  `json:"endpoint_id"`
	Status        string     `json:"status"`
	AttemptCount  int        `json:"attempt_count"`
	NextAttemptAt time.Time  `json:"next_attempt_at"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type DeliveryAttempt struct {
	ID              uuid.UUID       `json:"id"`
	DeliveryID      uuid.UUID       `json:"delivery_id"`
	AttemptNumber   int             `json:"attempt_number"`
	RequestHeaders  json.RawMessage `json:"request_headers,omitempty"`
	RequestBody     string          `json:"request_body,omitempty"`
	ResponseStatus  *int            `json:"response_status,omitempty"`
	ResponseHeaders json.RawMessage `json:"response_headers,omitempty"`
	ResponseBody    *string         `json:"response_body,omitempty"`
	DurationMs      *int            `json:"duration_ms,omitempty"`
	Error           *string         `json:"error,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

type APIKey struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	KeyHash   string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// DeliveryDetail joins delivery + event + endpoint for workers.
type DeliveryDetail struct {
	DeliveryID     uuid.UUID
	EventID        uuid.UUID
	EndpointID     uuid.UUID
	EndpointURL    string
	EndpointSecret string
	TimeoutMs      int
	MaxRetries     int
	AttemptCount   int
	EventType      string
	EventPayload   json.RawMessage
}

// ping verifies connectivity on startup.
func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}
