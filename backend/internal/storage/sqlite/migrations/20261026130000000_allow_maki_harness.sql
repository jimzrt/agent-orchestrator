-- Add Maki to the sessions harness constraint without changing usage bindings.
-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
PRAGMA writable_schema = ON;
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE sqlite_master SET sql = replace(sql, '''fake''))', '''maki'', ''fake''))')
WHERE type = 'table' AND name = 'sessions' AND sql LIKE '%CHECK (harness IN (%' AND sql LIKE '%''fake''))%' AND sql NOT LIKE '%''maki''%';
-- +goose StatementEnd
-- +goose StatementBegin
PRAGMA writable_schema = RESET;
-- +goose StatementEnd

-- +goose Down
-- Refuse a rollback while Maki rows would violate the restored CHECK.
-- +goose StatementBegin
CREATE TEMP TABLE IF NOT EXISTS maki_schema_guard (valid INTEGER NOT NULL CHECK (valid = 1));
-- +goose StatementEnd
-- +goose StatementBegin
INSERT INTO maki_schema_guard (valid)
SELECT CASE WHEN EXISTS (SELECT 1 FROM sessions WHERE harness = 'maki') THEN 0 ELSE 1 END;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE maki_schema_guard;
-- +goose StatementEnd
-- +goose StatementBegin
PRAGMA writable_schema = ON;
-- +goose StatementEnd
-- +goose StatementBegin
UPDATE sqlite_master SET sql = replace(sql, '''maki'', ', '')
WHERE type = 'table' AND name = 'sessions' AND sql LIKE '%CHECK (harness IN (%';
-- +goose StatementEnd
-- +goose StatementBegin
PRAGMA writable_schema = RESET;
-- +goose StatementEnd
