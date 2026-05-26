-- Восстановление читаемых ФИО демо-пользователей (если при seed сломалась кодировка и в UI были «????»).
UPDATE users SET full_name = 'Иван Инициатор', position = 'Аналитик' WHERE login = 'initiator';
UPDATE users SET full_name = 'Пётр Согласующий', position = 'Руководитель' WHERE login = 'approver';
UPDATE users SET full_name = 'Сергей ИБ', position = 'ИБ' WHERE login = 'security';
UPDATE users SET full_name = 'Анна Админ', position = 'Администратор' WHERE login = 'admin';
