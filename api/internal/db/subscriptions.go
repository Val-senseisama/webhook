package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type CreateSubscriptionParams struct {
	EndpointID  uuid.UUID
	EventTypes  []string
	FilterRules json.RawMessage
}

func (d *DB) CreateSubscription(ctx context.Context, p CreateSubscriptionParams) (Subscription, error) {
	var s Subscription
	row := d.pool.QueryRow(ctx, `
		INSERT INTO subscriptions (endpoint_id, event_types, filter_rules)
		VALUES ($1, $2, $3)
		RETURNING id, endpoint_id, event_types, filter_rules, created_at`,
		p.EndpointID, p.EventTypes, p.FilterRules,
	)
	err := row.Scan(&s.ID, &s.EndpointID, &s.EventTypes, &s.FilterRules, &s.CreatedAt)
	return s, err
}

func (d *DB) ListSubscriptions(ctx context.Context, endpointID uuid.UUID) ([]Subscription, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, endpoint_id, event_types, filter_rules, created_at
		FROM subscriptions WHERE endpoint_id = $1 ORDER BY created_at DESC`, endpointID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.EndpointID, &s.EventTypes, &s.FilterRules, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (d *DB) DeleteSubscription(ctx context.Context, id, endpointID uuid.UUID) error {
	_, err := d.pool.Exec(ctx,
		`DELETE FROM subscriptions WHERE id = $1 AND endpoint_id = $2`, id, endpointID)
	return err
}

// GetMatchingEndpoints returns all enabled endpoints subscribed to the given event type
// within the same tenant. Supports wildcard "*" event_type subscriptions.
func (d *DB) GetMatchingEndpoints(ctx context.Context, tenantID uuid.UUID, eventType string) ([]Endpoint, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT DISTINCT e.id, e.tenant_id, e.name, e.url, e.secret, e.enabled, e.timeout_ms, e.max_retries, e.created_at, e.updated_at
		FROM endpoints e
		JOIN subscriptions s ON s.endpoint_id = e.id
		WHERE e.tenant_id = $1
		  AND e.enabled = true
		  AND (
		        '*' = ANY(s.event_types)
		     OR $2 = ANY(s.event_types)
		  )`,
		tenantID, eventType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []Endpoint
	for rows.Next() {
		var ep Endpoint
		if err := rows.Scan(&ep.ID, &ep.TenantID, &ep.Name, &ep.URL, &ep.Secret, &ep.Enabled,
			&ep.TimeoutMs, &ep.MaxRetries, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints, rows.Err()
}
