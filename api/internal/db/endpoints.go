package db

import (
	"context"

	"github.com/google/uuid"
)

type CreateEndpointParams struct {
	TenantID   uuid.UUID
	Name       string
	URL        string
	Secret     string
	TimeoutMs  int
	MaxRetries int
}

func (d *DB) CreateEndpoint(ctx context.Context, p CreateEndpointParams) (Endpoint, error) {
	var e Endpoint
	row := d.pool.QueryRow(ctx, `
		INSERT INTO endpoints (tenant_id, name, url, secret, timeout_ms, max_retries)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, name, url, secret, enabled, timeout_ms, max_retries, created_at, updated_at`,
		p.TenantID, p.Name, p.URL, p.Secret, p.TimeoutMs, p.MaxRetries,
	)
	err := row.Scan(&e.ID, &e.TenantID, &e.Name, &e.URL, &e.Secret, &e.Enabled,
		&e.TimeoutMs, &e.MaxRetries, &e.CreatedAt, &e.UpdatedAt)
	return e, err
}

func (d *DB) GetEndpoint(ctx context.Context, id, tenantID uuid.UUID) (Endpoint, error) {
	var e Endpoint
	row := d.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, url, secret, enabled, timeout_ms, max_retries, created_at, updated_at
		FROM endpoints WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	err := row.Scan(&e.ID, &e.TenantID, &e.Name, &e.URL, &e.Secret, &e.Enabled,
		&e.TimeoutMs, &e.MaxRetries, &e.CreatedAt, &e.UpdatedAt)
	return e, err
}

func (d *DB) ListEndpoints(ctx context.Context, tenantID uuid.UUID) ([]Endpoint, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, tenant_id, name, url, secret, enabled, timeout_ms, max_retries, created_at, updated_at
		FROM endpoints WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []Endpoint
	for rows.Next() {
		var e Endpoint
		if err := rows.Scan(&e.ID, &e.TenantID, &e.Name, &e.URL, &e.Secret, &e.Enabled,
			&e.TimeoutMs, &e.MaxRetries, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, e)
	}
	return endpoints, rows.Err()
}

type UpdateEndpointParams struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	Name       string
	URL        string
	Enabled    bool
	TimeoutMs  int
	MaxRetries int
}

func (d *DB) UpdateEndpoint(ctx context.Context, p UpdateEndpointParams) (Endpoint, error) {
	var e Endpoint
	row := d.pool.QueryRow(ctx, `
		UPDATE endpoints SET
			name        = $3,
			url         = $4,
			enabled     = $5,
			timeout_ms  = $6,
			max_retries = $7,
			updated_at  = now()
		WHERE id = $1 AND tenant_id = $2
		RETURNING id, tenant_id, name, url, secret, enabled, timeout_ms, max_retries, created_at, updated_at`,
		p.ID, p.TenantID, p.Name, p.URL, p.Enabled, p.TimeoutMs, p.MaxRetries,
	)
	err := row.Scan(&e.ID, &e.TenantID, &e.Name, &e.URL, &e.Secret, &e.Enabled,
		&e.TimeoutMs, &e.MaxRetries, &e.CreatedAt, &e.UpdatedAt)
	return e, err
}

func (d *DB) DeleteEndpoint(ctx context.Context, id, tenantID uuid.UUID) error {
	_, err := d.pool.Exec(ctx,
		`DELETE FROM endpoints WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (d *DB) RotateEndpointSecret(ctx context.Context, id, tenantID uuid.UUID, newSecret string) error {
	_, err := d.pool.Exec(ctx,
		`UPDATE endpoints SET secret = $3, updated_at = now() WHERE id = $1 AND tenant_id = $2`,
		id, tenantID, newSecret)
	return err
}
