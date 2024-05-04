
-- +migrate Up
ALTER TABLE files ADD COLUMN thumbnail TEXT NULL;

-- +migrate Down
ALTER TABLE files DROP COLUMN thumbnail;