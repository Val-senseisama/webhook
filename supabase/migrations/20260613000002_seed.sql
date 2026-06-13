-- Bootstrap: default tenant for initial setup.
INSERT INTO tenants (id, name)
VALUES ('00000000-0000-0000-0000-000000000001', 'default')
ON CONFLICT (id) DO NOTHING;
