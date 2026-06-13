package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type CreateEventParams struct {
	TenantID       uuid.UUID
	Source         string
	Type           string
	Payload        json.RawMessage
	Headers        json.RawMessage
	IdempotencyKey string
}

func (d *DB) CreateEvent(ctx context.Context, p CreateEventParams) (Event, error) {
	var e Event
	row := d.pool.QueryRow(ctx, `
		INSERT INTO events (tenant_id, source, type, payload, headers, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		RETURNING id, tenant_id, source, type, payload, headers, COALESCE(idempotency_key,''), status, created_at`,
		p.TenantID, p.Source, p.Type, p.Payload, p.Headers, p.IdempotencyKey,
	)
	err := row.Scan(&e.ID, &e.TenantID, &e.Source, &e.Type, &e.Payload, &e.Headers,
		&e.IdempotencyKey, &e.Status, &e.CreatedAt)
	return e, err
}

func (d *DB) GetEvent(ctx context.Context, id uuid.UUID) (Event, error) {
	var e Event
	row := d.pool.QueryRow(ctx, `
		SELECT id, tenant_id, source, type, payload, headers, COALESCE(idempotency_key,''), status, created_at
		FROM events WHERE id = $1`, id)
	err := row.Scan(&e.ID, &e.TenantID, &e.Source, &e.Type, &e.Payload, &e.Headers,
		&e.IdempotencyKey, &e.Status, &e.CreatedAt)
	return e, err
}

type ListEventsParams struct {
	TenantID uuid.UUID
	Type     string
	Source   string
	Limit    int
	Offset   int
}

func (d *DB) ListEvents(ctx context.Context, p ListEventsParams) ([]Event, error) {
	if p.Limit == 0 {
		p.Limit = 50
	}
	rows, err := d.pool.Query(ctx, `
		SELECT id, tenant_id, source, type, payload, headers, COALESCE(idempotency_key,''), status, created_at
		FROM events
		WHERE tenant_id = $1
		  AND ($2 = '' OR type = $2)
		  AND ($3 = '' OR source = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5`,
		p.TenantID, p.Type, p.Source, p.Limit, p.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.TenantID, &e.Source, &e.Type, &e.Payload, &e.Headers,
			&e.IdempotencyKey, &e.Status, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (d *DB) UpdateEventStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE events SET status = $2 WHERE id = $1`, id, status)
	return err
}
