-- 500 тестовых ролей для каталога (поиск, нагрузочный просмотр)
INSERT INTO access_roles (resource_id, name, description, risk_level)
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
);
