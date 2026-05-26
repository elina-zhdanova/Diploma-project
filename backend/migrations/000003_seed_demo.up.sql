-- Демо-данные: подразделение, RBAC, пользователи (пароль: password), каталог, владельцы ресурсов
-- bcrypt "password" (совместим с golang.org/x/crypto/bcrypt)
-- $2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi

INSERT INTO departments (id, name) VALUES
    ('a0000001-0000-0000-0000-000000000001'::uuid, 'Департамент ИТ')
ON CONFLICT (id) DO NOTHING;

INSERT INTO rbac_roles (id, name) VALUES
    ('b0000001-0000-0000-0000-000000000001'::uuid, 'initiator'),
    ('b0000002-0000-0000-0000-000000000002'::uuid, 'approver'),
    ('b0000003-0000-0000-0000-000000000003'::uuid, 'security'),
    ('b0000004-0000-0000-0000-000000000004'::uuid, 'admin')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, login, full_name, email, department_id, position, password_hash) VALUES
    ('c0000001-0000-0000-0000-000000000001'::uuid, 'initiator', 'Иван Инициатор', 'initiator@example.test', 'a0000001-0000-0000-0000-000000000001'::uuid, 'Аналитик',
     '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'),
    ('c0000002-0000-0000-0000-000000000002'::uuid, 'approver', 'Пётр Согласующий', 'approver@example.test', 'a0000001-0000-0000-0000-000000000001'::uuid, 'Руководитель',
     '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'),
    ('c0000003-0000-0000-0000-000000000003'::uuid, 'security', 'Сергей ИБ', 'security@example.test', 'a0000001-0000-0000-0000-000000000001'::uuid, 'ИБ',
     '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'),
    ('c0000004-0000-0000-0000-000000000004'::uuid, 'admin', 'Анна Админ', 'admin@example.test', 'a0000001-0000-0000-0000-000000000001'::uuid, 'Администратор',
     '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi')
ON CONFLICT (login) DO NOTHING;

INSERT INTO user_roles (user_id, rbac_role_id) VALUES
    ('c0000001-0000-0000-0000-000000000001'::uuid, 'b0000001-0000-0000-0000-000000000001'::uuid),
    ('c0000002-0000-0000-0000-000000000002'::uuid, 'b0000002-0000-0000-0000-000000000002'::uuid),
    ('c0000003-0000-0000-0000-000000000003'::uuid, 'b0000003-0000-0000-0000-000000000003'::uuid),
    ('c0000004-0000-0000-0000-000000000004'::uuid, 'b0000004-0000-0000-0000-000000000004'::uuid)
ON CONFLICT DO NOTHING;

INSERT INTO systems (id, name, description) VALUES
    ('d0000001-0000-0000-0000-000000000001'::uuid, 'Корпоративный портал', 'SSO и заявки'),
    ('d0000002-0000-0000-0000-000000000002'::uuid, 'CRM', 'Учёт клиентов')
ON CONFLICT (id) DO NOTHING;

INSERT INTO resources (id, system_id, name, description, owner_id, sensitive) VALUES
    ('e0000001-0000-0000-0000-000000000001'::uuid, 'd0000001-0000-0000-0000-000000000001'::uuid, 'Портал / отчёты', 'Доступ к отчётам', 'c0000002-0000-0000-0000-000000000002'::uuid, false),
    ('e0000002-0000-0000-0000-000000000002'::uuid, 'd0000002-0000-0000-0000-000000000002'::uuid, 'CRM / сделки', 'Конфиденциальные сделки', 'c0000002-0000-0000-0000-000000000002'::uuid, true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO access_roles (id, resource_id, name, description, risk_level) VALUES
    ('f0000001-0000-0000-0000-000000000001'::uuid, 'e0000001-0000-0000-0000-000000000001'::uuid, 'Читатель отчётов', 'Просмотр', 'low'),
    ('f0000002-0000-0000-0000-000000000002'::uuid, 'e0000002-0000-0000-0000-000000000002'::uuid, 'Редактор сделок', 'Изменение', 'high')
ON CONFLICT (id) DO NOTHING;
