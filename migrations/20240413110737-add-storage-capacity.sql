
-- +migrate Up
ALTER TABLE "users"
    ADD COLUMN "storage_usage" BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN "storage_capacity" BIGINT NOT NULL DEFAULT 10737418240; -- 10GB

-- +migrate Down
ALTER TABLE "users"
    DROP COLUMN "storage_usage",
    DROP COLUMN "storage_capacity";