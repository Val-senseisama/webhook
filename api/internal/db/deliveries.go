package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func (d *DB) CreateDelivery(ctx context.Context, eventID, endpointID uuid.UUID) (Delivery, error) {
	var del Delivery
	row := d.pool.QueryRow(ctx, `
		INSERT INTO deliveries (event_id, endpoint_id)
		VALUES ($1, $2)
		ON CONFLICT (event_id, endpoint_id) DO NOTHING
		RETURNING id, event_id, endpoint_id, status, attempt_count, next_attempt_at, last_attempt_at, created_at`,
		eventID, endpointID,
	)
	err := row.Scan(&del.ID, &del.EventID, &del.EndpointID, &del.Status, &del.AttemptCount,
		&del.NextAttemptAt, &del.LastAttemptAt, &del.CreatedAt)
	return del, err
}

func (d *DB) GetDeliveryDetail(ctx context.Context, id uuid.UUID) (DeliveryDetail, error) {
	var dd DeliveryDetail
	row := d.pool.QueryRow(ctx, `
		SELECT
			d.id, d.event_id, d.endpoint_id,
			e.url, e.secret, e.timeout_ms, e.max_retries,
			d.attempt_count,
			ev.type, ev.payload
		FROM deliveries d
		JOIN endpoints e  ON e.id  = d.endpoint_id
		JOIN events    ev ON ev.id = d.event_id
		WHERE d.id = $1`, id)
	err := row.Scan(
		&dd.DeliveryID, &dd.EventID, &dd.EndpointID,
		&dd.EndpointURL, &dd.EndpointSecret, &dd.TimeoutMs, &dd.MaxRetries,
		&dd.AttemptCount,
		&dd.EventType, &dd.EventPayload,
	)
	return dd, err
}

func (d *DB) MarkDeliveryInFlight(ctx context.Context, id uuid.UUID) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE deliveries SET status = 'in_flight', last_attempt_at = now(), attempt_count = attempt_count + 1
		 WHERE id = $1`, id)
	return err
}

func (d *DB) MarkDeliverySuccess(ctx context.Context, id uuid.UUID) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE deliveries SET status = 'success', last_attempt_at = now() WHERE id = $1`, id)
	return err
}

func (d *DB) MarkDeliveryFailed(ctx context.Context, id uuid.UUID, nextAttempt time.Time) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE deliveries SET status = 'failed', last_attempt_at = now(), next_attempt_at = $2
		 WHERE id = $1`, id, nextAttempt)
	return err
}

func (d *DB) MarkDeliveryDeadLettered(ctx context.Context, id uuid.UUID) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE deliveries SET status = 'dead_lettered', last_attempt_at = now() WHERE id = $1`, id)
	return err
}

func (d *DB) ResetDeliveryForRetry(ctx context.Context, id uuid.UUID) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE deliveries SET status = 'pending', next_attempt_at = now(), attempt_count = 0
		 WHERE id = $1`, id)
	return err
}

type ListDeliveriesParams struct {
	TenantID   uuid.UUID
	EndpointID *uuid.UUID
	EventID    *uuid.UUID
	Status     string
	Limit      int
	Offset     int
}

func (d *DB) ListDeliveries(ctx context.Context, p ListDeliveriesParams) ([]Delivery, error) {
	if p.Limit == 0 {
		p.Limit = 50
	}
	rows, err := d.pool.Query(ctx, `
		SELECT d.id, d.event_id, d.endpoint_id, d.status, d.attempt_count,
		       d.next_attempt_at, d.last_attempt_at, d.created_at
		FROM deliveries d
		JOIN endpoints e ON e.id = d.endpoint_id
		WHERE e.tenant_id = $1
		  AND ($2::uuid IS NULL OR d.endpoint_id = $2)
		  AND ($3::uuid IS NULL OR d.event_id    = $3)
		  AND ($4 = '' OR d.status = $4)
		ORDER BY d.created_at DESC
		LIMIT $5 OFFSET $6`,
		p.TenantID, p.EndpointID, p.EventID, p.Status, p.Limit, p.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []Delivery
	for rows.Next() {
		var del Delivery
		if err := rows.Scan(&del.ID, &del.EventID, &del.EndpointID, &del.Status, &del.AttemptCount,
			&del.NextAttemptAt, &del.LastAttemptAt, &del.CreatedAt); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, del)
	}
	return deliveries, rows.Err()
}

type RecordAttemptParams struct {
	DeliveryID      uuid.UUID
	AttemptNumber   int
	RequestHeaders  []byte
	RequestBody     string
	ResponseStatus  *int
	ResponseHeaders []byte
	ResponseBody    *string
	DurationMs      *int
	Error           *string
}

func (d *DB) RecordAttempt(ctx context.Context, p RecordAttemptParams) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO delivery_attempts
			(delivery_id, attempt_number, request_headers, request_body,
			 response_status, response_headers, response_body, duration_ms, error)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.DeliveryID, p.AttemptNumber, p.RequestHeaders, p.RequestBody,
		p.ResponseStatus, p.ResponseHeaders, p.ResponseBody, p.DurationMs, p.Error,
	)
	return err
}

func (d *DB) ListAttempts(ctx context.Context, deliveryID uuid.UUID) ([]DeliveryAttempt, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, delivery_id, attempt_number, request_headers, request_body,
		       response_status, response_headers, response_body, duration_ms, error, created_at
		FROM delivery_attempts
		WHERE delivery_id = $1
		ORDER BY attempt_number ASC`, deliveryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []DeliveryAttempt
	for rows.Next() {
		var a DeliveryAttempt
		if err := rows.Scan(&a.ID, &a.DeliveryID, &a.AttemptNumber, &a.RequestHeaders,
			&a.RequestBody, &a.ResponseStatus, &a.ResponseHeaders, &a.ResponseBody,
			&a.DurationMs, &a.Error, &a.CreatedAt); err != nil {
			return nil, err
		}
		attempts = append(attempts, a)
	}
	return attempts, rows.Err()
}
