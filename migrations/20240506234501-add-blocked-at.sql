
-- +migrate Up
ALTER TABLE "users"
    ADD COLUMN "blocked_at" TIMESTAMP NULL;

-- +migrate Down
ALTER TABLE "users"
    DROP COLUMN "blocked_at";
