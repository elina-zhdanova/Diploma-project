package dbschema

import (
	"context"
	_ "embed"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed embedded/000006_expand_catalog.up.sql
var expandCatalogSQL string

//go:embed embedded/000007_repair_utf8_catalog.up.sql
var repairUtf8CatalogSQL string

//go:embed embedded/000008_access_role_approvers.up.sql
var accessRoleApproversSQL string

// Ensure применяет идемпотентные правки схемы для БД, созданных до появления
// миграций 000002+ (том Postgres не пересоздаётся — initdb-скрипты не выполняются повторно).
func Ensure(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255)`,
		`ALTER TABLE requests ADD COLUMN IF NOT EXISTS justification TEXT`,
		`ALTER TABLE requests ADD COLUMN IF NOT EXISTS needed_until DATE`,
		`ALTER TABLE request_items ADD COLUMN IF NOT EXISTS justification TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE delegations ADD COLUMN IF NOT EXISTS permanent BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE delegations ALTER COLUMN start_date DROP NOT NULL`,
		`ALTER TABLE delegations ALTER COLUMN end_date DROP NOT NULL`,
		`CREATE TABLE IF NOT EXISTS delegation_scopes (
			delegation_id UUID NOT NULL REFERENCES delegations(id) ON DELETE CASCADE,
			access_role_id UUID NOT NULL REFERENCES access_roles(id) ON DELETE CASCADE,
			PRIMARY KEY (delegation_id, access_role_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_delegation_scopes_role ON delegation_scopes(access_role_id)`,
		`CREATE TABLE IF NOT EXISTS attachments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id),
			request_id UUID REFERENCES requests(id) ON DELETE SET NULL,
			original_name VARCHAR(512) NOT NULL,
			content_type VARCHAR(255) NOT NULL DEFAULT 'application/octet-stream',
			s3_key TEXT NOT NULL UNIQUE,
			size_bytes BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_user ON attachments(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_request ON attachments(request_id)`,
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s); err != nil {
			return err
		}
	}
	if err := ensureDemoSeed(ctx, pool); err != nil {
		return err
	}
	if err := ensureBulkTestRoles(ctx, pool); err != nil {
		return err
	}
	if err := ensureFixDemoUserDisplayNames(ctx, pool); err != nil {
		return err
	}
	if err := ensureExpandCatalog(ctx, pool); err != nil {
		return err
	}
	if err := ensureRepairUtf8Catalog(ctx, pool); err != nil {
		return err
	}
	return ensureAccessRoleApprovers(ctx, pool)
}

// ensureAccessRoleApprovers — таблица цепочек по ролям и backfill для ролей без согласующих.
func ensureAccessRoleApprovers(ctx context.Context, pool *pgxpool.Pool) error {
	for _, stmt := range splitSQLStatementsNoComments(accessRoleApproversSQL) {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// ensureRepairUtf8Catalog восстанавливает русские подписи справочников (см. embedded/000007_*.sql).
func ensureRepairUtf8Catalog(ctx context.Context, pool *pgxpool.Pool) error {
	for _, stmt := range splitSQLStatementsNoComments(repairUtf8CatalogSQL) {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// ensureExpandCatalog подмешивает Jira / Confluence / 1С / GitLab / SAP (см. embedded/000006_*.sql).
func ensureExpandCatalog(ctx context.Context, pool *pgxpool.Pool) error {
	for _, stmt := range splitSQLStatementsNoComments(expandCatalogSQL) {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func splitSQLStatementsNoComments(sql string) []string {
	lines := strings.Split(sql, "\n")
	var sb strings.Builder
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "--") {
			continue
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	body := strings.TrimSpace(sb.String())
	parts := strings.Split(body, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p+";")
	}
	return out
}

// ensureDemoSeed вставляет демо-данные из 000003, если таблица users пуста (том без seed).
func ensureDemoSeed(ctx context.Context, pool *pgxpool.Pool) error {
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM users`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	seed := []string{
		`INSERT INTO departments (id, name) VALUES
			('a0000001-0000-0000-0000-000000000001'::uuid, 'Департамент ИТ')
		ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO rbac_roles (id, name) VALUES
			('b0000001-0000-0000-0000-000000000001'::uuid, 'initiator'),
			('b0000002-0000-0000-0000-000000000002'::uuid, 'approver'),
			('b0000003-0000-0000-0000-000000000003'::uuid, 'security'),
			('b0000004-0000-0000-0000-000000000004'::uuid, 'admin')
		ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO users (id, login, full_name, email, department_id, position, password_hash) VALUES
			('c0000001-0000-0000-0000-000000000001'::uuid, 'initiator', 'Иван Инициатор', 'initiator@example.test', 'a0000001-0000-0000-0000-000000000001'::uuid, 'Аналитик',
			 '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'),
			('c0000002-0000-0000-0000-000000000002'::uuid, 'approver', 'Пётр Согласующий', 'approver@example.test', 'a0000001-0000-0000-0000-000000000001'::uuid, 'Руководитель',
			 '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'),
			('c0000003-0000-0000-0000-000000000003'::uuid, 'security', 'Сергей ИБ', 'security@example.test', 'a0000001-0000-0000-0000-000000000001'::uuid, 'ИБ',
			 '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'),
			('c0000004-0000-0000-0000-000000000004'::uuid, 'admin', 'Анна Админ', 'admin@example.test', 'a0000001-0000-0000-0000-000000000001'::uuid, 'Администратор',
			 '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi')
		ON CONFLICT (login) DO NOTHING`,
		`INSERT INTO user_roles (user_id, rbac_role_id) VALUES
			('c0000001-0000-0000-0000-000000000001'::uuid, 'b0000001-0000-0000-0000-000000000001'::uuid),
			('c0000002-0000-0000-0000-000000000002'::uuid, 'b0000002-0000-0000-0000-000000000002'::uuid),
			('c0000003-0000-0000-0000-000000000003'::uuid, 'b0000003-0000-0000-0000-000000000003'::uuid),
			('c0000004-0000-0000-0000-000000000004'::uuid, 'b0000004-0000-0000-0000-000000000004'::uuid)
		ON CONFLICT DO NOTHING`,
		`INSERT INTO systems (id, name, description) VALUES
			('d0000001-0000-0000-0000-000000000001'::uuid, 'Корпоративный портал', 'SSO и заявки'),
			('d0000002-0000-0000-0000-000000000002'::uuid, 'CRM', 'Учёт клиентов')
		ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO resources (id, system_id, name, description, owner_id, sensitive) VALUES
			('e0000001-0000-0000-0000-000000000001'::uuid, 'd0000001-0000-0000-0000-000000000001'::uuid, 'Портал / отчёты', 'Доступ к отчётам', 'c0000002-0000-0000-0000-000000000002'::uuid, false),
			('e0000002-0000-0000-0000-000000000002'::uuid, 'd0000002-0000-0000-0000-000000000002'::uuid, 'CRM / сделки', 'Конфиденциальные сделки', 'c0000002-0000-0000-0000-000000000002'::uuid, true)
		ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO access_roles (id, resource_id, name, description, risk_level) VALUES
			('f0000001-0000-0000-0000-000000000001'::uuid, 'e0000001-0000-0000-0000-000000000001'::uuid, 'Читатель отчётов', 'Просмотр', 'low'),
			('f0000002-0000-0000-0000-000000000002'::uuid, 'e0000002-0000-0000-0000-000000000002'::uuid, 'Редактор сделок', 'Изменение', 'high')
		ON CONFLICT (id) DO NOTHING`,
	}
	for _, s := range seed {
		if _, err := tx.Exec(ctx, s); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ensureBulkTestRoles — 500 тестовых ролей (см. migrations/000004), если их ещё нет.
func ensureBulkTestRoles(ctx context.Context, pool *pgxpool.Pool) error {
	const q = `INSERT INTO access_roles (resource_id, name, description, risk_level)
SELECT
    'e0000001-0000-0000-0000-000000000001'::uuid,
    format('Каталог-тест %s', lpad(gs::text, 4, '0')),
    'Сгенерированная роль для проверки поиска и заказа доступа',
    (ARRAY['low', 'medium', 'high'])[1 + ((gs - 1) % 3)]
FROM generate_series(1, 500) AS gs
WHERE NOT EXISTS (
    SELECT 1
    FROM access_roles ar
    WHERE ar.name = format('Каталог-тест %s', lpad(gs::text, 4, '0'))
)`
	_, err := pool.Exec(ctx, q)
	return err
}

// ensureFixDemoUserDisplayNames восстанавливает читаемые ФИО демо-пользователей (UTF-8),
// если при первичном seed сломалась кодировка и в UI отображались «????».
func ensureFixDemoUserDisplayNames(ctx context.Context, pool *pgxpool.Pool) error {
	fixes := []struct {
		login    string
		fullName string
		position string
	}{
		{"initiator", "Иван Инициатор", "Аналитик"},
		{"approver", "Пётр Согласующий", "Руководитель"},
		{"security", "Сергей ИБ", "ИБ"},
		{"admin", "Анна Админ", "Администратор"},
	}
	for _, f := range fixes {
		if _, err := pool.Exec(ctx, `UPDATE users SET full_name = $1, position = $2 WHERE login = $3`, f.fullName, f.position, f.login); err != nil {
			return err
		}
	}
	return nil
}
