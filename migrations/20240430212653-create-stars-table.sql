
-- +migrate Up
CREATE TABLE IF NOT EXISTS "stars"
(
    "file_id"       UUID NOT NULL,
    "user_id"       UUID NOT NULL,
    "created_at"    TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY ("file_id", "user_id")
    );

-- +migrate Down
DROP TABLE "stars";
