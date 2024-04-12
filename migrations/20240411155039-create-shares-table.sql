
-- +migrate Up
CREATE TABLE IF NOT EXISTS "shares"
(
    "file_id"       UUID NOT NULL,
    "user_id"       UUID NOT NULL,
    "role"          VARCHAR(255) NOT NULL, -- editor, viewer
    "created_at"    TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY ("file_id", "user_id")
);

-- +migrate Down
DROP TABLE "shares";