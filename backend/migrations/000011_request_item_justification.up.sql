-- Обоснование по каждой позиции заявки (роль доступа).
ALTER TABLE request_items
  ADD COLUMN IF NOT EXISTS justification TEXT NOT NULL DEFAULT '';
