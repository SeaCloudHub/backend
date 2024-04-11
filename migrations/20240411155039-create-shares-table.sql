
-- +migrate Up
CREATE TABLE IF NOT EXISTS "shares"
(
    "file_id"       UUID NOT NULL,
    "user_id"       UUID NOT NULL,
    "created_at"    TIMESTAMPTZ DEFAULT NOW()
);

-- +migrate Down
DROP TABLE "shares";