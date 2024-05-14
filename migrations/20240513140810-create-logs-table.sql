
-- +migrate Up
CREATE TABLE IF NOT EXISTS "logs"
(
    "id"            INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    "user_id"       UUID NOT NULL,
    "file_id"       UUID NOT NULL,
    "action"        VARCHAR(255) NOT NULL, -- create, update, delete, open, move, share, star
    "created_at"    TIMESTAMPTZ DEFAULT NOW()
);

-- +migrate Down
DROP TABLE "logs";