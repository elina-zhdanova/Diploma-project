-- Опциональная дата, до которой инициатору нужен доступ (включительно).

ALTER TABLE requests ADD COLUMN IF NOT EXISTS needed_until DATE;
