-- +goose Up
-- Bootstrap: insert the default tenant so the API can start.
-- The tenant ID matches DEFAULT_TENANT_ID in .env.
INSERT INTO tenants (id, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'default')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM tenants WHERE id = '00000000-0000-0000-0000-000000000001';
