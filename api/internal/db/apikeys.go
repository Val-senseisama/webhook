package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func (d *DB) GetAPIKeyByHash(ctx context.Context, keyHash string) (APIKey, error) {
	var k APIKey
	row := d.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key_hash, created_at FROM api_keys WHERE key_hash = $1`,
		keyHash)
	err := row.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.CreatedAt)
	return k, err
}

type CreateAPIKeyParams struct {
	TenantID uuid.UUID
	Name     string
	KeyHash  string
}

func (d *DB) CreateAPIKey(ctx context.Context, p CreateAPIKeyParams) (APIKey, error) {
	var k APIKey
	row := d.pool.QueryRow(ctx,
		`INSERT INTO api_keys (tenant_id, name, key_hash) VALUES ($1,$2,$3)
		 RETURNING id, tenant_id, name, key_hash, created_at`,
		p.TenantID, p.Name, p.KeyHash)
	err := row.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.CreatedAt)
	return k, err
}

func (d *DB) ListAPIKeys(ctx context.Context, tenantID uuid.UUID) ([]APIKey, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT id, tenant_id, name, key_hash, created_at
		 FROM api_keys WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (d *DB) GetAPIKey(ctx context.Context, id, tenantID uuid.UUID) (APIKey, error) {
	var k APIKey
	row := d.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key_hash, created_at
		 FROM api_keys WHERE id = $1 AND tenant_id = $2`,
		id, tenantID)
	err := row.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.CreatedAt)
	return k, err
}

func (d *DB) DeleteAPIKey(ctx context.Context, id, tenantID uuid.UUID) error {
	tag, err := d.pool.Exec(ctx,
		`DELETE FROM api_keys WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ErrNotFound is returned when a DELETE or UPDATE matches no rows.
var ErrNotFound = &notFoundError{}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "not found" }

// IsNotFound reports whether err is an ErrNotFound.
func IsNotFound(err error) bool {
	_, ok := err.(*notFoundError)
	_ = time.Time{} // keep "time" import used
	return ok
}
