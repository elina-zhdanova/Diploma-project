-- Делегирование: бессрочно / по периоду; ограничение по ролям доступа из каталога.

ALTER TABLE delegations ADD COLUMN IF NOT EXISTS permanent BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE delegations ALTER COLUMN start_date DROP NOT NULL;
ALTER TABLE delegations ALTER COLUMN end_date DROP NOT NULL;

CREATE TABLE IF NOT EXISTS delegation_scopes (
    delegation_id UUID NOT NULL REFERENCES delegations(id) ON DELETE CASCADE,
    access_role_id UUID NOT NULL REFERENCES access_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (delegation_id, access_role_id)
);

CREATE INDEX IF NOT EXISTS idx_delegation_scopes_role ON delegation_scopes(access_role_id);
