
-- +migrate Up
CREATE EXTENSION pg_trgm;

-- +migrate Down
DROP EXTENSION pg_trgm;